// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestOfficialPlanStartsWithRecentWindow(t *testing.T) {
	through := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	history := through.AddDate(0, -16, 0)
	parts := officialPlan(time.Time{}, time.Time{}, through, history)
	if len(parts) != 1 {
		t.Fatalf("parts %#v", parts)
	}
	if parts[0].end.Format("2006-01-02") != "2026-09-21" {
		t.Fatalf("end %s", parts[0].end.Format("2006-01-02"))
	}
	if parts[0].start.Before(through.AddDate(0, 0, -60)) {
		t.Fatalf("first sync reached %s", parts[0].start.Format("2006-01-02"))
	}
}

func TestOfficialPlanContinuesBackward(t *testing.T) {
	through := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	history := through.AddDate(0, -16, 0)
	earliest := through.AddDate(0, 0, -55)
	parts := officialPlan(through, earliest, through, history)
	if len(parts) < 2 {
		t.Fatalf("want tail plus older month, %#v", parts)
	}
	older := parts[len(parts)-1]
	if !older.end.Before(earliest) {
		t.Fatalf("older end %s should be before %s", older.end.Format("2006-01-02"), earliest.Format("2006-01-02"))
	}
}

func TestSyncOfficialStoresDateTotalsNotQuerySums(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	svc := New(db)
	svc.FetchGSCDate = func(ctx context.Context, token, site, start, end string) ([]model.GscFact, string, int, bool, error) {
		startDay, _ := time.Parse("2006-01-02", start)
		endDay, _ := time.Parse("2006-01-02", end)
		if day.Before(startDay) || day.After(endDay) {
			return nil, "2026-09-22", 0, false, nil
		}
		return []model.GscFact{{
			Slice: "date", SearchType: "web", Day: day, Clicks: 9, Impressions: 40, CTR: 0.2, Position: 3,
		}}, "2026-09-22", 1, false, nil
	}
	note := svc.SyncOfficial(ctx, 7, "token", "Example.com", "", now)
	if note != "" {
		t.Fatal(note)
	}
	rows, err := svc.rows.ListGscDaily(ctx, 7, "sc-domain:example.com", day, day)
	if err != nil || len(rows) != 1 || rows[0].Clicks != 9 {
		t.Fatalf("dailies %#v %v", rows, err)
	}
	imp, err := svc.rows.GetImport(ctx, 7, "gsc")
	if err != nil || imp == nil || imp.BoundarySource != "metadata" || imp.PausedReason != "" {
		t.Fatalf("import %#v %v", imp, err)
	}
	if imp.FinalizedThrough == nil || imp.FinalizedThrough.UTC().Format("2006-01-02") != "2026-09-21" {
		t.Fatalf("through %#v", imp.FinalizedThrough)
	}
	win, err := svc.rows.LatestWindow(ctx, 7, "gsc", "sc-domain:example.com")
	if err != nil || win == nil || win.Clicks != 9 || win.Impressions != 40 {
		t.Fatalf("window %#v %v", win, err)
	}
}

func TestSnapshotReportsImportState(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	t.Setenv("GOOGLE_SA_JSON", `{"client_email":"a@b.c","private_key":"x"}`)
	p, err := project.New(db).Create(ctx, project.CreateInput{
		Name: "Acme", Slug: "snap", NoSite: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.GscSite = "sc-domain:acme.com"
	if err := project.New(db).Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	through := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	if err := (&repo.Webstats{DB: db}).UpsertImport(ctx, &model.WebImport{
		ProjectID: p.ID, Source: "gsc", Property: "sc-domain:acme.com",
		State: "completed", FinalizedThrough: &through, BoundarySource: "metadata",
	}); err != nil {
		t.Fatal(err)
	}
	snap, err := New(db).Snapshot(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Sources) != 2 || snap.Sources[0].State != "ready" || snap.Sources[0].Through != "2026-09-20" {
		t.Fatalf("sources %#v", snap.Sources)
	}
	if snap.Sources[0].Calendar != "Pacific time" {
		t.Fatalf("calendar %q", snap.Sources[0].Calendar)
	}
}

func TestSyncOfficialMarksReauth(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	svc := New(db)
	svc.FetchGSCDate = func(ctx context.Context, token, site, start, end string) ([]model.GscFact, string, int, bool, error) {
		return nil, "", 0, false, errString("gsc HTTP 401 invalid_grant")
	}
	note := svc.SyncOfficial(ctx, 3, "token", "sc-domain:a.com", "", time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC))
	if note != "Reconnect Google" {
		t.Fatalf("note %q", note)
	}
	imp, err := svc.rows.GetImport(ctx, 3, "gsc")
	if err != nil || imp == nil || imp.PausedReason != "needs_reauth" {
		t.Fatalf("import %#v %v", imp, err)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
