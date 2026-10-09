// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/langx"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

type LLM interface {
	AskJSON(ctx context.Context, prompt string) (map[string]any, error)
}

type Service struct {
	projects    *project.Service
	projRepo    *repo.Projects
	pages       *repo.Pages
	audits      *repo.Audits
	questions   *repo.Questions
	competitors *repo.Competitors
	facts       *repo.Facts
	LLM         LLM
}

func New(db *gorm.DB) *Service {
	return &Service{
		projects:    project.New(db),
		projRepo:    &repo.Projects{DB: db},
		pages:       &repo.Pages{DB: db},
		audits:      &repo.Audits{DB: db},
		questions:   &repo.Questions{DB: db},
		competitors: &repo.Competitors{DB: db},
		facts:       &repo.Facts{DB: db},
	}
}

type Result struct {
	Slug          string      `json:"slug"`
	SkipLLM       bool        `json:"skip_llm"`
	Questions     int         `json:"questions"`
	Competitors   int         `json:"competitors"`
	Uncertain     []string    `json:"uncertain"`
	NeedsReview   bool        `json:"needs_review"`
	FactsMarkdown string      `json:"facts_markdown"`
	Brand         model.Brand `json:"brand"`
}

func (s *Service) Run(ctx context.Context, slug string, skipLLM bool) (*Result, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	digest, err := s.digestFromDB(ctx, p)
	if err != nil {
		return nil, err
	}
	if digest == "" {
		return nil, fmt.Errorf("no crawl results or brand materials; run crawl or add materials first")
	}

	lang := p.SamplingLanguage
	if lang == "" {
		lang = langx.Detect(digest)
	}
	if skipLLM || s.LLM == nil {
		return s.runTemplates(ctx, p, lang)
	}

	brandMap, err := s.LLM.AskJSON(ctx, brandPrompt+digest)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || brandMap == nil {
		return s.runTemplates(ctx, p, lang)
	}
	facts := brandFactsFromMap(brandMap)
	ApplyBrand(p, facts)

	// Competitors and prompts both depend only on the brand facts, so ask
	// for them at the same time; each call can take a minute or more.
	var compMap, qMap map[string]any
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); compMap, _ = s.LLM.AskJSON(ctx, competitorPrompt(facts)) }()
	go func() {
		defer wg.Done()
		qMap, _ = s.LLM.AskJSON(ctx, questionPrompt(facts, lang)+"\nTarget audience region: "+p.TargetRegion+". This describes the audience, not simulated engine geolocation.")
	}()
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var comps []model.Competitor
	if compMap != nil {
		if rows, ok := compMap["competitors"].([]any); ok {
			comps = FilterCompetitors(asMaps(rows), p.Name)
		}
	}
	var qs []model.Question
	if qMap != nil {
		if rows, ok := qMap["questions"].([]any); ok {
			qs = NormalizeQuestions(asMaps(rows))
			for i := range qs {
				qs[i].Language = lang
			}
		}
	}
	if len(qs) == 0 {
		qs = TemplateQuestions(p.Name, lang)
	}
	return s.save(ctx, p, facts, comps, qs, sourceLLM)
}

func (s *Service) runTemplates(ctx context.Context, p *model.Project, lang string) (*Result, error) {
	facts := BrandFactsFromProject(p)
	if facts.Definition == "" {
		facts.Definition = "(TBD: start with the brand name, then the audience and the category)"
	}
	qs, err := s.questions.List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if len(qs) == 0 {
		qs = TemplateQuestions(p.Name, lang)
	}
	comps, err := s.competitors.List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	return s.save(ctx, p, facts, comps, qs, sourceTemplate)
}

func (s *Service) save(ctx context.Context, p *model.Project, facts BrandFacts, comps []model.Competitor, qs []model.Question, source string) (*Result, error) {
	ApplyBrand(p, facts)
	p.Bootstrap = &model.BootstrapMeta{
		At: time.Now().Format(time.RFC3339), Source: source,
		Uncertain: facts.Uncertain, NeedsReview: true,
	}
	if err := s.projRepo.Save(ctx, p); err != nil {
		return nil, err
	}
	if err := s.questions.Replace(ctx, p.ID, qs); err != nil {
		return nil, err
	}
	if err := s.competitors.Replace(ctx, p.ID, comps); err != nil {
		return nil, err
	}
	md := RenderFacts(facts, p.Site, comps)
	if err := s.facts.Upsert(ctx, p.ID, md); err != nil {
		return nil, err
	}
	return &Result{
		Slug: p.Slug, SkipLLM: source != sourceLLM,
		Questions: len(qs), Competitors: len(comps),
		Uncertain: facts.Uncertain, NeedsReview: true,
		FactsMarkdown: md, Brand: p.Brand,
	}, nil
}

func (s *Service) digestFromDB(ctx context.Context, p *model.Project) (string, error) {
	pages, err := s.pages.List(ctx, p.ID)
	if err != nil {
		return "", err
	}
	var pts []PageText
	for _, pg := range pages {
		pts = append(pts, PageText{URL: pg.URL, Title: pg.Title, Text: pg.Text})
	}
	scores := map[string]float64{}
	if a, ap, err := s.audits.Latest(ctx, p.ID); err == nil && a != nil {
		for _, x := range ap {
			scores[x.URL] = x.Score
		}
	}
	return Digest(pts, scores, p.Materials, 14000), nil
}

