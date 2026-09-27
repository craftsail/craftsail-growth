// SPDX-License-Identifier: AGPL-3.0-or-later

package verify

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
)

var funcPage = regexp.MustCompile(`(?i)/(login|signin|signup|register|cart|checkout|account|auth|contact)(/|$)`)

type Outcome struct {
	OK       *bool
	Note     string
	Progress map[string]any
}

func Check(task model.Task, audit model.Audit, pages []model.AuditPage, metrics map[string]any) Outcome {
	acc := task.Acceptance
	if acc == nil {
		return Outcome{Note: "needs a manual check"}
	}
	if typ, _ := acc["type"].(string); typ != "auto" {
		return Outcome{Note: "needs a manual check"}
	}
	expr, _ := acc["check"].(string)
	site := audit.Site
	if site == nil {
		site = map[string]any{}
	}
	byURL := map[string]model.AuditPage{}
	for _, p := range pages {
		byURL[p.URL] = p
	}
	aff := task.Affected
	if aff == nil {
		aff = []string{}
	}
	base := len(aff)
	if task.BaselineCount > 0 {
		base = task.BaselineCount
	}

	yes, no := true, false
	switch {
	case expr == "site.no_ai_bot_block":
		blocked := asStrings(site["ai_bots_blocked"])
		if len(blocked) == 0 {
			return Outcome{OK: &yes, Note: "robots.txt blocks no AI crawler"}
		}
		return Outcome{OK: &no, Note: "still blocked: " + strings.Join(blocked, ", ")}
	case expr == "site.no_ai_ua_block":
		bad := asStrings(site["ai_ua_blocked"])
		probe, _ := site["ai_ua_probe"].(map[string]any)
		if len(probe) == 0 && len(bad) == 0 {
			return Outcome{Note: "no user-agent probe in this crawl; run crawl again"}
		}
		if len(bad) == 0 {
			return Outcome{OK: &yes, Note: "the home page answers every AI crawler user agent"}
		}
		return Outcome{OK: &no, Note: "still rejected: " + strings.Join(bad, ", ")}
	case expr == "site.robots_sitemap_declared":
		ok := asBool(site["robots_sitemap_declared"])
		if ok {
			return Outcome{OK: &yes, Note: "robots.txt declares the sitemap"}
		}
		return Outcome{OK: &no, Note: "robots.txt still does not declare the sitemap"}
	case expr == "site.has_sitemap":
		ok := asBool(site["has_sitemap"])
		if ok {
			return Outcome{OK: &yes, Note: fmt.Sprintf("sitemap is live (%d URLs)", asInt(site["sitemap_url_count"]))}
		}
		return Outcome{OK: &no, Note: "sitemap is still missing"}
	case expr == "site.has_llms_txt":
		ok := asBool(site["has_llms_txt"])
		if ok {
			return Outcome{OK: &yes, Note: "llms.txt is live"}
		}
		return Outcome{OK: &no, Note: "llms.txt is still missing"}
	case expr == "site.llms_txt_valid":
		if !asBool(site["has_llms_txt"]) {
			return Outcome{OK: &no, Note: "llms.txt is missing"}
		}
		lch, _ := site["llms_txt_check"].(map[string]any)
		if lch == nil {
			return Outcome{Note: "no llms.txt check in this crawl; run crawl again"}
		}
		nbad := len(asAny(lch["broken"])) + len(asAny(lch["robots_blocked"]))
		prog := map[string]any{"label": "broken llms.txt links", "cur": nbad, "target": 0, "op": "lte"}
		if nbad == 0 {
			return Outcome{OK: &yes, Note: fmt.Sprintf("all %v sampled links work", lch["checked"]), Progress: prog}
		}
		return Outcome{OK: &no, Note: fmt.Sprintf("%d links are still broken or blocked", nbad), Progress: prog}
	case expr == "site.sitemap_clean":
		if site["sitemap_noisy_urls"] == nil {
			return Outcome{Note: "no sitemap URL check in this crawl; run crawl again"}
		}
		n := asInt(site["sitemap_noisy_urls"])
		prog := map[string]any{"label": "low-value sitemap URLs", "cur": n, "target": 0, "op": "lte"}
		if n == 0 {
			return Outcome{OK: &yes, Note: "the sitemap has no low-value URLs", Progress: prog}
		}
		return Outcome{OK: &no, Note: fmt.Sprintf("%d parameter, search or pagination URLs remain", n), Progress: prog}
	case strings.HasPrefix(expr, "site.avg_score_gte:"):
		tgt := asFloat(strings.TrimPrefix(expr, "site.avg_score_gte:"))
		cur := audit.AvgScore
		ok := cur >= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("average score %v, target %v", cur, tgt),
			Progress: map[string]any{"label": "average content readiness score", "cur": cur, "target": tgt, "op": "gte"}}
	case strings.HasPrefix(expr, "site.en_pages_gte:"):
		tgt := asInt(mustParse(strings.TrimPrefix(expr, "site.en_pages_gte:")))
		lc := audit.LanguageCoverage
		if lc == nil {
			lc = map[string]any{}
		}
		cur := asInt(lc["en_pages"])
		ok := cur >= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d English content pages, target %d", cur, tgt),
			Progress: map[string]any{"label": "English content pages", "cur": cur, "target": tgt, "op": "gte"}}
	case strings.HasPrefix(expr, "site.hreflang_gte:"):
		tgt := asFloat(strings.TrimPrefix(expr, "site.hreflang_gte:"))
		lc := audit.LanguageCoverage
		if lc == nil {
			lc = map[string]any{}
		}
		total := asInt(lc["content_pages"])
		if total == 0 || lc["hreflang_pages"] == nil {
			return Outcome{Note: "no hreflang data in this audit; run crawl and audit again"}
		}
		hp := asInt(lc["hreflang_pages"])
		cur := float64(hp) / float64(total)
		ok := cur >= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("hreflang on %d of %d pages", hp, total),
			Progress: map[string]any{"label": "hreflang coverage", "cur": round2(cur), "target": tgt, "op": "gte", "pct": true}}
	case expr == "pages.static_text":
		bad := 0
		for _, u := range aff {
			path := pathOf(u)
			if funcPage.MatchString(path) {
				continue
			}
			for _, c := range byURL[u].IssueCodes {
				if c == "SPA_SHELL" {
					bad++
					break
				}
			}
		}
		o := &yes
		if bad > 0 {
			o = &no
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d of %d pages now serve text in static HTML", base-bad, base),
			Progress: map[string]any{"label": "pages without static text", "cur": bad, "target": 0, "op": "lte", "base": base}}
	case expr == "pages.has_jsonld":
		bad := 0
		for _, u := range aff {
			if len(byURL[u].JSONLDTypes) == 0 {
				bad++
			}
		}
		o := &yes
		if bad > 0 {
			o = &no
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d of %d pages have JSON-LD", base-bad, base),
			Progress: map[string]any{"label": "pages without JSON-LD", "cur": bad, "target": 0, "op": "lte", "base": base}}
	case expr == "pages.no_noindex":
		bad := 0
		for _, u := range aff {
			for _, c := range byURL[u].IssueCodes {
				if c == "NOINDEX" || c == "XROBOTS_NOINDEX" {
					bad++
					break
				}
			}
		}
		o := &yes
		if bad > 0 {
			o = &no
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d of %d pages no longer have noindex", base-bad, base),
			Progress: map[string]any{"label": "pages with noindex", "cur": bad, "target": 0, "op": "lte", "base": base}}
	case expr == "pages.quotable":
		if task.BaselineCount > 0 {
			base = task.BaselineCount
		} else {
			base = len(aff)
		}
		cur := 0
		for _, u := range aff {
			for _, c := range byURL[u].IssueCodes {
				if c == "NO_QUOTABLE_PASSAGE" {
					cur++
					break
				}
			}
		}
		tgt := base / 2
		ok := cur <= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d pages without a quotable passage (baseline %d, target <= %d)", cur, base, tgt),
			Progress: map[string]any{"label": "pages without a quotable passage", "cur": cur, "target": tgt, "op": "lte", "base": base}}
	case strings.HasPrefix(expr, "pages.block:"):
		blk := strings.TrimPrefix(expr, "pages.block:")
		if task.BaselineCount > 0 {
			base = task.BaselineCount
		}
		cur := 0
		for _, p := range pages {
			if !asBool(p.Blocks[blk]) {
				cur++
			}
		}
		tgt := base / 2
		ok := cur <= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d pages missing %s (baseline %d, target <= %d)", cur, blockName(blk), base, tgt),
			Progress: map[string]any{"label": "pages missing " + blockName(blk), "cur": cur, "target": tgt, "op": "lte", "base": base}}
	case strings.HasPrefix(expr, "pages.wordcount_gte:"):
		n := asInt(mustParse(strings.TrimPrefix(expr, "pages.wordcount_gte:")))
		if task.BaselineCount > 0 {
			base = task.BaselineCount
		}
		cur := 0
		for _, p := range pages {
			if p.WordCount >= 100 && p.WordCount < n {
				cur++
			}
		}
		tgt := int(float64(base) * 0.6)
		ok := cur <= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("%d pages under %d words (baseline %d, target <= %d)", cur, n, base, tgt),
			Progress: map[string]any{"label": fmt.Sprintf("pages under %d words", n), "cur": cur, "target": tgt, "op": "lte", "base": base}}
	case strings.HasPrefix(expr, "metrics.mention_rate_gte:"):
		// Old actions carry a market segment ("...:cn:0.3"); it is ignored.
		parts := strings.Split(expr, ":")
		if len(parts) < 2 {
			return Outcome{Note: "unknown check `" + expr + "`"}
		}
		cur := EngineAvg(metrics, "mention_rate")
		if cur == nil {
			return Outcome{Note: "no samples in this window"}
		}
		tgt := asFloat(parts[len(parts)-1])
		ok := *cur >= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("Mention rate %.1f%%, target %.0f%%", *cur*100, tgt*100),
			Progress: map[string]any{"label": "mention rate", "cur": round3(*cur), "target": tgt, "op": "gte", "pct": true}}
	case strings.HasPrefix(expr, "metrics.own_cite_gte:"):
		// Old actions carry a market segment ("...:cn:0.3"); it is ignored.
		parts := strings.Split(expr, ":")
		if len(parts) < 2 {
			return Outcome{Note: "unknown check `" + expr + "`"}
		}
		cur := EngineAvg(metrics, "own_domain_cite_rate")
		if cur == nil {
			return Outcome{Note: "no samples in this window"}
		}
		tgt := asFloat(parts[len(parts)-1])
		ok := *cur >= tgt
		o := &no
		if ok {
			o = &yes
		}
		return Outcome{OK: o, Note: fmt.Sprintf("Own-domain citation rate %.1f%%, target %.0f%%", *cur*100, tgt*100),
			Progress: map[string]any{"label": "own-domain citation rate", "cur": round3(*cur), "target": tgt, "op": "gte", "pct": true}}
	case strings.HasPrefix(expr, "external.any:"):
		targets := strings.Split(strings.TrimPrefix(expr, "external.any:"), ",")
		doms := citedDomains(metrics)
		var hit []string
		for _, t := range targets {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			for d := range doms {
				if d == t || strings.HasSuffix(d, "."+t) {
					hit = append(hit, t)
					break
				}
			}
		}
		ok := len(hit) > 0
		o := &no
		if ok {
			o = &yes
		}
		note := fmt.Sprintf("none of the %d target domains appear in sampled citations", len(targets))
		if ok {
			note = "cited: " + strings.Join(hit, ", ")
		}
		return Outcome{OK: o, Note: note, Progress: map[string]any{"label": "target domains cited", "cur": len(hit), "target": 1, "op": "gte"}}
	default:
		return Outcome{Note: "unknown check `" + expr + "`"}
	}
}

