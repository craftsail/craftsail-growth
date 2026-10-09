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

func TestSearchBoardPeriodReadsOfficialDailies(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{
		Name: "官方合计", URL: "https://example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	start, end := window(now)
	rows := &repo.Webstats{DB: db}
	property, ok := GSCPropertyKey(p.Site)
	if !ok {
		t.Fatal("site did not resolve")
	}
	if err := rows.UpsertGscFacts(ctx, []model.GscFact{{
		ProjectID: p.ID, Property: property, Slice: "query_page", SearchType: "web", Day: end,
		Query: "example tool", Page: "https://example.com/tool", Clicks: 400, Impressions: 800, Position: 6,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := rows.UpsertGscDaily(ctx, []model.GscDaily{{
		ProjectID: p.ID, Property: property, SearchType: "web", Day: end,
		Clicks: 7, Impressions: 70, Position: 3, CTR: 0.1,
	}}); err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	svc.Now = func() time.Time { return now }
	board, err := svc.SearchBoard(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if board.Period.Clicks != 7 || board.Period.Impressions != 70 {
		t.Fatalf("period %.0f/%.0f, want official 7/70 (window %s..%s)", board.Period.Clicks, board.Period.Impressions, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	if !board.Period.Measured {
		t.Fatal("official totals exist, period must be measured")
	}
	if len(board.Keywords) != 0 || len(board.Pages) != 0 || board.QueryCoverage.State != "missing" {
		t.Fatalf("query/page details must not substitute independent totals: %#v", board)
	}
}

func TestBoardTotalsDoNotFallBackToQueryRows(t *testing.T) {
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	p := periodFromOfficial(nil, now, now, now, now)
	if p.Measured || p.Clicks != 0 {
		t.Fatalf("no official totals must be unmeasured, got %+v", p)
	}
}
