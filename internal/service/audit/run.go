// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/htmlx"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

type Service struct {
	projects  *project.Service
	pages     *repo.Pages
	sites     *repo.SiteSignals
	audits    *repo.Audits
	questions *repo.Questions
}

func New(db *gorm.DB) *Service {
	return &Service{
		projects:  project.New(db),
		pages:     &repo.Pages{DB: db},
		sites:     &repo.SiteSignals{DB: db},
		audits:    &repo.Audits{DB: db},
		questions: &repo.Questions{DB: db},
	}
}

type Report struct {
	model.Audit
	Pages     []model.AuditPage `json:"pages"`
	Findings  []IssueRow        `json:"findings,omitempty"`
	Formula   string            `json:"formula"`
	CrawlView CrawlSummary      `json:"crawl_view"`
}

func (s *Service) Latest(ctx context.Context, slug string) (*Report, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	a, pages, err := s.audits.Latest(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, nil
	}
	rows, err := s.audits.LatestIssues(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	findings := make([]IssueRow, 0, len(rows))
	for _, r := range rows {
		findings = append(findings, IssueRow{Finding: Finding{Code: r.Code, URL: r.URL, Detail: r.Detail}, Severity: r.Severity, Layer: r.Layer, Blocked: r.Blocked})
	}
	return &Report{Audit: *a, Pages: pages, Findings: findings, Formula: FormulaMarkdown, CrawlView: s.crawlView(ctx, p.ID, a.AvgScore, pages)}, nil
}

func (s *Service) Run(ctx context.Context, slug string) (*Report, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	pages, err := s.pages.List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 && (p.NoSite || strings.TrimSpace(p.Site) == "") {
		a := &model.Audit{
			ProjectID: p.ID, RunAt: time.Now().Unix(), NoSite: true, Market: model.MarketAll,
			GradeDistribution: map[string]int{}, Site: map[string]any{},
			Layers: []map[string]any{}, BlockGap: []map[string]any{},
			LanguageCoverage: map[string]any{},
		}
		if err := s.audits.SaveRun(ctx, a, nil); err != nil {
			return nil, err
		}
		return &Report{Audit: *a, Formula: FormulaMarkdown, CrawlView: s.crawlView(ctx, p.ID, a.AvgScore, nil)}, nil
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no crawl results; run craftsail-growth crawl --slug %s first", slug)
	}
	sig, err := s.sites.Get(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	site := map[string]any{}
	if sig != nil && sig.Payload != nil {
		site = sig.Payload
	}

	qs, _ := s.questions.List(ctx, p.ID)
	qany := make([]any, 0, len(qs))
	for _, q := range qs {
		qany = append(qany, map[string]any{"text": q.Text})
	}
	cfg := map[string]any{
		"brand": map[string]any{
			"name": p.Name, "aliases": toAny(p.Brand.Aliases), "products": toAny(p.Brand.Products),
		},
		"questions": qany,
	}
	kws := KeywordsFromConfig(cfg)

	var docs []crawl.PageDoc
	var results []PageScore
	for _, pg := range pages {
		d := crawl.FromModel(pg)
		docs = append(docs, d)
		results = append(results, ScorePage(d, kws))
	}

	var okScores []float64
	for i, r := range results {
		if docs[i].Status == 200 {
			okScores = append(okScores, r.Score)
		}
	}
	avg := 0.0
	if len(okScores) > 0 {
		sum := 0.0
		for _, v := range okScores {
			sum += v
		}
		avg = float64(int(sum/float64(len(okScores))*10+0.5)) / 10
	}

	langDist := map[string]int{}
	for _, d := range docs {
		if d.WordCount >= 120 {
			lang := d.Language
			if d.Text != "" {
				lang = htmlx.PageLanguage(d.Text, d.Lang)
			}
			langDist[lang]++
		}
	}
	en, zh, ja := langDist["en"], langDist["zh"], langDist["ja"]
	contentPages := 0
	for _, n := range langDist {
		contentPages += n
	}
	hreflangPages := 0
	for _, d := range docs {
		if d.WordCount >= 120 && d.HreflangCount > 0 {
			hreflangPages++
		}
	}
	nlang := 0
	for _, v := range []int{zh, en, ja} {
		if v > 0 {
			nlang++
		}
	}
	multilingual := nlang >= 2

	var dupBodies [][]string
	{
		byBody := map[string][]string{}
		spaceRe := regexp.MustCompile(`\s+`)
		for _, d := range docs {
			if d.Status != 200 || d.WordCount < 120 {
				continue
			}
			body := spaceRe.ReplaceAllString(d.Text, "")
			if len([]rune(body)) > 600 {
				body = string([]rune(body)[:600])
			}
			sum := md5.Sum([]byte(body))
			k := hex.EncodeToString(sum[:])
			byBody[k] = append(byBody[k], d.URL)
		}
		for _, us := range byBody {
			if len(us) > 1 {
				dupBodies = append(dupBodies, us)
			}
		}
	}
	noisy := 0
	switch v := site["sitemap_noisy_urls"].(type) {
	case int:
		noisy = v
	case float64:
		noisy = int(v)
	}
	langStats := LangStats{
		ZH: zh, EN: en, Total: contentPages, Multilingual: multilingual,
		HreflangPages: hreflangPages, LowValueURLs: noisy, DupBodyGroups: dupBodies,
	}

	gradeDist := map[string]int{"A": 0, "B": 0, "C": 0, "D": 0}
	gap := map[string]int{}
	for _, r := range results {
		gradeDist[r.Grade]++
		for k, v := range r.Blocks {
			if !v && k != "FAQ" { // FAQ is displayed, not scored
				gap[k]++
			}
		}
	}
	var blockGap []map[string]any
	for k, v := range gap {
		blockGap = append(blockGap, map[string]any{"block": k, "missing_pages": v, "total": len(results)})
	}
	// sort by missing desc
	for i := 0; i < len(blockGap); i++ {
		for j := i + 1; j < len(blockGap); j++ {
			if asInt(blockGap[j]["missing_pages"]) > asInt(blockGap[i]["missing_pages"]) {
				blockGap[i], blockGap[j] = blockGap[j], blockGap[i]
			}
		}
	}

	// Collect every finding: page scores, page SEO, cross-page SEO, site signals.
	var findings []Finding
	pageIdx := map[string]int{}
	for i := range results {
		pageIdx[normURL(docs[i].URL)] = i
		for _, c := range results[i].IssueCodes {
			findings = append(findings, Finding{Code: c, URL: docs[i].URL})
		}
	}
	extra := append([]Finding{}, SiteSEO(docs)...)
	for i := range docs {
		extra = append(extra, PageSEO(docs[i])...)
	}
	extra = append(extra, SiteFindings(site, langStats)...)
	findings = append(findings, extra...)
	for _, f := range extra {
		if f.URL == "" {
			continue
		}
		// keep the legacy per-page columns in step with the new rules
		if i, ok := pageIdx[normURL(f.URL)]; ok && !hasString(results[i].IssueCodes, f.Code) {
			results[i].IssueCodes = append(results[i].IssueCodes, f.Code)
		}
	}
	issueRows := MarkBlocked(findings)
	layers := buildLayers(findings)

	var apages []model.AuditPage
	for i, r := range results {
		blk := map[string]any{}
		for k, v := range r.Blocks {
			blk[k] = v
		}
		dims := map[string]any{}
		for k, v := range r.Dimensions {
			dims[k] = v
		}
		pid := pages[i].ID
		apages = append(apages, model.AuditPage{
			PageID: pid, Score: r.Score, Grade: r.Grade, Dimensions: dims,
			IssueCodes: r.IssueCodes, Blocks: blk,
			URL: r.URL, Title: r.Title, WordCount: r.WordCount, JSONLDTypes: r.JSONLDTypes,
		})
	}

	a := &model.Audit{
		ProjectID: p.ID, RunAt: time.Now().Unix(), AvgScore: avg, PageCount: len(results),
		GradeDistribution: gradeDist, Site: site, Layers: layers, NoSite: false,
		BlockGap: blockGap,
		LanguageCoverage: map[string]any{
			"distribution": langDist, "zh_pages": zh, "en_pages": en, "ja_pages": ja,
			"content_pages": contentPages, "hreflang_pages": hreflangPages, "multilingual": multilingual,
		},
		KeywordsUsed: kws, Market: model.MarketAll,
	}
	issueModels := make([]model.AuditIssue, 0, len(issueRows))
	for _, r := range issueRows {
		issueModels = append(issueModels, model.AuditIssue{
			ProjectID: p.ID, URL: r.URL, Code: r.Code, Severity: r.Severity, Layer: r.Layer,
			Blocked: r.Blocked, Detail: r.Detail,
		})
	}
	if err := s.audits.SaveRunWithIssues(ctx, a, apages, issueModels); err != nil {
		return nil, err
	}
	return &Report{Audit: *a, Pages: apages, Findings: issueRows, Formula: FormulaMarkdown, CrawlView: s.crawlView(ctx, p.ID, a.AvgScore, apages)}, nil
}

func toAny(ss []string) []any {
	out := make([]any, 0, len(ss))
	for _, s := range ss {
		out = append(out, s)
	}
	return out
}

func asStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	}
	return 0
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

var layerText = map[string][2]string{
	LayerAccess:     {"Access", "Can crawlers fetch the content?"},
	LayerDiscover:   {"Discover", "Can crawlers find and identify every URL?"},
	LayerUnderstand: {"Understand", "Can machines tell what each page is about?"},
	LayerCite:       {"Cite", "Is there specific content worth citing?"},
}

// buildLayers summarizes findings per layer in fix order. A layer after the
// first failing one is marked blocked_by that layer.
func buildLayers(findings []Finding) []map[string]any {
	status := LayerStatus(findings)
	count := map[string]map[string]int{}
	for _, f := range findings {
		l := Lookup(f.Code).Layer
		if count[l] == nil {
			count[l] = map[string]int{}
		}
		count[l][f.Code]++
	}
	var out []map[string]any
	firstFail, firstFailKey := "", ""
	for _, l := range LayerOrder {
		var codes []string
		for c := range count[l] {
			codes = append(codes, c)
		}
		sort.Slice(codes, func(i, j int) bool {
			a, b := Lookup(codes[i]), Lookup(codes[j])
			if a.Severity != b.Severity {
				return sevRank(a.Severity) < sevRank(b.Severity)
			}
			return codes[i] < codes[j]
		})
		issues := []string{}
		items := []map[string]any{}
		for _, c := range codes {
			is := Lookup(c)
			issues = append(issues, fmt.Sprintf("[%s] %s (%d)", is.Severity, is.Title, count[l][c]))
			items = append(items, map[string]any{"code": c, "severity": is.Severity, "title": is.Title, "count": count[l][c]})
		}
		row := map[string]any{"key": l, "name": layerText[l][0], "question": layerText[l][1], "status": status[l], "issues": issues, "items": items}
		if firstFail != "" {
			row["blocked_by"] = firstFail
			row["blocked_by_key"] = firstFailKey
		}
		if status[l] == "fail" && firstFail == "" {
			firstFail = layerText[l][0]
			firstFailKey = l
		}
		out = append(out, row)
	}
	return out
}

func sevRank(s string) int {
	switch s {
	case SevCritical:
		return 0
	case SevWarning:
		return 1
	}
	return 2
}

func hasString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// IssueView is a stored finding joined with its registry entry.
type IssueView struct {
	model.AuditIssue
	Title    string   `json:"title"`
	Why      string   `json:"why"`
	Fix      string   `json:"fix"`
	Surface  string   `json:"surface"`
	Scope    string   `json:"scope"`
	Evidence string   `json:"evidence"`
	Refs     []string `json:"refs"`
}

type IssueFilter struct {
	Severity, Layer, Code, Surface string
}

// Issues returns the latest audit's findings with registry text, plus the
// references they cite.
func (s *Service) Issues(ctx context.Context, slug string, f IssueFilter) ([]IssueView, map[string]Reference, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.audits.LatestIssues(ctx, p.ID)
	if err != nil {
		return nil, nil, err
	}
	out := []IssueView{}
	refs := map[string]Reference{}
	for _, r := range rows {
		is := Lookup(r.Code)
		if (f.Severity != "" && r.Severity != f.Severity) || (f.Layer != "" && r.Layer != f.Layer) ||
			(f.Code != "" && r.Code != f.Code) || (f.Surface != "" && is.Surface != f.Surface) {
			continue
		}
		for _, k := range is.Refs {
			refs[k] = References[k]
		}
		out = append(out, IssueView{AuditIssue: r, Title: is.Title, Why: is.Why, Fix: is.Fix,
			Surface: is.Surface, Scope: is.Scope, Evidence: is.Evidence, Refs: is.Refs})
	}
	return out, refs, nil
}
