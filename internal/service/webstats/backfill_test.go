// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"testing"
	"time"
)

func TestCoverageResumesGapsAndEmptyDays(t *testing.T) {
	ctx := context.Background()
	svc := New(testDB(t))
	through := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	from := through.AddDate(0, 0, -19)
	calls := 0
	fetch := func(context.Context, dateChunk) (repo.SyncBatch, error) {
		calls++
		if calls == 2 {
			return repo.SyncBatch{}, errors.New("network interrupted")
		}
		return repo.SyncBatch{}, nil
	}
	if err := svc.syncReport(ctx, 1, "gsc", "sc-domain:a.com", "country", "web", from, through, 7, 2, fetch); err == nil {
		t.Fatal("expected interruption")
	}
	reports, _ := svc.rows.SyncReports(ctx, 1, "gsc", "sc-domain:a.com")
	days, _ := svc.rows.SyncDays(ctx, reports[0].ID, from, through)
	if len(days) != 7 || reports[0].State != "failed" {
		t.Fatalf("coverage %d report %#v", len(days), reports)
	}
	var parts []dateChunk
	if err := svc.syncReport(ctx, 1, "gsc", "sc-domain:a.com", "country", "web", from, through, 7, 3, func(_ context.Context, p dateChunk) (repo.SyncBatch, error) {
		parts = append(parts, p)
		return repo.SyncBatch{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(parts) != 3 || !parts[1].end.Equal(through.AddDate(0, 0, -7)) {
		t.Fatalf("gap not resumed: %#v", parts)
	}
	reports, _ = svc.rows.SyncReports(ctx, 1, "gsc", "sc-domain:a.com")
	days, _ = svc.rows.SyncDays(ctx, reports[0].ID, from, through)
	if len(days) != 20 || reports[0].State != "completed" {
		t.Fatalf("coverage %d report %#v", len(days), reports)
	}
	other, _ := svc.rows.SyncReports(ctx, 1, "gsc", "sc-domain:b.com")
	if len(other) != 0 {
		t.Fatal("coverage leaked across properties")
	}
}

func TestCoveragePlanRecentFirstAndDowntime(t *testing.T) {
	through := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	from := through.AddDate(0, -16, 0)
	parts := coveragePlan(nil, from, through, 28, 2)
	if len(parts) != 2 || !parts[0].end.Equal(through) || !parts[1].start.Equal(through.AddDate(0, 0, -55)) {
		t.Fatalf("initial %#v", parts)
	}
	var covered []model.WebSyncDay
	for d := through.AddDate(0, 0, -90); !d.After(through.AddDate(0, 0, -30)); d = d.AddDate(0, 0, 1) {
		covered = append(covered, model.WebSyncDay{Day: d})
	}
	parts = coveragePlan(covered, from, through, 28, 2)
	if !parts[1].end.Equal(through.AddDate(0, 0, -28)) || !parts[1].start.Equal(through.AddDate(0, 0, -29)) {
		t.Fatalf("downtime gap %#v", parts)
	}
}

func TestSyncCancellationIsPersisted(t *testing.T) {
	svc := New(testDB(t))
	ctx, cancel := context.WithCancel(context.Background())
	day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	err := svc.syncReport(ctx, 1, "ga4", "properties/1", "session", "", day, day, 7, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		cancel()
		return repo.SyncBatch{}, context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	rows, _ := svc.rows.SyncReports(context.Background(), 1, "ga4", "properties/1")
	if len(rows) != 1 || rows[0].State != "paused" {
		t.Fatalf("%#v", rows)
	}
}

func TestSyncIncompleteAndUnsupportedAreNotCoverage(t *testing.T) {
	for _, tt := range []struct {
		name      string
		err       error
		state     string
		wantError bool
	}{
		{"incomplete", ErrIncompleteReport, "partial", true},
		{"restricted", ErrMetricRestricted, "partial", true},
		{"unsupported", ErrSliceUnsupported, "unsupported", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(testDB(t))
			ctx := context.Background()
			day := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
			err := svc.syncReport(ctx, 1, "gsc", "sc-domain:a.com", "appearance", "web", day, day, 1, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) { return repo.SyncBatch{}, tt.err })
			if (err != nil) != tt.wantError {
				t.Fatalf("error %v", err)
			}
			progress, err := svc.syncProgress(ctx, 1, "gsc", "sc-domain:a.com")
			if err != nil || len(progress) != 1 || progress[0].State != tt.state || progress[0].CoveredDays != 0 {
				t.Fatalf("progress %#v %v", progress, err)
			}
		})
	}
}
