// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestReportIncludesMonitorWindow(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{
		Name: "Acme", Slug: "acme-monitor", NoSite: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	rows := &repo.Webstats{DB: db}
	through := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	if err := rows.UpsertImport(ctx, &model.WebImport{
		ProjectID: p.ID, Source: "gsc", Property: "sc-domain:acme.com",
		State: "completed", FinalizedThrough: &through, BoundarySource: "metadata",
	}); err != nil {
		t.Fatal(err)
	}
	if err := rows.UpsertWindow(ctx, &model.WebWindow{
		ProjectID: p.ID, Source: "gsc", Property: "sc-domain:acme.com",
		WindowDays: 28, FinalizedThrough: through,
		Clicks: 12, Impressions: 400, PreviousClicks: 8, PreviousImpressions: 300, CoveredDays: 28,
	}); err != nil {
		t.Fatal(err)
	}
	out, err := New(db).Build(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"### Import status",
		"final data through 2026-09-20 (Pacific time)",
		"Daily totals: 400 impressions, 12 clicks",
		"never from summing query rows",
	} {
		if !strings.Contains(out.Markdown, want) {
			t.Fatalf("missing %q\n%s", want, out.Markdown)
		}
	}
}

func TestReportIncludesWebInsight(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{
		Name: "Acme", Slug: "acme", URL: "https://acme.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	rows := &repo.Webstats{DB: db}
	prop, _ := webstats.GSCPropertyKey(p.Site)
	if err := rows.ActivateProperty(ctx, p.ID, "gsc", prop, ""); err != nil {
		t.Fatal(err)
	}
	day := time.Now().UTC().AddDate(0, 0, -4)
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	if err := rows.UpsertGscDaily(ctx, []model.GscDaily{{ProjectID: p.ID, Property: prop, SearchType: "web", Day: day, Clicks: 12, Impressions: 400}}); err != nil {
		t.Fatal(err)
	}
	out, err := New(db).Build(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"## Google Search and GA4 (supporting data)",
		"- Search Console: 400 impressions, 12 clicks",
		"official daily totals",
	} {
		if !strings.Contains(out.Markdown, want) {
			t.Fatalf("missing %q\n%s", want, out.Markdown)
		}
	}
}
