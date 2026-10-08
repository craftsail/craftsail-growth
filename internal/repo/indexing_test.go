// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestInspectionHistoryFailureIsolationAndRollback(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	a := model.IndexURL{ProjectID: 1, Property: "sc-domain:a.com", URL: "https://a.com/p", FromCrawl: true, FirstSeenAt: now.Unix(), LastSeenAt: now.Unix()}
	b := a
	b.Property = "https://a.com/"
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{a, b}); err != nil {
		t.Fatal(err)
	}
	due, _ := r.DueIndexURLs(ctx, 1, a.Property, now.Unix(), 30)
	if len(due) != 1 {
		t.Fatal(due)
	}
	a = due[0]
	result := &model.GscIndex{Verdict: "PASS", GoogleCanonical: a.URL, UserCanonical: a.URL, Raw: `{"provider":"evidence"}`}
	if err := r.RecordInspection(ctx, a, result, "", now); err != nil {
		t.Fatal(err)
	}
	if err := r.RecordInspection(ctx, a, nil, "gsc HTTP 500", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	// Re-discovery preserves inspection, timestamps and both discovery sources.
	rediscovered := a
	rediscovered.FromCrawl = false
	rediscovered.FromSearch = true
	rediscovered.FirstSeenAt = now.Add(time.Hour).Unix()
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{rediscovered}); err != nil {
		t.Fatal(err)
	}
	inventory, err := r.IndexInventory(ctx, 1, a.Property, "", "error", 1, 50, now.Unix())
	if err != nil || len(inventory.Items) != 1 {
		t.Fatalf("%#v %v", inventory, err)
	}
	got := inventory.Items[0]
	if !got.FromCrawl || !got.FromSearch || got.FirstSeenAt != now.Unix() || got.Verdict != "PASS" || got.FirstIndexedAt == nil || *got.FirstIndexedAt != now.Unix() || got.LastSuccessAt != now.Unix() || got.Latest.GoogleCanonical != a.URL {
		t.Fatalf("%#v", got)
	}
	hist, err := r.IndexHistory(ctx, 1, a.Property, a.URL, 1, 1)
	if err != nil || hist.Total != 2 || len(hist.Items) != 1 || hist.Items[0].Error == "" || hist.Items[0].Result != nil {
		t.Fatalf("%#v %v", hist, err)
	}
	other, _ := r.IndexHistory(ctx, 1, b.Property, a.URL, 1, 50)
	if other.Total != 0 {
		t.Fatal("property leaked")
	}
	other, _ = r.IndexHistory(ctx, 2, a.Property, a.URL, 1, 50)
	if other.Total != 0 {
		t.Fatal("project leaked")
	}
	latest, _ := r.ListIndex(ctx, 1, a.Property)
	if len(latest) != 1 || latest[0].Verdict != "PASS" || latest[0].Raw == "" {
		t.Fatal(latest)
	}
	// A failed history write rolls back the latest-result overwrite as well.
	if err := r.DB.Exec("CREATE TRIGGER reject_inspection BEFORE INSERT ON index_inspections BEGIN SELECT RAISE(FAIL, 'injected failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if err := r.RecordInspection(ctx, a, &model.GscIndex{Verdict: "FAIL"}, "", now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected rollback")
	}
	latest, _ = r.ListIndex(ctx, 1, a.Property)
	if latest[0].Verdict != "PASS" {
		t.Fatal("partial write")
	}
	hist, _ = r.IndexHistory(ctx, 1, a.Property, a.URL, 1, 50)
	if hist.Total != 2 {
		t.Fatal("partial history")
	}
}

func TestInspectionIntervalsAndFirstObservation(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	row := model.IndexURL{ProjectID: 1, Property: "sc-domain:a.com", URL: "https://a.com/a", FromCrawl: true, FirstSeenAt: now.Unix()}
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{row}); err != nil {
		t.Fatal(err)
	}
	due, _ := r.DueIndexURLs(ctx, 1, row.Property, now.Unix(), 1)
	row = due[0]
	for _, tc := range []struct {
		verdict string
		delay   time.Duration
	}{{"FAIL", 24 * time.Hour}, {"PASS", 7 * 24 * time.Hour}, {"", 6 * time.Hour}} {
		var result *model.GscIndex
		failure := "unavailable"
		if tc.verdict != "" {
			result = &model.GscIndex{Verdict: tc.verdict}
			failure = ""
		}
		if err := r.RecordInspection(ctx, row, result, failure, now); err != nil {
			t.Fatal(err)
		}
		before, _ := r.DueIndexURLs(ctx, 1, row.Property, now.Add(tc.delay-time.Second).Unix(), 1)
		after, _ := r.DueIndexURLs(ctx, 1, row.Property, now.Add(tc.delay).Unix(), 1)
		if len(before) != 0 || len(after) != 1 {
			t.Fatalf("interval %s", tc.verdict)
		}
		if tc.verdict == "FAIL" && after[0].FirstIndexedAt != nil {
			t.Fatal("fabricated first indexed time")
		}
	}
}

func TestIndexLegacyRowsAndLiteralFilters(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	// Pre-feature rows have no proven property. Preserve them without assigning one.
	legacy := model.GscIndex{ProjectID: 1, URL: "https://a.com/Doc/p", KeyHash: model.RowKey("https://a.com/Doc/p"), Verdict: "PASS"}
	if err := r.DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertIndex(ctx, []model.GscIndex{{ProjectID: 1, Property: "https://a.com/Doc/", URL: legacy.URL, Verdict: "FAIL"}, {ProjectID: 1, Property: "https://a.com/doc/", URL: legacy.URL, Verdict: "NEUTRAL"}}); err != nil {
		t.Fatal(err)
	}
	rows, err := r.ListIndex(ctx, 1, "https://a.com/Doc/")
	if err != nil || len(rows) != 1 || rows[0].Verdict != "FAIL" {
		t.Fatalf("%#v %v", rows, err)
	}
	var n int64
	r.DB.Model(&model.GscIndex{}).Count(&n)
	if n != 3 {
		t.Fatal("legacy result overwritten")
	}
	urls := []model.IndexURL{{ProjectID: 1, Property: "https://a.com/Doc/", URL: "https://a.com/Doc/a_b", FromCrawl: true}, {ProjectID: 1, Property: "https://a.com/Doc/", URL: "https://a.com/Doc/acb", FromCrawl: true}, {ProjectID: 1, Property: "https://a.com/doc/", URL: "https://a.com/doc/a_b", FromCrawl: true}}
	if err := r.DiscoverIndexURLs(ctx, urls); err != nil {
		t.Fatal(err)
	}
	out, err := r.IndexInventory(ctx, 1, "https://a.com/Doc/", "a_b", "", 1, 50, 1800000000)
	if err != nil || out.Known != 2 || out.Total != 1 {
		t.Fatalf("literal filter %#v %v", out, err)
	}
}
