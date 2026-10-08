// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestBoardIndependentGrainsAndUncappedQueries(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	svc := New(db)
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Grain", URL: "https://example.com/"})
	if err != nil {
		t.Fatal(err)
	}
	prop, _ := GSCPropertyKey(p.Site)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return now }
	from, through := window(now)
	var facts []model.GscFact
	for i := 0; i < 205; i++ {
		facts = append(facts, model.GscFact{ProjectID: p.ID, Property: prop, Slice: "query", SearchType: "web", Day: through, Query: fmt.Sprintf("query-%03d", i), Clicks: 10, Impressions: 100, Position: 4})
	}
	facts = append(facts,
		model.GscFact{ProjectID: p.ID, Property: prop, Slice: "page", SearchType: "web", Day: through, Page: "https://example.com/", Clicks: 70, Impressions: 700, Position: 3},
		model.GscFact{ProjectID: p.ID, Property: prop, Slice: "query_page", SearchType: "web", Day: through, Query: "query-000", Page: "https://example.com/", Clicks: 2, Impressions: 20, Position: 8},
		model.GscFact{ProjectID: p.ID, Property: "sc-domain:other.com", Slice: "page", SearchType: "web", Day: through, Page: "https://other.com/", Clicks: 900},
		model.GscFact{ProjectID: p.ID + 1, Property: prop, Slice: "query", SearchType: "web", Day: through, Query: "other-project", Clicks: 900})
	if err := svc.rows.UpsertGscFacts(ctx, facts); err != nil {
		t.Fatal(err)
	}
	if err := svc.syncReport(ctx, p.ID, "gsc", prop, "page", "web", from, through, 28, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		return repo.SyncBatch{GSC: []model.GscFact{facts[205]}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	b, err := svc.SearchBoard(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Keywords) != 205 || b.Keywords[0].Clicks != 10 || b.Keywords[0].Query != "query-000" || b.Keywords[0].Page != "https://example.com/" {
		t.Fatalf("queries %#v", b.Keywords)
	}
	if len(b.Pages) != 1 || b.Pages[0].Clicks != 70 || b.Pages[0].Impressions != 700 {
		t.Fatalf("pages %#v", b.Pages)
	}
	if b.PageCoverage.State != "covered" || b.QueryCoverage.State != "missing" {
		t.Fatalf("coverage %#v %#v", b.PageCoverage, b.QueryCoverage)
	}
}

func TestGSCIndependentRequestAndAggregation(t *testing.T) {
	for _, tt := range []struct {
		report string
		dims   []string
		agg    string
	}{
		{"query", []string{"date", "query"}, "byProperty"},
		{"page", []string{"date", "page"}, "byPage"},
		{"country_device", []string{"date", "country", "device"}, "byProperty"},
		{"page_country_device", []string{"date", "page", "country", "device"}, "byPage"},
	} {
		t.Run(tt.report, func(t *testing.T) {
			c := &Client{HTTP: &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
				var body gscFactBody
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(body.Dimensions, tt.dims) || body.AggregationType != tt.agg || body.DataState != "final" {
					t.Fatalf("request %#v", body)
				}
				return 200, fmt.Sprintf(`{"responseAggregationType":%q,"rows":[]}`, tt.agg)
			})}}
			_, q, err := c.FetchGSCReport(context.Background(), "token", "sc-domain:example.com", tt.report, "web", "2026-09-01", "2026-09-01")
			if err != nil || !q.Known || len(q.Aggregations) != 1 || q.Aggregations[0] != tt.agg {
				t.Fatalf("quality %#v %v", q, err)
			}
		})
	}
}
