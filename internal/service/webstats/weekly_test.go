// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

// seedWeeklySearchData seeds 35 days (five full weeks ending on end) of GSC
// daily totals (10 clicks, 100 impressions per day) and query rows: one
// brand query and one non-brand query per day.
func seedWeeklySearchData(t *testing.T, s *Service, p *model.Project, prop string, start, end time.Time) {
	t.Helper()
	ctx := context.Background()
	err := s.syncReport(ctx, p.ID, "gsc", prop, "daily", "web", start, end, 35, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscDaily
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows, model.GscDaily{ProjectID: p.ID, Property: prop, SearchType: "web", Day: d, Clicks: 10, Impressions: 100})
		}
		return repo.SyncBatch{GSCDaily: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	err = s.syncReport(ctx, p.ID, "gsc", prop, "query", "web", start, end, 35, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscFact
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows,
				model.GscFact{ProjectID: p.ID, Property: prop, Slice: "query", SearchType: "web", Day: d, KeyHash: fmt.Sprintf("brand-%s", d.Format("0102")), Query: p.Name, Clicks: 3, Impressions: 30},
				model.GscFact{ProjectID: p.ID, Property: prop, Slice: "query", SearchType: "web", Day: d, KeyHash: fmt.Sprintf("nonbrand-%s", d.Format("0102")), Query: "best notes app", Clicks: 2, Impressions: 20},
			)
		}
		return repo.SyncBatch{GSC: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWeeklySearch(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "Weekly", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	end := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC) // Sunday
	start := end.AddDate(0, 0, -34)                     // five full weeks
	seedWeeklySearchData(t, s, p, prop, start, end)

	got, err := s.WeeklySearch(ctx, p, end)
	if err != nil {
		t.Fatal(err)
	}
	checkWeek := func(t *testing.T, w Week) {
		t.Helper()
		if w.Clicks == nil || *w.Clicks != 70 {
			t.Fatalf("clicks = %v, want 70", w.Clicks)
		}
		if w.Impressions == nil || *w.Impressions != 700 {
			t.Fatalf("impressions = %v, want 700", w.Impressions)
		}
		if w.NonBrandClicks == nil || *w.NonBrandClicks != 14 {
			t.Fatalf("non-brand clicks = %v, want 14", w.NonBrandClicks)
		}
		if w.NonBrandImpressions == nil || *w.NonBrandImpressions != 140 {
			t.Fatalf("non-brand impressions = %v, want 140", w.NonBrandImpressions)
		}
		if w.VisibleClicks == nil || *w.VisibleClicks != 35 {
			t.Fatalf("visible clicks = %v, want 35", w.VisibleClicks)
		}
	}
	checkWeek(t, got.Weeks[0]) // this week
	checkWeek(t, got.Weeks[2]) // four weeks ago
	for i, w := range got.Weeks {
		if w.OrganicSessions != nil || w.OrganicEngaged != nil || w.OrganicKeyEvents != nil || w.AISessions != nil {
			t.Fatalf("week %d: GA cells should be nil without GA4 data: %+v", i, w)
		}
		if w.NewPages != nil {
			t.Fatalf("week %d: new pages should be nil without any publish date: %+v", i, w)
		}
	}

	// A week past the imported data is unknown, not zero.
	got, err = s.WeeklySearch(ctx, p, end.AddDate(0, 0, 7))
	if err != nil {
		t.Fatal(err)
	}
	if got.Weeks[0].Clicks != nil {
		t.Fatalf("uncovered week clicks = %v, want nil", got.Weeks[0].Clicks)
	}
}

// TestWeeklySearchWeekAnchorUsesUTCDate pins the week anchor to through's
// UTC date, not its own location's weekday: a through value just after UTC
// midnight in a negative offset is still a Sunday locally, but the week
// must end on the UTC Sunday so start/end stay aligned with the data
// (which is keyed by UTC/Pacific dates, never the caller's zone).
func TestWeeklySearchWeekAnchorUsesUTCDate(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "WeekAnchor", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	// Oct 4 2026 23:00 in UTC-7 is Oct 5 06:00 UTC: a Sunday locally, but a
	// Monday in UTC. The complete week must still end on the UTC Sunday,
	// 2026-10-04.
	loc := time.FixedZone("UTC-7", -7*3600)
	through := time.Date(2026, 10, 4, 23, 0, 0, 0, loc)
	got, err := s.WeeklySearch(ctx, p, through)
	if err != nil {
		t.Fatal(err)
	}
	if got.Weeks[0].Through != "2026-10-04" || got.Weeks[0].From != "2026-09-28" {
		t.Fatalf("week anchor used the wrong date: from=%s through=%s, want from=2026-09-28 through=2026-10-04", got.Weeks[0].From, got.Weeks[0].Through)
	}
}

func TestWeeklySearchGAChannel(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "WeeklyGA", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	if err := s.projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	end := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC) // Sunday
	start := end.AddDate(0, 0, -34)
	seedWeeklySearchData(t, s, p, prop, start, end)

	weekStart := end.AddDate(0, 0, -6)
	err = s.syncReport(ctx, p.ID, "ga4", "123", "channel", "", weekStart, end, 7, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GaFact
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			rows = append(rows,
				model.GaFact{ProjectID: p.ID, Property: "123", Report: "channel", Day: d, KeyHash: fmt.Sprintf("organic-%s", d.Format("0102")), Channel: "Organic Search", Source: "google", Medium: "organic", Sessions: 20, Engaged: 10, KeyEvents: 2},
				model.GaFact{ProjectID: p.ID, Property: "123", Report: "channel", Day: d, KeyHash: fmt.Sprintf("ai-%s", d.Format("0102")), Channel: "Referral", Source: "chatgpt.com", Medium: "referral", Sessions: 5, Engaged: 3, KeyEvents: 1},
			)
		}
		return repo.SyncBatch{Quality: model.GoogleQuality{Known: true, TimeZones: []string{"UTC"}}, GA: rows}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.WeeklySearch(ctx, p, end)
	if err != nil {
		t.Fatal(err)
	}
	w := got.Weeks[0]
	if w.OrganicSessions == nil || *w.OrganicSessions != 140 { // 20 * 7 days
		t.Fatalf("organic sessions = %v, want 140", w.OrganicSessions)
	}
	if w.OrganicEngaged == nil || *w.OrganicEngaged != 70 { // 10 * 7 days
		t.Fatalf("organic engaged = %v, want 70", w.OrganicEngaged)
	}
	if w.OrganicKeyEvents == nil || *w.OrganicKeyEvents != 14 { // 2 * 7 days
		t.Fatalf("organic key events = %v, want 14", w.OrganicKeyEvents)
	}
	if w.AISessions == nil || *w.AISessions != 35 { // 5 * 7 days, chatgpt.com referral
		t.Fatalf("ai sessions = %v, want 35", w.AISessions)
	}
	// A week not synced for GA stays nil, even though GSC covers it.
	if got.Weeks[1].OrganicSessions != nil {
		t.Fatalf("unsynced GA week should be nil: %+v", got.Weeks[1])
	}
}
