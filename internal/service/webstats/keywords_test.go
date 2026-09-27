// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"testing"
	"time"
)

func TestAggregateKeywordsWeightsPositionByImpressions(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	_ = day
	rows := []QueryRow{
		{Query: "acme login", Page: "/login", Clicks: 10, Impressions: 100, Position: 1},
		{Query: "acme login", Page: "/home", Clicks: 0, Impressions: 100, Position: 9},
	}
	got := AggregateKeywords(rows)
	if len(got) != 1 || got[0].Query != "acme login" {
		t.Fatalf("%+v", got)
	}
	if got[0].Clicks != 10 || got[0].Impressions != 200 {
		t.Fatalf("totals %+v", got[0])
	}
	if got[0].Position != 5 {
		t.Fatalf("position %v", got[0].Position)
	}
	if got[0].CTR != 0.05 {
		t.Fatalf("ctr %v", got[0].CTR)
	}
	if got[0].Page != "/login" && got[0].Page != "/home" {
		t.Fatalf("page %s", got[0].Page)
	}
	if PositionBand(1) != "top3" || PositionBand(8) != "top10" || PositionBand(15) != "top20" || PositionBand(30) != "deep" {
		t.Fatal("bands")
	}
}
