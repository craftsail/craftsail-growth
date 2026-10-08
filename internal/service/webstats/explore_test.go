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

func TestExploreDetailCoverageAndIndependentTotals(t *testing.T) {
	db := testDB(t)
	svc := New(db)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Detail", URL: "https://a.com/"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	from := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	through := from.AddDate(0, 0, 2)
	previous := from.AddDate(0, 0, -3)
	base := model.GscFact{ProjectID: p.ID, Property: prop, Slice: "page", SearchType: "web", Day: from, Page: "https://a.com/A", Clicks: 70, Impressions: 700, Position: 4}
	for _, slice := range []string{"page", "query_page"} {
		err := svc.syncReport(ctx, p.ID, "gsc", prop, slice, "web", previous, through, 7, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
			v := base
			v.Slice = slice
			if slice == "query_page" {
				v.Query = "term"
				v.Clicks = 2
			}
			return repo.SyncBatch{GSC: []model.GscFact{v}, Quality: model.GoogleQuality{Known: true, Aggregations: []string{"byPage"}}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	input := ExploreInput{From: "2026-09-18", Through: "2026-09-20", Value: base.Page}
	detail, err := svc.SearchDetail(ctx, p.Slug, "page", input)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Summary.Clicks != 70 || len(detail.Related.Items) != 1 || detail.Related.Items[0].Clicks != 2 || !detail.Comparable {
		t.Fatalf("detail %#v", detail)
	}
	if len(detail.Daily) != 3 || detail.Daily[1].Clicks == nil || *detail.Daily[1].Clicks != 0 {
		t.Fatalf("covered empty %#v", detail.Daily)
	}
	reports, _ := svc.rows.SyncReports(ctx, p.ID, "gsc", prop)
	for _, r := range reports {
		if r.Report == "page" {
			if err := db.Where("report_id = ? AND date(day) = ?", r.ID, "2026-09-19").Delete(&model.WebSyncDay{}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	detail, err = svc.SearchDetail(ctx, p.Slug, "page", input)
	if err != nil || detail.Comparable || detail.Daily[1].Clicks != nil {
		t.Fatalf("unmeasured gap %#v %v", detail, err)
	}
	input.Country = "USA"
	list, err := svc.ExploreSearch(ctx, p.Slug, "page", input)
	if err != nil || list.Coverage.Report != "page_country_device" || list.Total != 0 || list.Comparable {
		t.Fatalf("grain fallback %#v %v", list, err)
	}
}

func TestExploreValidationAndLatestBoundary(t *testing.T) {
	db := testDB(t)
	svc := New(db)
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Boundary", URL: "https://a.com/"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	through := now.AddDate(0, 0, -5)
	if err := svc.rows.UpsertImport(ctx, &model.WebImport{ProjectID: p.ID, Source: "gsc", Property: prop, FinalizedThrough: &through}); err != nil {
		t.Fatal(err)
	}
	out, err := svc.ExploreSearch(ctx, p.Slug, "query", ExploreInput{})
	if err != nil || out.Coverage.Through != "2026-09-19" || out.Coverage.TotalDays != 28 {
		t.Fatalf("default %#v %v", out, err)
	}
	for _, in := range []ExploreInput{{From: "2026-09-01"}, {From: "invalid", Through: "2026-09-20"}, {From: "2026-09-20", Through: "2026-09-01"}, {From: "2024-01-01", Through: "2026-09-01"}, {From: "2026-09-20", Through: "2027-01-01"}, {Page: -1}, {PageSize: 10000}, {Device: "TV"}, {Country: "us"}, {Sort: "SQL"}} {
		if _, err := svc.ExploreSearch(ctx, p.Slug, "query", in); err == nil {
			t.Fatalf("accepted %#v", in)
		}
	}
}
