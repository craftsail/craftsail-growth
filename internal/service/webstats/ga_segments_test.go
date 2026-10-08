// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"testing"
	"time"
)

func TestGAEventAndSegmentReportsNeverSubstituteSessionTotals(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	p, err := s.projects.Create(ctx, project.CreateInput{Name: "GA", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	_ = s.projects.Save(ctx, p)
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return day.AddDate(0, 0, 3) }
	for _, report := range []string{"channel_event", "channel_segment"} {
		err = s.syncReport(ctx, p.ID, "ga4", "123", report, "", day.AddDate(0, 0, -1), day, 2, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
			return repo.SyncBatch{Quality: model.GoogleQuality{Known: true, TimeZones: []string{"UTC"}}, GA: []model.GaFact{
				{ProjectID: p.ID, Property: "123", Report: report, Day: day, Country: "Brazil", Device: "mobile", Channel: "Organic Search", EventName: "sign_up", EventCount: 5, KeyEvents: 3, Sessions: 10},
				{ProjectID: p.ID, Property: "123", Report: report, Day: day, Country: "Brazil", Device: "desktop", Channel: "Organic Search", EventName: "sign_up", EventCount: 80, Sessions: 90},
				{ProjectID: p.ID, Property: "123", Report: report, Day: day, Country: "Brazil", Device: "mobile", Channel: "Organic Search", EventName: "page_view", EventCount: 200, Sessions: 300},
			}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	in := ExploreInput{From: "2026-09-20", Through: "2026-09-20", Country: "Brazil", Device: "mobile", Events: "sign_up"}
	out, err := s.ExploreGA(ctx, p.Slug, "channel", in)
	if err != nil || !out.EventMode || out.Total != 1 || out.Items[0].EventCount != 5 || out.Items[0].KeyEvents != 3 || out.Coverage.Report != "channel_event" || !out.Comparable {
		t.Fatalf("events %#v %v", out, err)
	}
	in.Events = ""
	out, err = s.ExploreGA(ctx, p.Slug, "channel", in)
	if err != nil || out.EventMode || out.Coverage.Report != "channel_segment" || out.Items[0].Sessions != 310 {
		t.Fatalf("segment %#v %v", out, err)
	}
	in.Country = ""
	in.Device = ""
	out, err = s.ExploreGA(ctx, p.Slug, "channel", in)
	if err != nil || out.Total != 0 || out.Comparable || out.Coverage.Report != "channel" {
		t.Fatalf("fell back to detailed totals %#v %v", out, err)
	}
	for _, report := range []string{"channel_event", "landing_event"} {
		spec, _ := gaFactSpecByName(report)
		for _, m := range spec.Metrics {
			if m == "sessions" {
				t.Fatal("events requested session denominator")
			}
		}
	}
}
func TestPageIdentityPreservesMeaning(t *testing.T) {
	a := pageIdentity("https://EXAMPLE.com/A?utm_source=x&lang=pt#intro")
	if a != "example.com/A?lang=pt" {
		t.Fatal(a)
	}
	for _, b := range []string{"https://other.com/A?lang=pt", "https://example.com/a?lang=pt", "https://example.com/A/?lang=pt", "https://example.com/A?lang=en"} {
		if a == pageIdentity(b) {
			t.Fatalf("false match %s", b)
		}
	}
	if pageIdentity("https://user:secret@example.com/a") != "" {
		t.Fatal("credentials accepted")
	}
}
func TestLandingMappingKeepsHostsAndAmbiguity(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	p, err := s.projects.Create(ctx, project.CreateInput{Name: "Mapping", URL: "https://a.com"})
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	_ = s.projects.Save(ctx, p)
	prop, _ := GSCPropertyKey(p.Site)
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return day.AddDate(0, 0, 3) }
	ga := model.GaFact{ProjectID: p.ID, Property: "123", Report: "landing_context", Day: day, Hostname: "a.com", Landing: "/A?lang=pt&utm_source=google", Sessions: 10}
	err = s.syncReport(ctx, p.ID, "ga4", "123", "landing_context", "", day, day, 1, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		return repo.SyncBatch{GA: []model.GaFact{ga}, Quality: model.GoogleQuality{Known: true, TimeZones: []string{"UTC"}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := "https://a.com/A?lang=pt"
	gsc := model.GscFact{ProjectID: p.ID, Property: prop, Slice: "page", SearchType: "web", Day: day, Page: raw, Clicks: 5}
	err = s.syncReport(ctx, p.ID, "gsc", prop, "page", "web", day, day, 1, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		return repo.SyncBatch{GSC: []model.GscFact{gsc}, Quality: model.GoogleQuality{Known: true, Aggregations: []string{"byPage"}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	in := ExploreInput{From: "2026-09-20", Through: "2026-09-20", Value: ga.Landing}
	out, err := s.MapLanding(ctx, p.Slug, in)
	if err != nil || out.State != "candidate" || len(out.URLs) != 1 || out.URLs[0] != raw {
		t.Fatalf("%#v %v", out, err)
	}
	ga.Hostname = "b.com"
	ga.KeyHash = ""
	if err = s.rows.UpsertGaFacts(ctx, []model.GaFact{ga}); err != nil {
		t.Fatal(err)
	}
	out, err = s.MapLanding(ctx, p.Slug, in)
	if err != nil || out.State != "ambiguous" || len(out.Hosts) != 2 {
		t.Fatalf("merged hosts %#v %v", out, err)
	}
	in.From = "2026-09-19"
	out, err = s.MapLanding(ctx, p.Slug, in)
	if err != nil || out.State != "unverified" {
		t.Fatalf("unknown coverage %#v %v", out, err)
	}
}
