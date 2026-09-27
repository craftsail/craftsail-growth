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

func ptrF(v float64) *float64 { return &v }

func TestBuildInsightUsesOfficialTotals(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := []QueryRow{{Query: "best geo audit", Page: "https://ex.com/", Clicks: 1, Impressions: 50, Position: 4}}
	ga := []model.GaFact{{Source: "chatgpt.com", Medium: "referral", Landing: "https://ex.com/", Sessions: 3}}
	ins := buildInsight(day, day, Totals{Clicks: ptrF(100), Impressions: ptrF(2000), Sessions: ptrF(80)}, rows, ga, []string{"什么是品牌"})
	if c, _ := ins["gsc_clicks"].(*float64); c == nil || *c != 100 {
		t.Fatal("clicks must come from the official daily total, not the sum of query rows")
	}
	if ins["ai_sessions"] != 3.0 {
		t.Fatalf("ai_sessions = %v", ins["ai_sessions"])
	}
	gaps, _ := ins["gap_queries"].([]map[string]any)
	if len(gaps) != 1 || gaps[0]["query"] != "best geo audit" {
		t.Fatalf("gap queries = %v", ins["gap_queries"])
	}
}

func TestInsightWithoutOfficialTotalsIsUnmeasured(t *testing.T) {
	ins := buildInsight(time.Now(), time.Now(), Totals{}, nil, nil, nil)
	if c, _ := ins["gsc_clicks"].(*float64); c != nil {
		t.Fatal("no official totals must stay nil")
	}
}

func TestInsightReadsFactsForActiveProperty(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Acme", Slug: "acme", URL: "https://ex.com/"})
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	_, end := window(fixed)
	rows := &repo.Webstats{DB: db}
	prop, _ := GSCPropertyKey(p.Site)
	if err := rows.ActivateProperty(ctx, p.ID, "gsc", prop, ""); err != nil {
		t.Fatal(err)
	}
	if err := rows.UpsertGscDaily(ctx, []model.GscDaily{{ProjectID: p.ID, Property: prop, SearchType: "web", Day: end, Clicks: 9, Impressions: 90}}); err != nil {
		t.Fatal(err)
	}
	if err := rows.UpsertGscFacts(ctx, []model.GscFact{
		{ProjectID: p.ID, Property: prop, Slice: "query_page", SearchType: "web", Day: end, Query: "geo audit", Page: "https://ex.com/", Clicks: 2, Impressions: 20},
		{ProjectID: p.ID, Property: "sc-domain:old.com", Slice: "query_page", SearchType: "web", Day: end, Query: "old site", Page: "https://old.com/", Clicks: 50, Impressions: 500},
	}); err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	svc.Now = func() time.Time { return fixed }
	ins, err := svc.Insight(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := ins["gsc_clicks"].(*float64); c == nil || *c != 9 {
		t.Fatalf("clicks = %v", ins["gsc_clicks"])
	}
	top, _ := ins["top_queries"].([]map[string]any)
	if len(top) != 1 || top[0]["query"] != "geo audit" {
		t.Fatalf("top queries must only use the active property, got %v", top)
	}
}
