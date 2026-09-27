// SPDX-License-Identifier: AGPL-3.0-or-later

// Package report renders the period report. Every number comes from the
// metrics package (through sample.Measure / PeriodMetrics), the audit issue
// registry, or the official Google daily tables, so the report matches the
// dashboard.
package report

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/verify"
)

type Service struct {
	db          *gorm.DB
	projects    *project.Service
	audit       *audit.Service
	sample      *sample.Service
	verify      *verify.Service
	opportunity *opportunity.Service
	tasks       *repo.Tasks
	reports     *repo.Reports
}

func New(db *gorm.DB) *Service {
	return &Service{
		db: db, projects: project.New(db), audit: audit.New(db), sample: sample.New(db, sample.NewAsker()),
		verify: verify.New(db), opportunity: opportunity.NewDB(db),
		tasks: &repo.Tasks{DB: db}, reports: &repo.Reports{DB: db},
	}
}

type Out struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	On       string `json:"on"`
}

func (s *Service) Build(ctx context.Context, slug string) (*Out, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	md, cards := s.markdown(ctx, p)
	h := Document(p.Name+" · AI visibility report", md, cards)
	on := time.Now()
	_ = s.reports.Upsert(ctx, &model.Report{ProjectID: p.ID, ReportOn: on, Markdown: md, HTML: h})
	return &Out{Markdown: md, HTML: h, On: on.Format("2006-01-02")}, nil
}

func (s *Service) Latest(ctx context.Context, slug string) (*model.Report, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.reports.Latest(ctx, p.ID)
}

func (s *Service) markdown(ctx context.Context, p *model.Project) (string, [][2]string) {
	var b strings.Builder
	var cards [][2]string
	today := time.Now().Format("2006-01-02")
	fmt.Fprintf(&b, "# %s · AI visibility report · %s\n\n", p.Name, today)
	fmt.Fprintf(&b, "- Site: %s\n- Window: last %d days\n\n", orDash(p.Site), metrics.DefaultWindowDays)

	s.writeVisibility(ctx, &b, p, &cards)
	s.writeEngines(ctx, &b, p)
	s.writeAudit(ctx, &b, p, &cards)
	s.writeOpportunities(ctx, &b, p)
	s.writeActions(ctx, &b, p)
	s.writeWebInsight(ctx, &b, p)
	s.writeCrawlSearch(ctx, &b, p)

	b.WriteString("## How to read this report\n\n")
	b.WriteString("- Visibility counts unbranded prompts only. API and web answers are never added together. The range is a 95% Wilson interval; below 30 answers the sample is marked small.\n")
	b.WriteString("- A change is only called up or down when the 95% Newcombe interval of the difference excludes zero.\n")
	b.WriteString("- Audit rules carry an evidence level. Observational rules and rules of thumb are advice, not failures.\n")
	b.WriteString("- Nothing here guarantees that an engine will cite a page. Changes between periods are observations, not proof of cause.\n")
	return b.String(), cards
}

func (s *Service) writeVisibility(ctx context.Context, b *strings.Builder, p *model.Project, cards *[][2]string) {
	b.WriteString("## AI visibility\n\n")
	wrote := false
	for _, access := range []string{"api", "web"} {
		cur, err := s.sample.Measure(ctx, p.Slug, sample.MeasureQuery{Range: "30d", Access: access})
		if err != nil || cur == nil || cur.Access != access || (cur.VisibilityN == 0 && cur.RecognitionN == 0) {
			continue
		}
		wrote = true
		label := map[string]string{"api": "API", "web": "Web"}[access]
		fmt.Fprintf(b, "### %s answers\n\n", label)
		if cur.Visibility != nil && cur.VisibilityCI != nil {
			small := ""
			if cur.LowSample {
				small = " (small sample)"
			}
			fmt.Fprintf(b, "- Visibility: **%.0f%%** (%d of %d unbranded answers; 95%% CI %.0f-%.0f%%)%s\n",
				*cur.Visibility, cur.VisibilityX, cur.VisibilityN, cur.VisibilityCI.Lo, cur.VisibilityCI.Hi, small)
			*cards = append(*cards, [2]string{"Visibility (" + label + ")", fmt.Sprintf("%.0f%%", *cur.Visibility)})
			if prev, err := s.sample.MeasureWindow(ctx, p.Slug, access, 2*metrics.DefaultWindowDays, metrics.DefaultWindowDays); err == nil && prev != nil && prev.VisibilityN > 0 {
				fmt.Fprintf(b, "- Previous %d days: %d of %d. Change: %s.\n", metrics.DefaultWindowDays, prev.VisibilityX, prev.VisibilityN,
					changeWord(metrics.Change(prev.VisibilityX, prev.VisibilityN, cur.VisibilityX, cur.VisibilityN)))
			}
		} else {
			b.WriteString("- Visibility: not measured (no successful unbranded answers)\n")
		}
		if cur.Recognition != nil {
			fmt.Fprintf(b, "- Branded recognition: %.0f%% of %d branded answers\n", *cur.Recognition, cur.RecognitionN)
		}
		if cur.Share != nil {
			fmt.Fprintf(b, "- Share of voice: %.0f%%\n", *cur.Share)
		}
		if cur.OwnCited != nil {
			fmt.Fprintf(b, "- Answers citing your own domain: %.0f%%\n", *cur.OwnCited)
		}
		if cur.Citations > 0 {
			share := "not measured"
			if cur.CitationShare != nil {
				share = fmt.Sprintf("%.0f%%", *cur.CitationShare)
			}
			fmt.Fprintf(b, "- Citations: %d from %d domains; owned share %s\n", cur.Citations, cur.UniqueDomains, share)
		}
		if len(cur.Leaders) > 0 {
			b.WriteString("\n| Brand | Mentions | Share |\n|---|---:|---:|\n")
			for i, l := range cur.Leaders {
				if i >= 8 {
					break
				}
				name := l.Name
				if l.IsBrand {
					name = "**" + name + "**"
				}
				fmt.Fprintf(b, "| %s | %d | %.0f%% |\n", cell(name), l.Mentions, l.Share)
			}
		}
		b.WriteString("\n")
	}
	if !wrote {
		b.WriteString("No answers were sampled in this window. Add engine keys and run a sample, or import a manual sampling sheet.\n\n")
	}
}

