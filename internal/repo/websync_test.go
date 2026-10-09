// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
	"time"
)

func TestSyncReplacementScopeEmptyAndSuperseded(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	report := model.WebSyncReport{ProjectID: 1, Property: "sc-domain:a.com", Source: "gsc", Report: "query_page", SearchType: "web", Version: 1, Token: "old", From: day, Through: day}
	if err := r.BeginSyncReport(ctx, &report); err != nil {
		t.Fatal(err)
	}
	fact := model.GscFact{ProjectID: 1, Property: report.Property, Slice: report.Report, SearchType: "web", Day: day, Query: "old", Clicks: 4}
	other := fact
	other.Property = "sc-domain:b.com"
	if err := r.UpsertGscFacts(ctx, []model.GscFact{other}); err != nil {
		t.Fatal(err)
	}
	if err := r.ReplaceSyncBatch(ctx, report, day, day, SyncBatch{GSC: []model.GscFact{fact}}); err != nil {
		t.Fatal(err)
	}
	fact.Query = "new"
	fact.Clicks = 7
	if err := r.ReplaceSyncBatch(ctx, report, day, day, SyncBatch{GSC: []model.GscFact{fact}}); err != nil {
		t.Fatal(err)
	}
	var rows []model.GscFact
	r.DB.Where("property = ?", report.Property).Find(&rows)
	if len(rows) != 1 || rows[0].Query != "new" {
		t.Fatalf("stale rows %#v", rows)
	}
	old := report
	report.Token = "new"
	if err := r.BeginSyncReport(ctx, &report); err != nil {
		t.Fatal(err)
	}
	if err := r.ReplaceSyncBatch(ctx, old, day, day, SyncBatch{}); !errors.Is(err, ErrSyncSuperseded) {
		t.Fatalf("stale write %v", err)
	}
	if err := r.FinishSyncReport(ctx, old, "completed", ""); !errors.Is(err, ErrSyncSuperseded) {
		t.Fatalf("stale finish %v", err)
	}
	if err := r.ReplaceSyncBatch(ctx, report, day, day, SyncBatch{}); err != nil {
		t.Fatal(err)
	}
	rows = nil
	r.DB.Find(&rows)
	if len(rows) != 1 || rows[0].Property != other.Property {
		t.Fatalf("scope %#v", rows)
	}
	days, err := r.SyncDays(ctx, report.ID, day, day)
	if err != nil || len(days) != 1 {
		t.Fatalf("empty coverage %#v %v", days, err)
	}
}

func TestSyncReplacementRollsBackOnInsertFailure(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	report := model.WebSyncReport{ProjectID: 1, Property: "properties/1", Source: "ga4", Report: "session", Version: 1, Token: "one", From: day, Through: day}
	if err := r.BeginSyncReport(ctx, &report); err != nil {
		t.Fatal(err)
	}
	fact := model.GaFact{ProjectID: 1, Property: report.Property, Report: report.Report, Day: day, Source: "old", Sessions: 3}
	if err := r.UpsertGaFacts(ctx, []model.GaFact{fact}); err != nil {
		t.Fatal(err)
	}
	if err := r.DB.Exec("CREATE TRIGGER reject_fact BEFORE INSERT ON ga_facts BEGIN SELECT RAISE(FAIL, 'injected failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	fact.Source = "new"
	if err := r.ReplaceSyncBatch(ctx, report, day, day, SyncBatch{GA: []model.GaFact{fact}}); err == nil {
		t.Fatal("expected failure")
	}
	var rows []model.GaFact
	r.DB.Find(&rows)
	days, _ := r.SyncDays(ctx, report.ID, day, day)
	if len(rows) != 1 || rows[0].Source != "old" || len(days) != 0 {
		t.Fatalf("rollback rows %#v days %#v", rows, days)
	}
}
