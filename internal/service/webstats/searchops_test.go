// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "testing"

func TestSearchOpsFourKinds(t *testing.T) {
	kws := []KeywordRow{
		{Query: "acme vs salesforce", Position: 8, Impressions: 1000, Clicks: 10, CTR: 0.01},
		{Query: "acme login", Position: 1.2, Impressions: 500, Clicks: 20, CTR: 0.04},
	}
	pages := []PageClicks{{URL: "/blog/old", Current: 2, Previous: 40}}
	pairs := []QueryPage{
		{Query: "acme pricing", Page: "/pricing", Impressions: 80},
		{Query: "acme pricing", Page: "/plans", Impressions: 70},
	}
	got := SearchOpportunities(kws, pages, pairs)
	seen := map[string]bool{}
	for _, op := range got {
		seen[op.Type] = true
		if op.Detail == "" || op.Title == "" {
			t.Fatalf("empty copy %+v", op)
		}
	}
	if !seen["striking_distance"] || !seen["low_ctr"] || !seen["content_decay"] || !seen["cannibalization"] {
		t.Fatalf("%+v", got)
	}
}
