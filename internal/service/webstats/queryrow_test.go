// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestQueryRowsMergeDaysAndWeightPosition(t *testing.T) {
	facts := []model.GscFact{
		{Query: "best cli", Page: "https://e.com/a", Clicks: 2, Impressions: 10, Position: 3},
		{Query: "best cli", Page: "https://e.com/a", Clicks: 1, Impressions: 30, Position: 7},
		{Query: "cli", Page: "https://e.com/b", Clicks: 0, Impressions: 5, Position: 12},
	}
	rows := QueryRowsFromFacts(facts)
	if len(rows) != 2 {
		t.Fatalf("rows = %+v", rows)
	}
	r := rows[0]
	if r.Query != "best cli" || r.Clicks != 3 || r.Impressions != 40 {
		t.Fatalf("merged = %+v", r)
	}
	if r.Position != 6 {
		t.Fatalf("impression-weighted position = %v", r.Position)
	}
}
