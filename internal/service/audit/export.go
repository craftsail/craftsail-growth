// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// Dimension and block keys are stored with their original (Chinese) names;
// these tables give the English labels used in every export and in the UI.
var dimMax = []struct {
	Key   string
	Label string
	Max   float64
}{
	{"可抓取性", "Crawlability", 15}, {"内容长度", "Length", 15}, {"结构规范", "Structure", 20},
	{"可抽取块", "Extractable blocks", 25}, {"权威信号", "Authority", 15}, {"对题性", "Relevance", 10},
}

var blockLabels = []struct{ Key, Label string }{
	{"定义", "definition"}, {"数字事实", "numbers"}, {"对比", "comparison"}, {"操作步骤", "steps"},
}

func (s *Service) ExportMarkdown(ctx context.Context, slug string) (string, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	rep, err := s.Latest(ctx, slug)
	if err != nil {
		return "", err
	}
	if rep == nil {
		return "", fmt.Errorf("no audit yet; run a crawl and audit first")
	}
	return Markdown(p, rep), nil
}

func Markdown(p *model.Project, rep *Report) string {
	var b strings.Builder
	site := p.Site
	if site == "" {
		site = "(no site)"
	}
	fmt.Fprintf(&b, "# %s · Site audit\n\n", p.Name)
	fmt.Fprintf(&b, "Site: %s\n", site)
	fmt.Fprintf(&b, "Average: %.1f / 100 · %d pages · grades A%d B%d C%d D%d\n\n",
		rep.AvgScore, rep.PageCount,
		rep.GradeDistribution["A"], rep.GradeDistribution["B"],
		rep.GradeDistribution["C"], rep.GradeDistribution["D"])
	b.WriteString("Use this report to fix the site so crawlers that do not render JavaScript (GPTBot, ClaudeBot and similar) can read and cite the text.\n")
	b.WriteString("Fix site-level issues and D/C pages first. Do not invent brand facts. After a fix the text must be present in the static HTML.\n\n")
	b.WriteString(FormulaMarkdown)
	b.WriteString("\n")

	var siteRows []IssueRow
	for _, f := range rep.Findings {
		if f.URL == "" {
			siteRows = append(siteRows, f)
		}
	}
	if len(siteRows) > 0 {
		b.WriteString("## Site-level issues\n\n")
		for _, f := range siteRows {
			is := Lookup(f.Code)
			fmt.Fprintf(&b, "- [%s] %s (%s). Fix: %s\n", is.Severity, is.Title, f.Code, is.Fix)
		}
		b.WriteString("\n")
	}
	if len(rep.Layers) > 0 {
		b.WriteString("## Readiness layers\n\n")
		for _, l := range rep.Layers {
			fmt.Fprintf(&b, "### %v (%v)\n\n%v\n\n", l["name"], l["status"], l["question"])
			if x, _ := l["blocked_by"].(string); x != "" {
				fmt.Fprintf(&b, "Blocked by %s.\n\n", x)
			}
			if iss, ok := l["issues"].([]any); ok {
				for _, i := range iss {
					fmt.Fprintf(&b, "- %v\n", i)
				}
				if len(iss) > 0 {
					b.WriteString("\n")
				}
			}
			if iss, ok := l["issues"].([]string); ok {
				for _, i := range iss {
					fmt.Fprintf(&b, "- %s\n", i)
				}
				if len(iss) > 0 {
					b.WriteString("\n")
				}
			}
		}
	}
	if len(rep.BlockGap) > 0 {
		b.WriteString("## Missing extractable blocks\n\n| Block | Pages missing | Pages |\n|---|---:|---:|\n")
		for _, g := range rep.BlockGap {
			fmt.Fprintf(&b, "| %s | %v | %v |\n", blockLabel(fmt.Sprint(g["block"])), g["missing_pages"], g["total"])
		}
		b.WriteString("\n")
	}

	pages := append([]model.AuditPage(nil), rep.Pages...)
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Grade != pages[j].Grade {
			return pages[i].Grade > pages[j].Grade
		}
		return pages[i].Score < pages[j].Score
	})
	b.WriteString("## Pages (worst first)\n")
	for _, pg := range pages {
		fmt.Fprintf(&b, "\n### %s · %.1f · %s\n\n", pg.Grade, pg.Score, or(pg.Title, pg.URL))
		fmt.Fprintf(&b, "- URL: %s\n- Words: %d\n", pg.URL, pg.WordCount)
		if len(pg.JSONLDTypes) > 0 {
			fmt.Fprintf(&b, "- JSON-LD: %s\n", strings.Join(pg.JSONLDTypes, ", "))
		} else {
			b.WriteString("- JSON-LD: none\n")
		}
		var dimParts []string
		for _, d := range dimMax {
			dimParts = append(dimParts, fmt.Sprintf("%s %.0f/%.0f", d.Label, asF(pg.Dimensions[d.Key]), d.Max))
		}
		fmt.Fprintf(&b, "- Score parts: %s\n", strings.Join(dimParts, " · "))
		if pg.Blocks != nil {
			var miss []string
			for _, bl := range blockLabels {
				if !truthy(pg.Blocks[bl.Key]) {
					miss = append(miss, bl.Label)
				}
			}
			if len(miss) > 0 {
				fmt.Fprintf(&b, "- Missing blocks: %s\n", strings.Join(miss, ", "))
			}
		}
		if len(pg.IssueCodes) > 0 {
			b.WriteString("- Issues:\n")
			for _, c := range pg.IssueCodes {
				is := Lookup(c)
				fmt.Fprintf(&b, "  - [%s] %s (%s). Fix: %s\n", is.Severity, is.Title, c, is.Fix)
			}
		}
	}
	b.WriteString("\n")
	return b.String()
}

func blockLabel(key string) string {
	for _, bl := range blockLabels {
		if bl.Key == key {
			return bl.Label
		}
	}
	return key
}

func or(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func asF(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	}
	return 0
}

func truthy(v any) bool {
	b, _ := v.(bool)
	return b
}