func changeWord(c string) string {
	switch c {
	case metrics.ChangeUp:
		return "significantly up"
	case metrics.ChangeDown:
		return "significantly down"
	case metrics.ChangeFlat:
		return "within noise"
	}
	return "not enough data"
}

func (s *Service) writeEngines(ctx context.Context, b *strings.Builder, p *model.Project) {
	m, _ := s.sample.PeriodMetrics(ctx, p.Slug)
	if m == nil {
		return
	}
	plats, _ := m.Payload["platforms"].(map[string]any)
	if len(plats) == 0 {
		return
	}
	var codes []string
	for code := range plats {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	b.WriteString("## By engine\n\n| Engine | Access | Answers | Visibility | 95% CI | Top 3 | Own domain cited |\n|---|---|---:|---:|---|---:|---:|\n")
	for _, code := range codes {
		pm, _ := plats[code].(map[string]any)
		label, _ := pm["label"].(string)
		if label == "" {
			label = code
		}
		modes, _ := pm["by_mode"].(map[string]any)
		var names []string
		for mode := range modes {
			names = append(names, mode)
		}
		sort.Strings(names)
		for _, mode := range names {
			ms, _ := modes[mode].(map[string]any)
			n, _ := asF(ms["mention_n"])
			ci := "-"
			if c, ok := ms["mention_ci"].(map[string]any); ok {
				lo, _ := asF(c["lo"])
				hi, _ := asF(c["hi"])
				ci = fmt.Sprintf("%.0f-%.0f%%", lo*100, hi*100)
			}
			fmt.Fprintf(b, "| %s | %s | %.0f | %s | %s | %s | %s |\n", cell(label), metrics.AccessOf(mode), n,
				pctOf(ms["mention_rate"]), ci, pctOf(ms["top3_rate"]), pctOf(ms["own_domain_cite_rate"]))
		}
	}
	b.WriteString("\n")
	// The CN-GEO comparison only means something when answers cite Chinese
	// sources, so it is shown only when some of the dataset's domains appear.
	{
		if cited := citedDomains(plats); len(cited) > 0 && len(CompareCited(cited).Covered) > 0 {
			bn := CompareCited(cited)
			fmt.Fprintf(b, "### Compared with the CN-GEO citation dataset\n\nCross-platform sources cited in your answers: **%d of %d** (%.0f%%). In that dataset brand sites account for 1.37%% of citations.\n\n",
				len(bn.Covered), len(bn.Covered)+len(bn.Missing), bn.CoverageRate*100)
			if len(bn.Missing) > 0 {
				b.WriteString("| Not yet cited | Type | Citations in the dataset |\n|---|---|---:|\n")
				for i, row := range bn.Missing {
					if i >= 8 {
						break
					}
					fmt.Fprintf(b, "| `%s` | %s | %d |\n", row.Domain, row.Category, row.National)
				}
				b.WriteString("\n")
			}
		}
	}
}

func (s *Service) writeAudit(ctx context.Context, b *strings.Builder, p *model.Project, cards *[][2]string) {
	b.WriteString("## Site audit\n\n")
	rep, _ := s.audit.Latest(ctx, p.Slug)
	if rep == nil {
		b.WriteString("Not audited yet.\n\n")
		return
	}
	passed := passedLayers(rep.Layers)
	b.WriteString("| Layer | Status | Question |\n|---|---|---|\n")
	for _, l := range rep.Layers {
		st := fmt.Sprint(l["status"])
		blockedBy, _ := l["blocked_by"].(string)
		if blockedBy != "" && st != "fail" {
			st += " (blocked by " + blockedBy + ")"
		}
		fmt.Fprintf(b, "| %v | %s | %v |\n", l["name"], st, l["question"])
	}
	*cards = append(*cards, [2]string{"Readiness", fmt.Sprintf("%d / %d layers", passed, len(rep.Layers))})
	issues, _, err := s.audit.Issues(ctx, p.Slug, audit.IssueFilter{})
	if err == nil && len(issues) > 0 {
		type row struct {
			code, title, sev, evidence string
			n                          int
			blocked                    bool
		}
		by := map[string]*row{}
		for _, is := range issues {
			if is.Severity == audit.SevInfo {
				continue
			}
			r := by[is.Code]
			if r == nil {
				r = &row{code: is.Code, title: is.Title, sev: is.Severity, evidence: is.Evidence, blocked: is.Blocked}
				by[is.Code] = r
			}
			r.n++
		}
		rows := make([]*row, 0, len(by))
		for _, r := range by {
			rows = append(rows, r)
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].sev != rows[j].sev {
				return rows[i].sev == audit.SevCritical
			}
			return rows[i].n > rows[j].n
		})
		if len(rows) > 0 {
			b.WriteString("\n| Issue | Severity | Count | Evidence |\n|---|---|---:|---|\n")
			for _, r := range rows {
				title := r.title
				if r.blocked {
					title += " (blocked)"
				}
				fmt.Fprintf(b, "| %s `%s` | %s | %d | %s |\n", cell(title), r.code, r.sev, r.n, r.evidence)
			}
		}
	}
	b.WriteString("\n")
}

