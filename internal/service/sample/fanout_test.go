// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import "testing"

func TestSummarizeQueriesDropsSentinelAndVerbatim(t *testing.T) {
	snap := SummarizeQueries([]struct {
		Question string
		Queries  []string
	}{
		{Question: "有什么好用的 CRM", Queries: []string{"unavailable", "有什么好用的 CRM", "最好的 CRM 2026"}},
	})
	if snap.Unavailable != 1 {
		t.Fatalf("unavailable %d", snap.Unavailable)
	}
	if len(snap.Queries) != 1 || snap.Queries[0].Word != "最好的 CRM 2026" {
		t.Fatalf("%+v", snap.Queries)
	}
	found := false
	for _, w := range snap.Added {
		if w.Word == "2026" {
			found = true
		}
	}
	if !found {
		t.Fatalf("added %+v", snap.Added)
	}
}
