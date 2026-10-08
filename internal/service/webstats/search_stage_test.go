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

func TestObservationPromotionPersistenceAndScope(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "Stage", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	end := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC) // Sunday
	s.Now = func() time.Time { return end.AddDate(0, 0, 3) }
	if err := s.rows.ActivateProperty(ctx, p.ID, "gsc", prop, ""); err != nil {
		t.Fatal(err)
	}
	fill := func(clicks float64) {
		t.Helper()
		err := s.syncReport(ctx, p.ID, "gsc", prop, "daily", "web", end.AddDate(0, 0, -55), end, 56, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
			var rows []model.GscDaily
			for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
				rows = append(rows, model.GscDaily{ProjectID: p.ID, Property: prop, SearchType: "web", Day: d, Clicks: clicks, Impressions: 100})
			}
			return repo.SyncBatch{GSCDaily: rows}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	fill(10)
	o, _, err := s.searchObservation(ctx, p)
	if err != nil || o.Mode != "new_site" {
		t.Fatalf("GET promoted: %#v %v", o, err)
	}
	if err := s.promoteSearchStage(ctx, p.ID, prop, end); err != nil {
		t.Fatal(err)
	}
	o, _, err = s.searchObservation(ctx, p)
	if err != nil || o.Mode != "established" || o.EstablishedThrough != "2026-09-20" {
		t.Fatalf("promotion %#v %v", o, err)
	}
	for _, w := range o.Weeks {
		if w.Clicks == nil || *w.Clicks != 70 {
			t.Fatalf("week %#v", w)
		}
	}
	// Even after all historical data is removed, a qualified property retains its stage.
	if err := s.rows.DB.Where("project_id = ?", p.ID).Delete(&model.GscDaily{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.rows.ActivateProperty(ctx, p.ID, "gsc", prop, ""); err != nil {
		t.Fatal(err)
	}
	o, _, err = s.searchObservation(ctx, p)
	if err != nil || o.Mode != "established" || o.Weeks[0].Clicks != nil {
		t.Fatalf("latched stage/unknown week %#v %v", o, err)
	}
	p.GscSite = "sc-domain:other.com"
	if err := s.rows.ActivateProperty(ctx, p.ID, "gsc", p.GscSite, ""); err != nil {
		t.Fatal(err)
	}
	o, _, err = s.searchObservation(ctx, p)
	if err != nil || o.Mode != "new_site" || o.PagesWithImpressions != nil {
		t.Fatalf("property leakage %#v %v", o, err)
	}
	p.SearchMode = "established"
	o, _, err = s.searchObservation(ctx, p)
	if err != nil || o.Mode != "established" || o.Reason != "manual" || o.Coverage.State == "covered" {
		t.Fatalf("manual %#v %v", o, err)
	}
}

func TestObservationRequiresTwoCoveredWindowsAndDistinguishesZero(t *testing.T) {
	for _, tt := range []struct {
		name   string
		days   int
		clicks float64
		want   string
	}{{"one window", 28, 10, "new_site"}, {"small sample", 56, 1, "new_site"}, {"qualified", 56, 10, "established"}} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := New(testDB(t))
			p, err := project.New(s.rows.DB).Create(ctx, project.CreateInput{Name: "Stage", URL: "https://example.com"})
			if err != nil {
				t.Fatal(err)
			}
			prop, _ := GSCPropertyKey(p.Site)
			end := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
			s.Now = func() time.Time { return end.AddDate(0, 0, 3) }
			if err := s.rows.ActivateProperty(ctx, p.ID, "gsc", prop, ""); err != nil {
				t.Fatal(err)
			}
			err = s.syncReport(ctx, p.ID, "gsc", prop, "daily", "web", end.AddDate(0, 0, 1-tt.days), end, tt.days, 1, func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
				var rows []model.GscDaily
				for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
					rows = append(rows, model.GscDaily{ProjectID: p.ID, Property: prop, SearchType: "web", Day: d, Clicks: tt.clicks, Impressions: 100})
				}
				return repo.SyncBatch{GSCDaily: rows}, nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := s.promoteSearchStage(ctx, p.ID, prop, end); err != nil {
				t.Fatal(err)
			}
			o, _, err := s.searchObservation(ctx, p)
			if err != nil || o.Mode != tt.want {
				t.Fatalf("%#v %v", o, err)
			}
			// A completed empty page report establishes zero visible pages.
			err = s.syncReport(ctx, p.ID, "gsc", prop, "page", "web", end.AddDate(0, 0, -27), end, 28, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) { return repo.SyncBatch{}, nil })
			if err != nil {
				t.Fatal(err)
			}
			o, _, err = s.searchObservation(ctx, p)
			if err != nil || o.PagesWithImpressions == nil || *o.PagesWithImpressions != 0 {
				t.Fatalf("zero pages %#v %v", o, err)
			}
			b, err := s.SearchBoard(ctx, p.Slug)
			if err != nil {
				t.Fatal(err)
			}
			if tt.days < 56 && (b.Period.Comparable || b.Period.ClicksDelta != nil || len(b.Ops) > 0) {
				t.Fatalf("partial comparison %#v", b)
			}
		})
	}
}