func (s *Service) writeOpportunities(ctx context.Context, b *strings.Builder, p *model.Project) {
	items, err := s.opportunity.List(ctx, p.Slug, opportunity.ListFilter{Status: "new"})
	if err != nil || len(items) == 0 {
		return
	}
	b.WriteString("## Top opportunities\n\n| Priority | Source | What | Why |\n|---|---|---|---|\n")
	for i, it := range items {
		if i >= 10 {
			break
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", it.Priority, it.Source, cell(it.Title), cell(it.Why))
	}
	b.WriteString("\n")
}

func (s *Service) writeActions(ctx context.Context, b *strings.Builder, p *model.Project) {
	tasks, err := s.tasks.List(ctx, p.ID)
	if err != nil || len(tasks) == 0 {
		return
	}
	b.WriteString("## Actions\n\n| Code | Priority | Action | Status |\n|---|---|---|---|\n")
	for _, t := range tasks {
		if t.Status == model.TaskDismissed {
			continue
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", t.Code, t.Priority, cell(t.Title), t.Status)
	}
	b.WriteString("\n")
	if _, rows, _ := s.verify.Latest(ctx, p.Slug); len(rows) > 0 {
		b.WriteString("Latest verification:\n\n")
		for _, r := range rows {
			fmt.Fprintf(b, "- %s %s: %s\n", r.TaskCode, r.Verdict, r.Note)
		}
		b.WriteString("\n")
	}
}

func citedDomains(plats map[string]any) map[string]int {
	out := map[string]int{}
	for _, v := range plats {
		m, _ := v.(map[string]any)
		modes, _ := m["by_mode"].(map[string]any)
		for _, mv := range modes {
			ms, _ := mv.(map[string]any)
			if d, ok := ms["top_cited_domains"].(map[string]any); ok {
				for k, n := range d {
					if f, ok := asF(n); ok {
						out[k] += int(f)
					}
				}
			}
		}
	}
	return out
}

func pctOf(v any) string {
	f, ok := asF(v)
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%.0f%%", f*100)
}

func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "/")
	return strings.ReplaceAll(s, "\n", " ")
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func asF(v any) (float64, bool) {
	switch t := v.(type) {
	case nil:
		return 0, false
	case float64:
		return t, true
	case *float64:
		if t == nil {
			return 0, false
		}
		return *t, true
	case int:
		return float64(t), true
	}
	return 0, false
}

func insightStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func insightNum(v any) string {
	f, ok := asF(v)
	if !ok {
		return "not measured"
	}
	if !math.IsNaN(f) && !math.IsInf(f, 0) && f == math.Trunc(f) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// passedLayers counts layers that pass and are not behind a failing layer,
// matching the dashboard's readiness card.
func passedLayers(layers []map[string]any) int {
	n := 0
	for _, l := range layers {
		blockedBy, _ := l["blocked_by"].(string)
		if fmt.Sprint(l["status"]) == "ok" && blockedBy == "" {
			n++
		}
	}
	return n
}
