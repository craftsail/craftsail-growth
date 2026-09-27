// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

// LangStats summarizes the crawl for the site-level language checks.
type LangStats struct {
	ZH, EN, Total int
	Multilingual  bool
	HreflangPages int
	LowValueURLs  int
	DupBodyGroups [][]string
}

func siteFind(code string, detail map[string]any) Finding { return Finding{Code: code, Detail: detail} }

// SiteFindings turns crawl-level signals into registered findings.
func SiteFindings(site map[string]any, ls LangStats) []Finding {
	var out []Finding
	if b := asStringSlice(site["ai_bots_blocked"]); len(b) > 0 {
		out = append(out, siteFind("ROBOTS_BLOCKS_AI", map[string]any{"bots": b}))
	}
	if b := asStringSlice(site["ai_ua_blocked"]); len(b) > 0 {
		out = append(out, siteFind("AI_UA_BLOCKED", map[string]any{"bots": b}))
	}
	if !asBool(site["has_sitemap"]) {
		out = append(out, siteFind("NO_SITEMAP", nil))
	} else if site["robots_sitemap_declared"] == false {
		out = append(out, siteFind("SITEMAP_NOT_DECLARED", nil))
	}
	if ls.LowValueURLs > 0 {
		out = append(out, siteFind("SITEMAP_LOW_VALUE_URLS", map[string]any{"count": ls.LowValueURLs}))
	}
	if !asBool(site["has_llms_txt"]) {
		out = append(out, siteFind("NO_LLMS_TXT", nil))
	} else if lch, ok := site["llms_txt_check"].(map[string]any); ok {
		if broken, ok := lch["broken"].([]any); ok && len(broken) > 0 {
			out = append(out, siteFind("LLMS_TXT_BROKEN_LINKS", map[string]any{"broken": broken}))
		}
	}
	for _, g := range ls.DupBodyGroups {
		out = append(out, siteFind("DUPLICATE_BODY", map[string]any{"urls": g}))
	}
	if ls.Multilingual && ls.Total > 0 && float64(ls.HreflangPages)/float64(ls.Total) < 0.3 {
		out = append(out, siteFind("HREFLANG_LOW", map[string]any{"pages": ls.HreflangPages, "total": ls.Total}))
	}
	if ls.Total > 0 {
		if ls.ZH > 0 && ls.EN > 0 {
			mx := ls.EN
			if ls.ZH > mx {
				mx = ls.ZH
			}
			if abs(ls.EN-ls.ZH) > int(float64(mx)*0.7) {
				out = append(out, siteFind("LANG_IMBALANCE", map[string]any{"zh": ls.ZH, "en": ls.EN}))
			}
		}
	}
	return out
}

// IssueRow is a finding with its registry fields and blocked flag.
type IssueRow struct {
	Finding
	Severity string
	Layer    string
	Blocked  bool
}

func layerIndex(l string) int {
	for i, x := range LayerOrder {
		if x == l {
			return i
		}
	}
	return len(LayerOrder)
}

// MarkBlocked flags findings whose layer is below the first layer that has a
// critical finding. Those fixes are invisible until the upstream one lands.
func MarkBlocked(fs []Finding) []IssueRow {
	first := len(LayerOrder)
	for _, f := range fs {
		is := Lookup(f.Code)
		if is.Severity == SevCritical && layerIndex(is.Layer) < first {
			first = layerIndex(is.Layer)
		}
	}
	out := make([]IssueRow, 0, len(fs))
	for _, f := range fs {
		is := Lookup(f.Code)
		out = append(out, IssueRow{Finding: f, Severity: is.Severity, Layer: is.Layer, Blocked: layerIndex(is.Layer) > first})
	}
	return out
}

// LayerStatus: fail if the layer has a critical finding, warn if it has a
// warning, otherwise ok. Info findings do not change status.
func LayerStatus(fs []Finding) map[string]string {
	st := map[string]string{}
	for _, l := range LayerOrder {
		st[l] = "ok"
	}
	for _, f := range fs {
		is := Lookup(f.Code)
		switch is.Severity {
		case SevCritical:
			st[is.Layer] = "fail"
		case SevWarning:
			if st[is.Layer] == "ok" {
				st[is.Layer] = "warn"
			}
		}
	}
	return st
}