func (s *Service) Facts(ctx context.Context, slug string) (string, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	f, err := s.facts.Get(ctx, p.ID)
	if err != nil {
		return "", err
	}
	if f == nil {
		return "", nil
	}
	return f.Markdown, nil
}

func (s *Service) Questions(ctx context.Context, slug string) ([]model.Question, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	rows, err := s.questions.List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Tags = model.SanitizeTags(rows[i].Tags)
		rows[i].SystemTags = model.ComputeSystemTags(rows[i].Text, p.Name, p.Brand.Aliases, p.Site)
	}
	return rows, nil
}

func (s *Service) SaveQuestions(ctx context.Context, slug string, rows []model.Question) error {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	for i := range rows {
		if rows[i].Language != "" && rows[i].Language != "en" && rows[i].Language != "zh" && rows[i].Language != "pt" {
			return project.ErrInvalidLanguage
		}
		if rows[i].Intent == "" {
			rows[i].Intent = model.IntentOf(rows[i].GroupName)
		}
		if rows[i].QID == "" {
			rows[i].QID = "q" + pad3(i+1)
		}
		rows[i].Tags = model.SanitizeTags(rows[i].Tags)
	}
	return s.questions.Replace(ctx, p.ID, rows)
}

func (s *Service) Competitors(ctx context.Context, slug string) ([]model.Competitor, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.competitors.List(ctx, p.ID)
}

func (s *Service) SaveCompetitors(ctx context.Context, slug string, rows []model.Competitor) error {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	return s.competitors.Replace(ctx, p.ID, rows)
}

func asMaps(rows []any) []map[string]any {
	var out []map[string]any
	for _, r := range rows {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func brandFactsFromMap(m map[string]any) BrandFacts {
	f := BrandFacts{
		Name: asString(m["name"]), Industry: asString(m["industry"]),
		TargetUsers: asString(m["target_users"]), BusinessGoal: asString(m["business_goal"]),
		Definition: asString(m["definition"]),
		Aliases:    stringList(m["aliases"]), Products: stringList(m["products"]),
		Suitable: stringList(m["suitable"]), Unsuitable: stringList(m["unsuitable"]),
		Disambiguation: stringList(m["disambiguation"]), Uncertain: stringList(m["uncertain"]),
	}
	if rows, ok := m["key_numbers"].([]any); ok {
		for _, r := range rows {
			mm, _ := r.(map[string]any)
			f.KeyNumbers = append(f.KeyNumbers, model.KeyNumber{
				Fact: asString(mm["fact"]), Value: asString(mm["value"]), Source: asString(mm["source"]),
			})
		}
	}
	if rows, ok := m["pricing"].([]any); ok {
		for _, r := range rows {
			mm, _ := r.(map[string]any)
			f.Pricing = append(f.Pricing, model.Offer{
				Name: asString(mm["name"]), Price: asString(mm["price"]),
				Currency: asString(mm["currency"]), Desc: asString(mm["desc"]),
			})
		}
	}
	return f
}

func stringList(v any) []string {
	switch t := v.(type) {
	case []any:
		var out []string
		for _, x := range t {
			if s := asString(x); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	}
	return nil
}

const (
	sourceLLM      = "site text + LLM extraction"
	sourceTemplate = "templates (skip-llm)"
)

const brandPrompt = `You are a GEO (generative engine optimization) analyst. Below is text crawled from a product website.

Extract brand facts **from this text only** and output JSON. Hard rules:

- **Never add information that is not in the text**
- For any field the text does not state, use the value "TBD"
- Quote numbers exactly as written

Fields: name, aliases[], products[], industry, target_users, business_goal, definition (one sentence starting with the brand name),
disambiguation[], key_numbers[{fact, value, source}], pricing[{name, price, currency, desc}], suitable[], unsuitable[], uncertain[].
Write values in the language of the site text.

Site text:
`

func competitorPrompt(f BrandFacts) string {
	return fmt.Sprintf("List real competitors as JSON {\"competitors\": [{\"name\", \"site\", \"aliases\": []}]}. Product: %s. Category: %s. Positioning: %s.\nOnly list names that really exist. Never invent names.", f.Name, f.Industry, f.Definition)
}

func questionPrompt(f BrandFacts, lang string) string {
	language := "English"
	if lang == "zh" {
		language = "Chinese"
	}
	if lang == "pt" {
		language = "Portuguese"
	}
	return fmt.Sprintf("Design a prompt library for the brand %s (%s) as JSON {\"questions\": [{\"id\", \"group\", \"text\"}]}. "+
		"Use exactly these seven group keys: 推荐 (recommendation), 比较 (comparison), 替代 (alternatives), 价格 (pricing), 风险 (risks), 品牌验证 (brand check), 场景 (use case). "+
		"Most prompts must be unbranded buyer questions; only the 品牌验证 group may name the brand. "+
		"Name the product category the way a buyer would. Write every prompt in %s.", f.Name, f.Industry, language)
}

func (s *Service) Libraries(ctx context.Context, slug string) ([]model.QuestionLibrary, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.questions.Libraries(ctx, p.ID)
}
