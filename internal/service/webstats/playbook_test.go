// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestVisibleShareWeeks(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "Share", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	end := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC) // Sunday
	start := end.AddDate(0, 0, -34)                     // five full weeks
	week := func(d time.Time) int { return int(d.Sub(start).Hours() / 24 / 7) }
	err = s.syncReport(ctx, p.ID, "gsc", prop, "daily", "web", start, end, 35, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscDaily
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows, model.GscDaily{ProjectID: p.ID, Property: prop, SearchType: "web", Day: d, Clicks: 10, Impressions: 100})
		}
		return repo.SyncBatch{GSCDaily: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Visible query clicks per day: 2, 3, 4, 5, 6 in weeks 1..5 → share 20%..60%.
	err = s.syncReport(ctx, p.ID, "gsc", prop, "query", "web", start, end, 35, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscFact
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows, model.GscFact{ProjectID: p.ID, Property: prop, Slice: "query", SearchType: "web", Day: d,
				KeyHash: fmt.Sprintf("q-%s", d.Format("0102")), Query: "q", Clicks: float64(2 + week(d))})
		}
		return repo.SyncBatch{GSC: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.visibleShare(ctx, p.ID, prop, end)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0.2, 0.3, 0.4, 0.5, 0.6}
	if len(got) != 5 {
		t.Fatalf("weeks = %d", len(got))
	}
	for i, w := range want {
		if got[i] == nil || *got[i] < w-1e-9 || *got[i] > w+1e-9 {
			t.Fatalf("week %d = %v, want %v", i, got[i], w)
		}
	}
	// A week without query coverage is unknown, not zero.
	got, err = s.visibleShare(ctx, p.ID, prop, end.AddDate(0, 0, 7))
	if err != nil || got[4] != nil {
		t.Fatalf("uncovered week = %v, %v", got[4], err)
	}

	// The week anchor must use through's UTC date, not the weekday of
	// through's own location: this value is a Saturday in UTC-7 but its UTC
	// date is still the Sunday "end" above, so it must anchor on the same
	// week.
	loc := time.FixedZone("UTC-7", -7*3600)
	tricky := time.Date(2026, 9, 26, 20, 0, 0, 0, loc) // UTC: 2026-09-27 03:00
	gotTricky, err := s.visibleShare(ctx, p.ID, prop, tricky)
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range want {
		if gotTricky[i] == nil || math.Abs(*gotTricky[i]-w) > 1e-9 {
			t.Fatalf("week anchor used through's own weekday instead of its UTC date: week %d = %v, want %v", i, gotTricky[i], w)
		}
	}
}
