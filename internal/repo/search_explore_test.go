// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
	"time"
)

func TestSearchSQLFilteringPagingAndWeights(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	f := SearchFilter{ProjectID: 1, Property: "sc-domain:a.com", Slice: "query", Group: "query", From: day.AddDate(0, 0, -1), Through: day, PreviousFrom: day.AddDate(0, 0, -3), Limit: 50, Sort: "clicks", Direction: "desc"}
	var facts []model.GscFact
	for i := 0; i < 205; i++ {
		facts = append(facts, model.GscFact{ProjectID: 1, Property: f.Property, Slice: "query", SearchType: "web", Day: day, Query: fmt.Sprintf("term-%03d", i), Clicks: 10, Impressions: 10, Position: 1})
	}
	facts = append(facts, model.GscFact{ProjectID: 1, Property: f.Property, Slice: "query", SearchType: "web", Day: f.From, Query: "term-000", Impressions: 90, Position: 9}, model.GscFact{ProjectID: 1, Property: f.Property, Slice: "query", SearchType: "web", Day: f.PreviousFrom, Query: "lost", Clicks: 30, Impressions: 100, Position: 4})
	for _, change := range []func(*model.GscFact){func(v *model.GscFact) { v.ProjectID = 2 }, func(v *model.GscFact) { v.Property = "sc-domain:b.com" }, func(v *model.GscFact) { v.Slice = "query_page" }, func(v *model.GscFact) { v.SearchType = "image" }} {
		copy := facts[0]
		copy.Query = "leak"
		change(&copy)
		facts = append(facts, copy)
	}
	if err := r.UpsertGscFacts(ctx, facts); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for offset := 0; offset < 206; offset += 50 {
		f.Offset = offset
		rows, total, err := r.SearchMetrics(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		if total != 206 {
			t.Fatalf("total %d", total)
		}
		for _, row := range rows {
			if seen[row.Name] {
				t.Fatal("duplicate page boundary", row.Name)
			}
			seen[row.Name] = true
			if row.Name == "term-000" && (row.Position != 8.2 || row.CTR != 0.1) {
				t.Fatalf("weighted metrics %#v", row)
			}
		}
	}
	f.Offset = 0
	f.Text = "term-204"
	rows, total, err := r.SearchMetrics(ctx, f)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Name != "term-204" {
		t.Fatalf("filter before limit %#v %d %v", rows, total, err)
	}
	f.Text = "lost"
	rows, _, err = r.SearchMetrics(ctx, f)
	if err != nil || len(rows) != 1 || rows[0].Clicks != 0 || rows[0].PreviousClicks != 30 || rows[0].ClicksChange != -30 {
		t.Fatalf("lost entity %#v %v", rows, err)
	}
	f.Text = ""
	f.Sort = "position"
	f.Direction = "asc"
	f.Limit = 1
	ranked, _, err := r.SearchMetrics(ctx, f)
	if err != nil || len(ranked) != 1 || ranked[0].Impressions == 0 {
		t.Fatalf("unmeasured rank sorted before measured rows: %#v %v", ranked, err)
	}
	count := 0
	if err := r.WalkSearchMetrics(ctx, f, func(SearchMetric) error { count++; return nil }); err != nil || count != 206 {
		t.Fatalf("export %d %v", count, err)
	}
}

func TestSearchLiteralAndExactScope(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	ctx := context.Background()
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	f := SearchFilter{ProjectID: 1, Property: "a", Slice: "page_country_device", Group: "page", Country: "usa", Device: "mobile", From: day, Through: day, PreviousFrom: day.AddDate(0, 0, -1), Limit: 50}
	var facts []model.GscFact
	for _, page := range []string{"/A", "/a", "/%_!", "/other"} {
		facts = append(facts, model.GscFact{ProjectID: 1, Property: "a", Slice: f.Slice, SearchType: "web", Day: day, Page: page, Country: "usa", Device: "mobile", Clicks: 1})
	}
	other := facts[0]
	other.Country = "bra"
	other.Clicks = 900
	facts = append(facts, other)
	if err := r.UpsertGscFacts(ctx, facts); err != nil {
		t.Fatal(err)
	}
	f.Text = "%_!"
	rows, total, err := r.SearchMetrics(ctx, f)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Name != "/%_!" {
		t.Fatalf("literal %#v %v", rows, err)
	}
	f.Text = ""
	f.ExactPage = "/A"
	rows, total, err = r.SearchMetrics(ctx, f)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Clicks != 1 {
		t.Fatalf("exact country/device %#v %v", rows, err)
	}
	f.Sort = "clicks; DROP TABLE gsc_facts"
	if _, _, err := r.SearchMetrics(ctx, f); err == nil {
		t.Fatal("accepted unsafe order")
	}
}