func TestPageDeclineMustPersistInBothWeeks(t *testing.T) {
	end := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	drops := []PageClicks{{URL: "/persistent"}, {URL: "/burst"}}
	previous := []model.GscFact{{Page: "/persistent", Day: end.AddDate(0, 0, -28), Clicks: 50}, {Page: "/persistent", Day: end.AddDate(0, 0, -35), Clicks: 50}, {Page: "/burst", Day: end.AddDate(0, 0, -28), Clicks: 500}}
	markSustainedPages(nil, previous, drops, end)
	if !drops[0].Sustained || drops[1].Sustained {
		t.Fatalf("%#v", drops)
	}
	if pctDelta(1, 0) != nil {
		t.Fatal("zero baseline invented a percentage")
	}
}

func TestSnapshotHidesPartialSearchComparison(t *testing.T) {
	end := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	o := &SearchObservation{Property: "sc-domain:example.com", Coverage: GrainCoverage{Through: "2026-09-20", State: "covered"}, PreviousCoverage: GrainCoverage{From: "2026-07-27", State: "partial"}}
	windows := []model.WebWindow{{Source: "gsc", Property: o.Property, WindowDays: 28, FinalizedThrough: end, Clicks: 10, PreviousClicks: 100}}
	var rows []model.GscDaily
	for i := 0; i < 56; i++ {
		rows = append(rows, model.GscDaily{Day: end.AddDate(0, 0, -i), Clicks: 1})
	}
	if len(snapshotDeltas(windows, o, rows)) != 0 {
		t.Fatal("partial coverage compared")
	}
	o.PreviousCoverage.State = "covered"
	if len(snapshotDeltas(windows, o, rows[:55])) != 0 {
		t.Fatal("missing daily row compared")
	}
	o.Coverage.Quality = model.GoogleQuality{Known: true, Aggregations: []string{"byProperty"}}
	o.PreviousCoverage.Quality = o.Coverage.Quality
	if len(snapshotDeltas(windows, o, rows)) != 2 {
		t.Fatal("covered comparison omitted")
	}
	windows[0].Property = "sc-domain:other.com"
	if len(snapshotDeltas(windows, o, rows)) != 0 {
		t.Fatal("wrong resource compared")
	}
}

func TestSearchSettingsMigrationDefaults(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Legacy", URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"search_mode", "search_min_impressions"} {
		if err := db.Migrator().DropColumn(&model.Project{}, column); err != nil {
			t.Fatal(err)
		}
	}
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	got, err := project.New(db).Get(ctx, p.Slug)
	if err != nil || got.SearchMode != "auto" || got.SearchMinImpressions != 500 {
		t.Fatalf("migration %#v %v", got, err)
	}
}
