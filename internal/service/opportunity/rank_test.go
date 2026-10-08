// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func TestFromAuditGroupsByCodeAndSkipsBlocked(t *testing.T) {
	rows := []model.AuditIssue{
		{Code: "NO_JSONLD", URL: "https://e.com/a", Severity: "warning", Layer: "understand"},
		{Code: "NO_JSONLD", URL: "https://e.com/b", Severity: "warning", Layer: "understand"},
		{Code: "SPA_SHELL", URL: "https://e.com/c", Severity: "critical", Layer: "access"},
		{Code: "NO_DEFINITION", URL: "https://e.com/d", Severity: "warning", Layer: "cite", Blocked: true},
		{Code: "NO_LLMS_TXT", Severity: "info", Layer: "discover"},
	}
	items := FromAudit(rows)
	if len(items) != 2 {
		t.Fatalf("want NO_JSONLD and SPA_SHELL only, got %+v", items)
	}
	byKey := map[string]Item{}
	for _, it := range items {
		byKey[it.Key] = it
	}
	if it := byKey["audit:NO_JSONLD"]; it.Priority != "P1" || len(it.URLs) != 2 || it.Baseline["count"] != 2 {
		t.Fatalf("NO_JSONLD = %+v", it)
	}
	if it := byKey["audit:SPA_SHELL"]; it.Priority != "P0" || it.Acceptance["check"] != "issue.absent:SPA_SHELL" {
		t.Fatalf("SPA_SHELL = %+v", it)
	}
}

func TestObservationalAuditCapsAtP1(t *testing.T) {
	if p := auditPriority(audit.Issue{Severity: audit.SevCritical, Evidence: audit.EvObservational}); p != "P1" {
		t.Fatalf("got %s", p)
	}
}

func TestFromCitation(t *testing.T) {
	ops := []sample.Opportunity{{Category: "outreach", Title: "Get into the reviews", QID: "q3", URLs: []string{"https://g2.com/x"}}}
	items := FromCitation(ops, "api", map[string][2]int{"q3": {1, 10}})
	if len(items) != 1 || items[0].Key != "citation:q3:outreach" || items[0].Priority != "P1" {
		t.Fatalf("%+v", items)
	}
	if items[0].Acceptance["check"] != "metrics.prompt_up:q3:api" {
		t.Fatalf("acceptance = %v", items[0].Acceptance)
	}
	if b, _ := items[0].Baseline["prompt"].(map[string]any); b["x"] != 1 || b["n"] != 10 {
		t.Fatalf("baseline = %v", items[0].Baseline)
	}
	locked := FromCitation([]sample.Opportunity{{Category: "outreach", QID: "q4", Difficulty: "locked-in"}}, "api", nil)
	if locked[0].Priority != "P2" {
		t.Fatalf("locked-in sources rank lower, got %s", locked[0].Priority)
	}
}

func TestFromSearch(t *testing.T) {
	items := FromSearch([]webstats.SearchOp{{Type: "striking_distance", Title: "best cli", Query: "best cli"}}, Brand{})
	if len(items) != 1 || items[0].Key != "search:striking_distance:best cli" || items[0].Priority != "P2" || items[0].Acceptance["type"] != "manual" {
		t.Fatalf("%+v", items)
	}
	if items[0].Title != "Ranking candidate to review: best cli" {
		t.Fatalf("title = %q", items[0].Title)
	}
	if got := FromSearch([]webstats.SearchOp{{Type: "low_ctr", Title: "site:acme.com", Query: "site:acme.com"}}, Brand{}); len(got) != 0 {
		t.Fatalf("site: queries are the owner's own searches, got %+v", got)
	}
}

func TestFromMetricOnlyOnSignificantDrop(t *testing.T) {
	// 18/30 -> 6/30: Newcombe interval -0.586 to -0.153, a real drop
	got := FromMetric("api", 18, 30, 6, 30)
	if len(got) != 1 || got[0].Priority != "P0" {
		t.Fatalf("significant drop, got %+v", got)
	}
	if b, _ := got[0].Baseline["visibility"].(map[string]any); b["x"] != 18 || b["n"] != 30 {
		t.Fatalf("baseline = %v", got[0].Baseline)
	}
	// 6/20 -> 5/20: interval -0.309 to 0.218, noise
	if got := FromMetric("api", 6, 20, 5, 20); len(got) != 0 {
		t.Fatalf("noise, got %+v", got)
	}
}

func TestSortByPriorityThenSource(t *testing.T) {
	items := []Item{{Key: "b", Priority: "P2"}, {Key: "a", Priority: "P0", Source: "metric"}, {Key: "c", Priority: "P0", Source: "audit"}}
	Sort(items)
	if items[0].Key != "c" || items[1].Key != "a" || items[2].Key != "b" {
		t.Fatalf("order = %v %v %v", items[0].Key, items[1].Key, items[2].Key)
	}
}

func TestFromSearchDropsBrandedAndMergesVariants(t *testing.T) {
	b := Brand{Name: "acmecli", Site: "https://acmecli.example"}
	ops := []webstats.SearchOp{
		{Type: "striking_distance", Title: "acmecli github", Query: "acmecli github"},
		{Type: "low_ctr", Title: "acme cli", Query: "acme cli"},
		{Type: "low_ctr", Title: "docx to pdf cli", Query: "docx to pdf cli"},
		{Type: "low_ctr", Title: "cli docx to pdf", Query: "cli docx to pdf"},
		{Type: "low_ctr", Title: "docx-to-pdf CLI", Query: "docx-to-pdf CLI"},
		{Type: "cannibalization", Title: "word cli", Query: "word cli", URL: "https://a/1", URLs: []string{"https://a/1", "https://a/2"}},
	}
	got := FromSearch(ops, b)
	if len(got) != 2 {
		t.Fatalf("want the merged low_ctr item and the cannibalization item, got %+v", got)
	}
	if v, _ := got[0].Detail["variants"].([]string); len(v) != 2 {
		t.Fatalf("variants = %v", got[0].Detail["variants"])
	}
	if len(got[1].URLs) != 2 {
		t.Fatalf("cannibalization keeps every competing page, got %v", got[1].URLs)
	}
}

// The dashboard translates citation titles from the prompt text in Detail.
func TestFromCitationCarriesPrompt(t *testing.T) {
	items := FromCitation([]sample.Opportunity{
		{Category: "social", QID: "q001", Title: "Show up in the discussion: best tools", Prompts: []sample.OppRef{{QID: "q001", Text: "best tools"}}},
		{Category: "existing-content", Title: "Recover owned pages that stopped being cited"},
	}, "api", nil)
	if got := items[0].Detail["prompt"]; got != "best tools" {
		t.Fatalf("prompt = %v", got)
	}
	if _, ok := items[1].Detail["prompt"]; ok {
		t.Fatal("an item without a prompt must not carry one")
	}
}

func TestSearchObservationsCannotBecomeActions(t *testing.T) {
	ops := []webstats.SearchOp{{Type: "multiple_pages"}, {Type: "striking_distance", Reason: "small_sample"}, {Type: "striking_distance", Reason: "coverage"}, {Type: "striking_distance", Reason: "new_site"}, {Type: "striking_distance", Query: "search tool", Reason: "candidate", Facts: map[string]float64{"position": 8, "impressions": 900}}}
	got := FromSearch(ops, Brand{})
	if len(got) != 1 || got[0].Detail["reason"] != "candidate" || got[0].Detail["facts"].(map[string]float64)["impressions"] != 900 {
		t.Fatalf("%#v", got)
	}
}
