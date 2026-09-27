// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// auditPriority: critical -> P0, warning -> P1, info -> P2. Observational and
// rule-of-thumb rules are capped at P1.
func auditPriority(is audit.Issue) string {
	p := "P2"
	switch is.Severity {
	case audit.SevCritical:
		p = "P0"
	case audit.SevWarning:
		p = "P1"
	}
	if p == "P0" && (is.Evidence == audit.EvObservational || is.Evidence == audit.EvHeuristic) {
		p = "P1"
	}
	return p
}

// FromAudit groups unblocked critical and warning findings by code. Blocked
// findings stay on the audit page: fixing them is invisible until the
// upstream layer is fixed.
func FromAudit(rows []model.AuditIssue) []Item {
	by := map[string]*Item{}
	var order []string
	for _, r := range rows {
		if r.Blocked || r.Severity == audit.SevInfo {
			continue
		}
		it := by[r.Code]
		if it == nil {
			is := audit.Lookup(r.Code)
			it = &Item{
				Key: "audit:" + r.Code, Source: "audit", Kind: r.Code, Priority: auditPriority(is),
				Title: is.Title, Why: is.Why, Fix: is.Fix, Evidence: is.Evidence, Refs: is.Refs,
				Acceptance: map[string]any{"type": "auto", "check": "issue.absent:" + r.Code},
			}
			by[r.Code] = it
			order = append(order, r.Code)
		}
		if r.URL != "" {
			it.URLs = append(it.URLs, r.URL)
		}
	}
	out := make([]Item, 0, len(order))
	for _, c := range order {
		it := *by[c]
		it.Baseline = map[string]any{"count": len(it.URLs)}
		out = append(out, it)
	}
	return out
}

// FromCitation turns citation gaps into items. perPrompt holds the current
// x/n (mentions / successful unbranded runs) per prompt for the baseline.
func FromCitation(ops []sample.Opportunity, access string, perPrompt map[string][2]int) []Item {
	out := make([]Item, 0, len(ops))
	for _, o := range ops {
		p := "P1"
		if o.Difficulty == "locked-in" || o.QID == "" {
			p = "P2"
		}
		acc := map[string]any{"type": "manual"}
		var base map[string]any
		if o.QID != "" {
			acc = map[string]any{"type": "auto", "check": fmt.Sprintf("metrics.prompt_up:%s:%s", o.QID, access)}
			if xn, ok := perPrompt[o.QID]; ok {
				base = map[string]any{"prompt": map[string]any{"x": xn[0], "n": xn[1]}}
			}
		}
		detail := map[string]any{"difficulty": o.Difficulty, "access": access}
		if len(o.Prompts) > 0 {
			// The dashboard builds the translated title from the prompt.
			detail["prompt"] = o.Prompts[0].Text
		}
		out = append(out, Item{
			Key: fmt.Sprintf("citation:%s:%s", o.QID, o.Category), Source: "citation", Kind: o.Category, Priority: p,
			Title: o.Title, Why: o.Why, QID: o.QID, URLs: o.URLs, Evidence: audit.EvObservational, Refs: []string{"chen2025"},
			Fix:        "Earn a mention or citation on the sources listed here; AI answers draw mostly on third-party pages.",
			Acceptance: acc, Detail: detail, Baseline: base,
		})
	}
	return out
}

// Brand identifies the project for filtering branded search queries.
type Brand struct {
	Name    string
	Aliases []string
	Site    string
}

// FromSearch turns Search Console opportunities into items. Queries that
// name the brand are dropped: they are people looking for you already, not
// demand to win. Spelling and word-order variants of one query ("acme cli
// github", "github acmecli") become one item.
func FromSearch(ops []webstats.SearchOp, b Brand) []Item {
	out := make([]Item, 0, len(ops))
	seen := map[string]int{}
	for _, o := range ops {
		q := strings.TrimSpace(o.Query)
		if strings.HasPrefix(strings.ToLower(q), "site:") {
			continue // the owner's own site: searches are not demand
		}
		if q != "" && (model.IsPromptBranded(q, b.Name, b.Aliases, b.Site) || b.inCompact(q)) {
			continue
		}
		if q != "" {
			vk := o.Type + "|" + variantKey(q)
			if i, ok := seen[vk]; ok {
				v, _ := out[i].Detail["variants"].([]string)
				out[i].Detail["variants"] = append(v, q)
				continue
			}
			seen[vk] = len(out)
		}
		id := q
		if id == "" {
			id = o.URL
		}
		urls := o.URLs
		if len(urls) == 0 {
			urls = nonEmpty(o.URL)
		}
		out = append(out, Item{
			Key: fmt.Sprintf("search:%s:%s", o.Type, id), Source: "search", Kind: o.Type, Priority: "P2",
			Title: searchTitle(o), Why: o.Detail, URLs: urls, Acceptance: map[string]any{"type": "manual"},
			Detail: map[string]any{"query": q, "metric": o.Metric, "severity": o.Severity},
		})
	}
	return out
}

// inCompact matches the brand ignoring spaces, hyphens and dots, so
// "acme cli" counts as a search for "acmecli".
func (b Brand) inCompact(q string) bool {
	cq := compact(q)
	for _, n := range append([]string{b.Name}, b.Aliases...) {
		if c := compact(n); len([]rune(c)) >= 3 && strings.Contains(cq, c) {
			return true
		}
	}
	return false
}

func compact(s string) string {
	var rs []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			rs = append(rs, r)
		}
	}
	return string(rs)
}

// variantKey ignores case, spacing, punctuation and word order by comparing
// the sorted letters and digits of a query.
func variantKey(q string) string {
	var rs []rune
	for _, r := range strings.ToLower(q) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			rs = append(rs, r)
		}
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
	return string(rs)
}

// FromMetric raises an item only when the visibility drop is outside the
// noise band (Newcombe 95% interval of the difference excludes zero).
func FromMetric(access string, prevX, prevN, curX, curN int) []Item {
	if metrics.Change(prevX, prevN, curX, curN) != metrics.ChangeDown {
		return nil
	}
	iv := metrics.NewcombeDiff(curX, curN, prevX, prevN)
	return []Item{{
		Key: "metric:visibility_down:" + access, Source: "metric", Kind: "visibility_down", Priority: "P0",
		Title: "Visibility dropped", Evidence: audit.EvHeuristic,
		Why: fmt.Sprintf("Unbranded visibility went from %d/%d to %d/%d; the 95%% interval of the change is %.0f to %.0f points.",
			prevX, prevN, curX, curN, iv.Lo*100, iv.Hi*100),
		Fix:        "Open Answers for the prompts that lost mentions and compare their citations with the previous window.",
		Acceptance: map[string]any{"type": "auto", "check": "metrics.visibility_not_down:" + access},
		Detail:     map[string]any{"prev": []int{prevX, prevN}, "cur": []int{curX, curN}, "access": access},
		Baseline:   map[string]any{"visibility": map[string]any{"x": prevX, "n": prevN}},
	}}
}

var sourceOrder = map[string]int{"audit": 0, "metric": 1, "citation": 2, "search": 3}

// Sort orders by priority, then source, keeping input order otherwise.
func Sort(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return sourceOrder[items[i].Source] < sourceOrder[items[j].Source]
	})
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func searchTitle(o webstats.SearchOp) string {
	switch o.Type {
	case "striking_distance":
		return "Close to the top three: " + o.Title
	case "low_ctr":
		return "Low click-through: " + o.Title
	case "content_decay":
		return "Clicks declining: " + o.Title
	case "cannibalization":
		return "Several pages compete for: " + o.Title
	}
	return o.Title
}
