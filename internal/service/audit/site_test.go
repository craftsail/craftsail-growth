// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import "testing"

func TestSiteFindingsFromSignals(t *testing.T) {
	site := map[string]any{
		"ai_bots_blocked": []any{"GPTBot"}, "ai_ua_blocked": []any{},
		"has_sitemap": false, "has_llms_txt": false,
	}
	got := codesOf(SiteFindings(site, LangStats{}))
	for _, c := range []string{"ROBOTS_BLOCKS_AI", "NO_SITEMAP", "NO_LLMS_TXT"} {
		if got[c] != 1 {
			t.Fatalf("missing %s in %v", c, got)
		}
	}
	if got["AI_UA_BLOCKED"] != 0 {
		t.Fatal("empty ai_ua_blocked must not fire")
	}
}

func TestSiteFindingsLanguage(t *testing.T) {
	ok := map[string]any{"has_sitemap": true, "has_llms_txt": true}
	got := codesOf(SiteFindings(ok, LangStats{ZH: 5, EN: 0, Total: 5}))
	if len(got) != 0 {
		t.Fatalf("a single-language site has no language finding: %v", got)
	}
	got = codesOf(SiteFindings(ok, LangStats{ZH: 10, EN: 2, Total: 12}))
	if got["LANG_IMBALANCE"] != 1 {
		t.Fatalf("10 zh vs 2 en should be imbalanced: %v", got)
	}
}

func TestBlockedMarksDownstreamLayers(t *testing.T) {
	fs := []Finding{{Code: "ROBOTS_BLOCKS_AI"}, {Code: "NO_JSONLD", URL: "https://e.com/"}, {Code: "NO_SITEMAP"}}
	for _, r := range MarkBlocked(fs) {
		switch r.Code {
		case "ROBOTS_BLOCKS_AI":
			if r.Blocked {
				t.Fatal("the failing layer itself is not blocked")
			}
		case "NO_JSONLD", "NO_SITEMAP":
			if !r.Blocked {
				t.Fatalf("%s is downstream of a critical access failure", r.Code)
			}
		}
	}
}

func TestLayerStatus(t *testing.T) {
	st := LayerStatus([]Finding{{Code: "NO_SITEMAP"}, {Code: "NO_LLMS_TXT"}})
	if st[LayerAccess] != "ok" || st[LayerDiscover] != "warn" || st[LayerCite] != "ok" {
		t.Fatalf("status = %v", st)
	}
	st = LayerStatus([]Finding{{Code: "SPA_SHELL", URL: "https://e.com/x"}})
	if st[LayerAccess] != "fail" {
		t.Fatalf("status = %v", st)
	}
}
