// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"net/http"
	"testing"
	"time"
)

func TestGAQualityAcrossPagesAndMetricPersistence(t *testing.T) {
	calls := 0
	c := &Client{GALimit: 1, HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (int, string) {
		calls++
		if calls == 1 {
			return 200, `{"rowCount":1,"metadata":{"currencyCode":"USD","timeZone":"America/New_York","subjectToThresholding":true,"dataLossFromOtherRow":true,"samplingMetadatas":[{"samplesReadCount":"5","samplingSpaceSize":"10"}],"schemaRestrictionResponse":{"activeMetricRestrictions":[{"metricName":"totalRevenue"}]}},"rows":[{"dimensionValues":[{"value":"20260901"}],"metricValues":[{"value":"2"},{"value":"1"},{"value":"1"},{"value":"1"},{"value":"3"},{"value":"0"},{"value":"0"},{"value":"0.5"},{"value":"0.5"},{"value":"12.5"}]}]}`
		}
		return 200, `{"metadata":{"currencyCode":"USD","timeZone":"America/New_York"},"rows":[]}`
	})}}
	rows, q, err := c.FetchGAReport(context.Background(), "token", "properties/1", "session", "2026-09-01", "2026-09-01")
	if err != nil || len(rows) != 1 || rows[0].EngagementDuration == nil || *rows[0].EngagementDuration != 12.5 {
		t.Fatalf("rows %#v err %v", rows, err)
	}
	if !q.Known || !q.Sampled || !q.Thresholded || !q.OtherRow || !q.Restricted || len(q.Currencies) != 1 {
		t.Fatalf("metadata %#v", q)
	}
	svc := New(testDB(t))
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	rows[0].ProjectID = 1
	rows[0].Property = "1"
	if err := svc.syncReport(context.Background(), 1, "ga4", "1", "session", "", day, day, 1, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		return repo.SyncBatch{GA: rows, Quality: q}, nil
	}); err != nil {
		t.Fatal(err)
	}
	progress, err := svc.syncProgress(context.Background(), 1, "ga4", "1")
	if err != nil || len(progress) != 1 || !progress[0].Quality.Sampled || !progress[0].Quality.Thresholded {
		t.Fatalf("persisted %#v %v", progress, err)
	}
	stored, err := svc.rows.ListGaSessionFacts(context.Background(), 1, "1", day, day)
	if err != nil || len(stored) != 1 || stored[0].EngagementDuration == nil || *stored[0].EngagementDuration != 12.5 || stored[0].EventValue != nil {
		t.Fatalf("stored %#v %v", stored, err)
	}
}

func TestGAUnavailableAndZeroMetricsDiffer(t *testing.T) {
	spec, _ := gaFactSpecByName("event")
	missing := factFromGA("event", spec, gaAPIRow{})
	zero := factFromGA("event", spec, gaAPIRow{MetricValues: []gaValue{{Value: "1"}, {Value: "0"}}})
	if missing.EventValue != nil || zero.EventValue == nil || *zero.EventValue != 0 {
		t.Fatalf("missing %#v zero %#v", missing, zero)
	}
	// A legacy row must retain unknown values through migration.
	db := testDB(t)
	if err := db.Create(&model.GaFact{ProjectID: 1, Property: "1", Report: "event", Day: time.Now(), KeyHash: "old"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"event_value", "engagement_duration"} {
		if err := db.Migrator().DropColumn(&model.GaFact{}, column); err != nil {
			t.Fatal(err)
		}
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	var old model.GaFact
	db.First(&old)
	if old.EventValue != nil || old.EngagementDuration != nil {
		t.Fatalf("invented legacy zero %#v", old)
	}
}

func TestGARejectsTruncationAndExplainedEmptyReports(t *testing.T) {
	for _, meta := range []string{`{"dataTruncationReasons":[{"dataTruncationType":"DATA_TRUNCATION_TYPE_PROPERTY"}]}`, `{"emptyReason":"NO_DATA_AVAILABLE"}`} {
		c := &Client{HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (int, string) { return 200, `{"metadata":` + meta + `,"rows":[]}` })}}
		if _, _, err := c.FetchGAReport(context.Background(), "token", "1", "event", "2026-09-01", "2026-09-01"); err != ErrIncompleteReport {
			t.Fatalf("fact %v", err)
		}
		if _, _, _, _, err := c.fetchGADateQuality(context.Background(), "token", "1", "2026-09-01", "2026-09-01"); err != ErrIncompleteReport {
			t.Fatalf("daily %v", err)
		}
	}
}

func TestQualityRefreshAndVersionIsolation(t *testing.T) {
	svc := New(testDB(t))
	ctx := context.Background()
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	previous := model.WebSyncReport{ProjectID: 1, Source: "ga4", Property: "1", Report: "session", Version: currentSyncVersion - 1, Token: "legacy", From: day, Through: day}
	if err := svc.rows.BeginSyncReport(ctx, &previous); err != nil {
		t.Fatal(err)
	}
	if err := svc.rows.ReplaceSyncBatch(ctx, previous, day, day, repo.SyncBatch{Quality: model.GoogleQuality{Known: true, Sampled: true}}); err != nil {
		t.Fatal(err)
	}
	progress, err := svc.syncProgress(ctx, 1, "ga4", "1")
	if err != nil || len(progress) != 0 {
		t.Fatalf("legacy template reused %#v %v", progress, err)
	}
	for _, sampled := range []bool{true, false} {
		err := svc.syncReport(ctx, 1, "ga4", "1", "session", "", day, day, 1, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
			return repo.SyncBatch{Quality: model.GoogleQuality{Known: true, Sampled: sampled}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		progress, err := svc.syncProgress(ctx, 1, "ga4", "1")
		if err != nil || len(progress) != 1 || !progress[0].Quality.Known || progress[0].Quality.Sampled != sampled {
			t.Fatalf("revised quality %#v %v", progress, err)
		}
	}
}
