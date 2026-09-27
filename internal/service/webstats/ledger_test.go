// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

func TestGSCPropertyKey(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"sc-domain:Example.com/", "sc-domain:example.com", true},
		{"https://Example.com/path", "https://example.com/path/", true},
		{"https://example.com/", "https://example.com/", true},
		{"example.com", "sc-domain:example.com", true},
		{"", "", false},
		{"not a property", "", false},
	}
	for _, tc := range cases {
		got, ok := GSCPropertyKey(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("GSCPropertyKey(%q) = %q %v, want %q %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestGAPropertyKey(t *testing.T) {
	id, ok := GAPropertyKey("properties/123456")
	if !ok || id != "123456" {
		t.Fatalf("got %q %v", id, ok)
	}
	if _, ok := GAPropertyKey("G-ABC"); ok {
		t.Fatal("measurement id accepted")
	}
	if _, ok := GAPropertyKey("UA-1-2"); ok {
		t.Fatal("legacy id accepted")
	}
}

func TestFinalizedThroughUsesMetadata(t *testing.T) {
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	through, source := FinalizedThrough(now, "2026-09-22")
	if through.Format("2006-01-02") != "2026-09-21" || source != "metadata" {
		t.Fatalf("through %s source %s", through.Format("2006-01-02"), source)
	}
}

func TestFinalizedThroughFallsBackTwoPacificDays(t *testing.T) {
	now := time.Date(2026, 9, 23, 6, 0, 0, 0, time.UTC)
	through, source := FinalizedThrough(now, "")
	if through.Format("2006-01-02") != "2026-09-20" || source != "fallback" {
		t.Fatalf("through %s source %s", through.Format("2006-01-02"), source)
	}
}

func TestClassifyGoogle(t *testing.T) {
	class, label := ClassifyGoogle(401, "invalid_grant")
	if class != "needs_reauth" || label != "Reconnect Google" {
		t.Fatalf("reauth %s %s", class, label)
	}
	class, label = ClassifyGoogle(429, "Quota exceeded")
	if class != "rate_limited" || label != "Waiting for Google quota; will retry later" {
		t.Fatalf("quota %s %s", class, label)
	}
	class, label = ClassifyGoogle(400, "Invalid site URL")
	if class != "config_invalid" || label != "Choose the property again" {
		t.Fatalf("config %s %s", class, label)
	}
	class, label = ClassifyGoogle(400, "Found duplicate metrics: conversions")
	if class != "provider" || label == "Choose the property again" {
		t.Fatalf("duplicate metric classified as property error: %s %s", class, label)
	}
}

func TestOfficialDailiesIgnoreQueryRows(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := GSCDailiesFromFacts("sc-domain:a.com", []model.GscFact{
		{Slice: "query_page", SearchType: "web", Day: day, Clicks: 1, Impressions: 10},
		{Slice: "date", SearchType: "web", Day: day, Clicks: 9, Impressions: 100, CTR: 0.09, Position: 4},
		{Slice: "date", SearchType: "image", Day: day, Clicks: 3, Impressions: 8},
	})
	if len(rows) != 1 || rows[0].Clicks != 9 || rows[0].Impressions != 100 || rows[0].Property != "sc-domain:a.com" {
		t.Fatalf("%#v", rows)
	}
}

func TestWindowUsesDateTotalsOnly(t *testing.T) {
	through := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	var rows []model.GscDaily
	for i := 0; i < 56; i++ {
		rows = append(rows, model.GscDaily{
			Day:         through.AddDate(0, 0, -i),
			Clicks:      2,
			Impressions: 10,
		})
	}
	win := WindowFromGSC(rows, through, 28)
	if win.Clicks != 56 || win.PreviousClicks != 56 || win.CoveredDays != 28 {
		t.Fatalf("%+v", win)
	}
}

func TestPresentSource(t *testing.T) {
	view := PresentSource(SourceInput{})
	if view.State != "not_connected" || view.Label != "Not connected" {
		t.Fatalf("%+v", view)
	}
	view = PresentSource(SourceInput{Connected: true})
	if view.State != "choose_property" || view.Label != "Choose a property" {
		t.Fatalf("%+v", view)
	}
	view = PresentSource(SourceInput{Connected: true, Property: "sc-domain:a.com", PausedReason: "needs_reauth"})
	if view.State != "needs_reauth" || !view.CanReconnect {
		t.Fatalf("%+v", view)
	}
	view = PresentSource(SourceInput{Connected: true, Property: "sc-domain:a.com", PausedReason: "rate_limited"})
	if view.State != "rate_limited" || view.CanReconnect {
		t.Fatalf("%+v", view)
	}
	through := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	view = PresentSource(SourceInput{
		Connected: true, Property: "sc-domain:a.com", State: "completed", Through: &through, Calendar: "Pacific time",
	})
	if view.State != "ready" || view.Through != "2026-09-20" || view.Calendar != "Pacific time" {
		t.Fatalf("%+v", view)
	}
}

func TestSwitchPropertyKeepsHistory(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	rows := &repo.Webstats{DB: db}
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if err := rows.ActivateProperty(ctx, 1, "gsc", "sc-domain:old.com", ""); err != nil {
		t.Fatal(err)
	}
	if err := rows.UpsertGscDaily(ctx, []model.GscDaily{{
		ProjectID: 1, Property: "sc-domain:old.com", SearchType: "web", Day: day, Clicks: 4, Impressions: 10, FetchedAt: day.Unix(),
	}}); err != nil {
		t.Fatal(err)
	}
	if err := rows.ActivateProperty(ctx, 1, "gsc", "sc-domain:new.com", ""); err != nil {
		t.Fatal(err)
	}
	active, err := rows.ActiveProperty(ctx, 1, "gsc")
	if err != nil || active != "sc-domain:new.com" {
		t.Fatalf("active %q %v", active, err)
	}
	old, err := rows.ListGscDaily(ctx, 1, "sc-domain:old.com", day, day)
	if err != nil || len(old) != 1 || old[0].Clicks != 4 {
		t.Fatalf("old history %#v %v", old, err)
	}
	cur, err := rows.ListGscDaily(ctx, 1, "sc-domain:new.com", day, day)
	if err != nil || len(cur) != 0 {
		t.Fatalf("new property should not inherit old rows: %#v %v", cur, err)
	}
}
