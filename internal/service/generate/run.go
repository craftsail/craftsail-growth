// SPDX-License-Identifier: AGPL-3.0-or-later

package generate

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/langx"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
)

type Service struct {
	projects    *project.Service
	facts       *repo.Facts
	questions   *repo.Questions
	competitors *repo.Competitors
	assets      *repo.Assets
	audit       *audit.Service
	Asker       *sample.Asker
}

func New(db *gorm.DB) *Service {
	return &Service{
		projects: project.New(db), facts: &repo.Facts{DB: db}, questions: &repo.Questions{DB: db},
		competitors: &repo.Competitors{DB: db}, assets: &repo.Assets{DB: db}, audit: audit.New(db),
		Asker: sample.NewAsker(),
	}
}

type Result struct {
	Slug   string   `json:"slug"`
	Assets []string `json:"assets"`
}

// Run generates fix snippets from the brand facts: llms.txt, JSON-LD and
// definition / FAQ HTML blocks. They back the fixes for NO_LLMS_TXT and
// NO_JSONLD and must be reviewed before publishing.
func (s *Service) Run(ctx context.Context, slug string, which []string) (*Result, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	if len(which) == 0 {
		which = []string{"llms", "jsonld", "snippets"}
	}
	hasSite := !p.NoSite && p.Site != ""
	if !hasSite {
		var keep []string
		for _, w := range which {
			if w != "llms" && w != "jsonld" {
				keep = append(keep, w)
			}
		}
		which = keep
	}
	md := ""
	if f, _ := s.facts.Get(ctx, p.ID); f != nil {
		md = f.Markdown
	}
	facts := ParseFacts(md)
	qs, _ := s.questions.List(ctx, p.ID)
	var pages []model.AuditPage
	if rep, _ := s.audit.Latest(ctx, slug); rep != nil {
		pages = rep.Pages
	}
	var made []string
	put := func(path, kind, body string) error {
		made = append(made, path)
		return s.assets.Upsert(ctx, &model.Asset{ProjectID: p.ID, Path: path, Kind: kind, Content: body})
	}
	want := map[string]bool{}
	for _, w := range which {
		want[w] = true
	}
	// Snippets are written in the language of the brand facts, which
	// bootstrap drafts from the site text.
	lang := langx.Detect(md)
	if want["llms"] {
		if err := put("llms.txt", "llms", LLMS(p, facts, pages, lang)); err != nil {
			return nil, err
		}
	}
	if want["jsonld"] {
		for name, obj := range JSONLD(p, facts, qs) {
			if err := put("jsonld/"+name+".json", "jsonld", PrettyJSON(obj)); err != nil {
				return nil, err
			}
		}
	}
	if want["snippets"] {
		if err := put("snippets/definition.html", "snippet", DefinitionBlock(p, facts, lang)); err != nil {
			return nil, err
		}
		if err := put("snippets/faq.html", "snippet", FAQBlock(qs, lang)); err != nil {
			return nil, err
		}
	}
	if len(made) == 0 {
		return nil, fmt.Errorf("nothing to generate (projects without a site skip llms.txt and JSON-LD)")
	}
	return &Result{Slug: slug, Assets: made}, nil
}
