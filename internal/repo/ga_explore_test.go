// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
	"time"
)

func TestGAExploreWeightsScopesAndPaging(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	f := GAFilter{ProjectID: 1, Property: "123", Report: "channel", From: day.AddDate(0, 0, -1), Through: day, PreviousFrom: day.AddDate(0, 0, -3), Limit: 50, Sort: "sessions", Direction: "desc"}
	duration := 20.0
	var facts []model.GaFact
	for i := 0; i < 205; i++ {
		facts = append(facts, model.GaFact{ProjectID: 1, Property: f.Property, Report: f.Report, Day: day, Channel: "Organic Search", Source: fmt.Sprintf("source-%03d", i), Medium: "organic", Sessions: 10, Engaged: 2, KeyEvents: 3, EngagementDuration: &duration})
	}
	weighted := facts[0]
	weighted.Day = f.From
	weighted.Sessions = 90
	weighted.Engaged = 9
	weighted.EngagementDuration = nil
	lost := facts[0]
	lost.Day = f.PreviousFrom
	lost.Source = "lost"
	facts = append(facts, weighted, lost)
	for _, change := range []func(*model.GaFact){func(v *model.GaFact) { v.ProjectID = 2 }, func(v *model.GaFact) { v.Property = "456" }, func(v *model.GaFact) { v.Report = "session" }} {
		v := facts[0]
		v.Source = "leak"
		change(&v)
		facts = append(facts, v)
	}
	if err := r.UpsertGaFacts(ctx, facts); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for offset := 0; offset < 206; offset += 50 {
		f.Offset = offset
		rows, total, err := r.GAMetrics(ctx, f)
		if err != nil || total != 206 {
			t.Fatalf("total %d %v", total, err)
		}
		for _, v := range rows {
			if seen[v.Source] {
				t.Fatal("duplicate page", v.Source)
			}
			seen[v.Source] = true
			if v.Source == "source-000" && (v.EngagementRate == nil || *v.EngagementRate != 0.11 || v.Duration != nil || v.DurationPerSession != nil) {
				t.Fatalf("weighted/null %#v", v)
			}
			if v.Source == "source-001" && (v.DurationPerSession == nil || *v.DurationPerSession != 2) {
				t.Fatalf("duration %#v", v)
			}
		}
	}
	f.Offset = 0
	f.Text = "lost"
	rows, _, err := r.GAMetrics(ctx, f)
	if err != nil || len(rows) != 1 || rows[0].SessionsChange != -10 || rows[0].CurrentRows != 0 {
		t.Fatalf("lost %#v %v", rows, err)
	}
	f.Text = ""
	f.Limit = 1
	f.Sort = "engagement_rate"
	f.Direction = "asc"
	rows, _, err = r.GAMetrics(ctx, f)
	if err != nil || len(rows) != 1 || rows[0].EngagementRate == nil {
		t.Fatalf("null ordering %#v %v", rows, err)
	}
	count := 0
	if err := r.WalkGAMetrics(ctx, f, func(GAMetric) error { count++; return nil }); err != nil || count != 206 {
		t.Fatalf("export %d %v", count, err)
	}
}
func TestGALandingPreservesPathsAndLiteralFilter(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	f := GAFilter{ProjectID: 1, Property: "123", Report: "landing", From: day, Through: day, PreviousFrom: day.AddDate(0, 0, -1), Limit: 50, Sort: "sessions", Direction: "desc"}
	for _, path := range []string{"/A", "/a", "/%_!", "/a?x=1", "(not set)"} {
		if err := r.UpsertGaFacts(ctx, []model.GaFact{{ProjectID: 1, Property: f.Property, Report: f.Report, Day: day, Landing: path, Sessions: 1}}); err != nil {
			t.Fatal(err)
		}
	}
	rows, total, err := r.GAMetrics(ctx, f)
	if err != nil || total != 5 || len(rows) != 5 {
		t.Fatalf("paths %#v %d %v", rows, total, err)
	}
	f.Text = "%_!"
	rows, total, err = r.GAMetrics(ctx, f)
	if err != nil || total != 1 || rows[0].Landing != "/%_!" {
		t.Fatalf("literal %#v %v", rows, err)
	}
}