// EngineAvg averages a per-engine rate across all engines that have it.
func EngineAvg(metrics map[string]any, field string) *float64 {
	if metrics == nil {
		return nil
	}
	plats, _ := metrics["platforms"].(map[string]any)
	var vals []float64
	for _, v := range plats {
		m, _ := v.(map[string]any)
		if f, ok := m[field].(float64); ok {
			vals = append(vals, f)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	x := s / float64(len(vals))
	return &x
}

func citedDomains(metrics map[string]any) map[string]int {
	out := map[string]int{}
	if metrics == nil {
		return out
	}
	plats, _ := metrics["platforms"].(map[string]any)
	for _, v := range plats {
		m, _ := v.(map[string]any)
		doms, _ := m["top_cited_domains"].(map[string]any)
		for k, n := range doms {
			out[k] += asInt(n)
		}
	}
	return out
}

func asStrings(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		var o []string
		for _, x := range t {
			if s, ok := x.(string); ok {
				o = append(o, s)
			}
		}
		return o
	}
	return nil
}

func asAny(v any) []any {
	a, _ := v.([]any)
	return a
}

func asBool(v any) bool { b, _ := v.(bool); return b }

func asInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case string:
		var n int
		fmt.Sscanf(t, "%d", &n)
		return n
	}
	return 0
}

func asFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func mustParse(s string) any { return s }

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
func round3(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }

func pathOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return u.Path
}

// blockName maps the stored block key to its English name.
func blockName(key string) string {
	switch key {
	case "定义":
		return "definition"
	case "数字事实":
		return "numbers"
	case "对比":
		return "comparison"
	case "操作步骤":
		return "steps"
	}
	return key
}
