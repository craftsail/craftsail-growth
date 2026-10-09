// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "testing"

func TestSearchDiagnosticsSeparateActionsFromObservations(t *testing.T) {
	kws := []KeywordRow{{Query: "large", Position: 8, Impressions: 1000, Clicks: 10, CTR: 0.01}, {Query: "small", Position: 8, Impressions: 30}, {Query: "top low ctr", Position: 1, Impressions: 5000, CTR: 0.001}}
	pages := []PageClicks{{URL: "/sustained", Current: 0, Previous: 300, Sustained: true}, {URL: "/small", Current: 0, Previous: 10, Sustained: true}, {URL: "/burst", Current: 0, Previous: 300}}
	pairs := []QueryPage{{Query: "shared", Page: "/a", Impressions: 80}, {Query: "shared", Page: "/b", Impressions: 70}}
	policy := SearchPolicy{Mode: "established", MinImpressions: 500, QueriesCovered: true, PagesComparable: true, PairsCovered: true}
	actions, observations := SearchOpportunities(kws, pages, pairs, policy)
	if len(actions) != 2 || len(observations) != 2 {
		t.Fatalf("actions %#v observations %#v", actions, observations)
	}
	for _, op := range actions {
		if op.Type == "low_ctr" || op.Type == "cannibalization" || op.Type == "position_change" || op.Severity == "high" || op.URL == "/small" || op.URL == "/burst" {
			t.Fatalf("unsupported conclusion %#v", op)
		}
	}
	policy.Mode = "new_site"
	actions, observations = SearchOpportunities(kws, nil, nil, policy)
	if len(actions) != 0 || len(observations) != 2 {
		t.Fatalf("new site %#v %#v", actions, observations)
	}
	policy.Mode = "established"
	policy.QueriesCovered = false
	policy.PagesComparable = false
	actions, _ = SearchOpportunities(kws, pages, pairs, policy)
	if len(actions) != 0 {
		t.Fatalf("manual mode bypassed coverage %#v", actions)
	}
}
