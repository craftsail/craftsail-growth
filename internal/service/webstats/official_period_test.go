// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestPeriodFromOfficialUsesDateTotals(t *testing.T) {
	cur := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	prev := cur.AddDate(0, 0, -1)
	rows := []model.GscDaily{
		{Day: prev, SearchType: "web", Clicks: 10, Impressions: 100, Position: 4},
		{Day: cur, SearchType: "web", Clicks: 8, Impressions: 40, Position: 2},
		{Day: cur, SearchType: "image", Clicks: 50, Impressions: 80, Position: 1},
	}
	got := periodFromOfficial(rows, cur, cur, prev, prev)
	if got.Clicks != 8 || got.Impressions != 40 {
		t.Fatalf("clicks/impressions = %.0f/%.0f, want 8/40", got.Clicks, got.Impressions)
	}
	if !got.HasPrevious {
		t.Fatal("missing previous window")
	}
	if got.ClicksDelta == nil || *got.ClicksDelta != -20 {
		t.Fatalf("clicks delta %#v", got.ClicksDelta)
	}
	if got.Position != 2 {
		t.Fatalf("position %v", got.Position)
	}
}

func TestPeriodFromOfficialWeightsPositionByImpressions(t *testing.T) {
	a := time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)
	b := a.AddDate(0, 0, 1)
	rows := []model.GscDaily{
		{Day: a, SearchType: "web", Clicks: 1, Impressions: 100, Position: 1},
		{Day: b, SearchType: "web", Clicks: 1, Impressions: 100, Position: 3},
	}
	got := periodFromOfficial(rows, a, b, a.AddDate(0, 0, -2), a.AddDate(0, 0, -1))
	if got.Position != 2 {
		t.Fatalf("position %v, want 2", got.Position)
	}
	if got.HasPrevious {
		t.Fatal("empty previous window marked as present")
	}
}
