// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"math"
	"testing"
	"time"
)

func TestCTRReferenceExcludesTargetBrandAndFallsBack(t *testing.T) {
	p := &model.Project{Name: "Acme", Site: "https://acme.test"}
	var rows []KeywordRow
	for i := 0; i < 10; i++ {
		rows = append(rows, KeywordRow{Query: fmt.Sprintf("tool %d", i), Impressions: 100, Clicks: 10, Position: 5})
	}
	rows = append(rows, KeywordRow{Query: "target", Impressions: 100000, Clicks: 1, Position: 5}, KeywordRow{Query: "ac-me", Impressions: 100000, Clicks: 100000, Position: 5})
	refs := ctrReferences{project: p, pools: []ctrPool{{rows: rows[:1], scope: "segment", trusted: true}, {rows: rows, scope: "site", trusted: true}}}
	ref := refs.reference("target", 5)
	if ref.CTR == nil || *ref.CTR != 0.1 || ref.Scope != "site" || ref.Queries != 10 {
		t.Fatalf("bad reference %+v", ref)
	}
	if r := refs.reference("Acme tools", 5); r.CTR != nil || r.Reason != "brand" {
		t.Fatal(r)
	}
	refs.pools[1].rows = rows[1:]
	if r := refs.reference("target", 5); r.CTR != nil {
		t.Fatal("self or brand supplied missing sample", r)
	}
	refs.pools[0].trusted = false
	refs.pools[1].trusted = false
	if r := refs.reference("target", 5); r.Reason != "quality" {
		t.Fatal(r)
	}
}
func TestCTRServiceUsesCoveredNinetyDaysAndUnknownIsNotComparable(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Local CTR", URL: "https://ctr.test/"})
	if err != nil {
		t.Fatal(err)
	}
	end := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	start := end.AddDate(0, 0, -89)
	err = s.syncReport(ctx, p.ID, "gsc", p.Site, "query", "web", start, end, 90, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		var rows []model.GscFact
		for i := 0; i < 10; i++ {
			rows = append(rows, model.GscFact{ProjectID: p.ID, Property: p.Site, Slice: "query", SearchType: "web", Day: end, Query: fmt.Sprintf("tool %d", i), Position: 5, Clicks: 10, Impressions: 100})
		}
		return repo.SyncBatch{GSC: rows, Quality: model.GoogleQuality{Known: true, Aggregations: []string{"byProperty"}}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	refs, err := s.ctrReferences(ctx, p, p.Site, end, "usa", "mobile")
	if err != nil {
		t.Fatal(err)
	}
	if r := refs.reference("target", 5); r.CTR == nil || r.Scope != "site" {
		t.Fatalf("not loaded %+v", r)
	}
	ops := ctrOpportunities([]KeywordRow{{Query: "target", Impressions: 1000, CTR: .01, Position: 5}}, refs, 500)
	if len(ops) != 1 || math.Abs(ops[0].Metric-90) > 1e-8 || ops[0].Reference.Version != 1 {
		t.Fatal(ops)
	}
	if err := db.Model(&model.WebSyncDay{}).Where("1=1").Update("quality_json", "").Error; err != nil {
		t.Fatal(err)
	}
	refs, err = s.ctrReferences(ctx, p, p.Site, end, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if ops := ctrOpportunities([]KeywordRow{{Query: "target", Impressions: 1000, CTR: .01, Position: 5}}, refs, 500); len(ops) != 0 {
		t.Fatal(ops)
	}
	c, err := s.grainCoverage(ctx, p.ID, p.Site, "query", start, end)
	if err != nil || c.State != "covered" || trustedSearch(c) {
		t.Fatalf("unknown quality %+v %v", c, err)
	}
}
