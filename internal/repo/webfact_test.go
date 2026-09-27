// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestGscFactUpsertAndRawAppend(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	row := model.GscFact{ProjectID: 1, Slice: "query_page", SearchType: "web", Day: day, Query: "geo", Page: "https://ex.com/", Clicks: 1}
	if err := r.UpsertGscFacts(context.Background(), []model.GscFact{row}); err != nil {
		t.Fatal(err)
	}
	row.Clicks = 9
	if err := r.UpsertGscFacts(context.Background(), []model.GscFact{row}); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := r.DB.Model(&model.GscFact{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("facts = %d", n)
	}
	got := model.GscFact{}
	if err := r.DB.First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Clicks != 9 {
		t.Fatalf("clicks %v", got.Clicks)
	}
	if err := r.AppendRaw(context.Background(), &model.GoogleRaw{ProjectID: 1, Source: "gsc", Report: "query_page", Request: "{}", Body: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := r.AppendRaw(context.Background(), &model.GoogleRaw{ProjectID: 1, Source: "gsc", Report: "query_page", Request: "{}", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if err := r.DB.Model(&model.GoogleRaw{}).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("raws = %d", n)
	}
}

func TestGscFactsKeepPropertiesApart(t *testing.T) {
	r := &Webstats{DB: testDB(t)}
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	mk := func(prop string, clicks float64) model.GscFact {
		return model.GscFact{ProjectID: 1, Property: prop, Slice: "query_page", SearchType: "web", Day: day, Query: "q", Page: "https://e.com/", Clicks: clicks}
	}
	if err := r.UpsertGscFacts(context.Background(), []model.GscFact{mk("sc-domain:old.com", 1)}); err != nil {
		t.Fatal(err)
	}
	if err := r.UpsertGscFacts(context.Background(), []model.GscFact{mk("sc-domain:new.com", 5)}); err != nil {
		t.Fatal(err)
	}
	var n int64
	r.DB.Model(&model.GscFact{}).Count(&n)
	if n != 2 {
		t.Fatalf("switching property must not overwrite old rows, got %d rows", n)
	}
}

func TestPruneRawKeepsRecentAndZeroKeepsAll(t *testing.T) {
	db := testDB(t)
	r := &Webstats{DB: db}
	old := time.Now().AddDate(0, 0, -100).Unix()
	db.Create(&model.GoogleRaw{ProjectID: 1, Source: "gsc", Report: "r", FetchedAt: old})
	db.Create(&model.GoogleRaw{ProjectID: 1, Source: "gsc", Report: "r", FetchedAt: time.Now().Unix()})
	if n, _ := r.PruneRaw(context.Background(), 0); n != 0 {
		t.Fatal("retention 0 means keep everything")
	}
	if n, _ := r.PruneRaw(context.Background(), 90); n != 1 {
		t.Fatalf("pruned %d, want 1", n)
	}
}
