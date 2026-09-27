// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import "testing"

func TestReportedWebQueriesUsesSentinel(t *testing.T) {
	if len(ReportedWebQueries(false, nil)) != 0 {
		t.Fatal("no search should store nothing")
	}
	got := ReportedWebQueries(true, nil)
	if len(got) != 1 || got[0] != WebQueriesUnavailable {
		t.Fatalf("%v", got)
	}
	got = ReportedWebQueries(true, []string{"best crm", "best crm", "  "})
	if len(got) != 1 || got[0] != "best crm" {
		t.Fatalf("%v", got)
	}
}

func TestClassifyDomainCNAndOwn(t *testing.T) {
	own := "example.com"
	comps := []string{"rival.cn"}
	if got := ClassifyDomain("www.zhihu.com", own, comps); got != "social" {
		t.Fatalf("zhihu %s", got)
	}
	if got := ClassifyDomain("blog.example.com", own, comps); got != "brand" {
		t.Fatalf("own %s", got)
	}
	if got := ClassifyDomain("rival.cn", own, comps); got != "competitor" {
		t.Fatalf("rival %s", got)
	}
	if got := ClassifyDomain("en.wikipedia.org", own, comps); got != "reference" {
		t.Fatalf("wiki %s", got)
	}
	if got := ClassifyDomain("unknown.example.org", own, comps); got != "other" {
		t.Fatalf("other %s", got)
	}
}

func TestPageTypeFromPath(t *testing.T) {
	if PageType("https://example.com/") != "homepage" {
		t.Fatal(PageType("https://example.com/"))
	}
	if PageType("https://example.com/a-vs-b") != "comparison" {
		t.Fatal(PageType("https://example.com/a-vs-b"))
	}
	if PageType("https://www.bilibili.com/video/BV1") != "video" {
		t.Fatal(PageType("https://www.bilibili.com/video/BV1"))
	}
}

func TestStabilityNeedsTwoDays(t *testing.T) {
	if _, label := Stability(nil); label != "n/a" {
		t.Fatalf("label=%s", label)
	}
	_, label := Stability([]DayDomains{
		{Date: "2026-09-01", Counts: map[string]int{"a.com": 9, "b.com": 1}},
		{Date: "2026-09-02", Counts: map[string]int{"a.com": 9, "b.com": 1}},
	})
	if label != "locked-in" {
		t.Fatalf("stable label=%s", label)
	}
	_, label = Stability([]DayDomains{
		{Date: "2026-09-01", Counts: map[string]int{"a.com": 5}},
		{Date: "2026-09-02", Counts: map[string]int{"b.com": 5}},
	})
	if label != "wide-open" {
		t.Fatalf("churn label=%s", label)
	}
}

func TestNextRoundsSkipsFilled(t *testing.T) {
	if rounds := NextRounds(0, 3); len(rounds) != 3 || rounds[0] != 1 || rounds[2] != 3 {
		t.Fatalf("%v", rounds)
	}
	if rounds := NextRounds(2, 3); len(rounds) != 1 || rounds[0] != 3 {
		t.Fatalf("%v", rounds)
	}
	if len(NextRounds(3, 3)) != 0 {
		t.Fatal("full day should not run again")
	}
}

func TestQueriesFromPayloadIgnoresRelated(t *testing.T) {
	got := QueriesFromPayload(map[string]any{
		"related_queries": []any{"do not use"},
		"search_info":     map[string]any{"search_queries": []any{"best crm 2026", ""}},
	})
	if len(got) != 1 || got[0] != "best crm 2026" {
		t.Fatalf("%v", got)
	}
}
