// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func TestIndexReadPermissionsFiltersAndPropertyEdit(t *testing.T) {
	w := newWorld(t)
	w.h.web = webstats.New(w.db)
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	w.h.web.Now = func() time.Time { return now }
	p, err := w.h.projects.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	p.GscSite = "sc-domain:a.com"
	if err := w.h.projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	r := &repo.Webstats{DB: w.db}
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{{ProjectID: p.ID, Property: p.GscSite, URL: "https://a.com/a", FromCrawl: true}, {ProjectID: p.ID, Property: p.GscSite, URL: "https://a.com/b", FromSearch: true}}); err != nil {
		t.Fatal(err)
	}
	due, _ := r.DueIndexURLs(ctx, p.ID, p.GscSite, now.Unix(), 1)
	if err := r.RecordInspection(ctx, due[0], &model.GscIndex{Verdict: "PASS"}, "", now); err != nil {
		t.Fatal(err)
	}
	engine := testEngine(w.h)
	base := "/api/projects/alpha/indexing"
	for _, path := range []string{base, base + "/history?url=https%3A%2F%2Fa.com%2Fa"} {
		if res := call(engine, "GET", path, "", w.viewer); res.Code != 200 {
			t.Fatalf("viewer %d %s", res.Code, res.Body.String())
		}
		if res := call(engine, "GET", path, "", w.outsider); res.Code != 404 {
			t.Fatalf("outsider %d", res.Code)
		}
	}
	for _, suffix := range []string{"?page=-1", "?page_size=201", "?page=word", "?state=bad", "/history", "/history?url=https://a.com/a&page=0"} {
		if res := call(engine, "GET", base+suffix, "", w.viewer); res.Code != 400 {
			t.Fatalf("bad params %s: %d", suffix, res.Code)
		}
	}
	res := call(engine, "GET", base+"?state=pending&page_size=1", "", w.viewer)
	var payload struct {
		Data repo.IndexInventory `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Total != 1 || payload.Data.Known != 2 || payload.Data.Inspected != 1 || payload.Data.Items[0].URL != "https://a.com/b" {
		t.Fatalf("%s", res.Body.String())
	}
	// A settings edit hides the old property immediately, even if its ledger is active.
	if err := r.ActivateProperty(ctx, p.ID, "gsc", p.GscSite, ""); err != nil {
		t.Fatal(err)
	}
	p.GscSite = "https://a.com/"
	if err := w.h.projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	res = call(engine, "GET", base, "", w.viewer)
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Known != 0 || payload.Data.Property != p.GscSite {
		t.Fatalf("old property leaked: %s", res.Body.String())
	}
	res = call(engine, "GET", base+"/history?url=https%3A%2F%2Fa.com%2Fa", "", w.viewer)
	var hist struct {
		Data repo.InspectionHistory `json:"data"`
	}
	json.Unmarshal(res.Body.Bytes(), &hist)
	if hist.Data.Total != 0 {
		t.Fatal("old history leaked")
	}
	var n int64
	w.db.Model(&model.IndexInspection{}).Count(&n)
	if n != 1 {
		t.Fatal("GET mutated history")
	}
	w.db.Model(&model.IndexURL{}).Count(&n)
	if n != 2 {
		t.Fatal("GET mutated inventory")
	}
}

func TestIndexLifecycleEditPermissions(t *testing.T) {
	w := newWorld(t)
	w.h.web = webstats.New(w.db)
	ctx := context.Background()
	p, _ := w.h.projects.Get(ctx, "alpha")
	p.GscSite = "sc-domain:a.com"
	w.h.projects.Save(ctx, p)
	r := &repo.Webstats{DB: w.db}
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{{ProjectID: p.ID, Property: p.GscSite, URL: "https://a.com/a", FromCrawl: true}}); err != nil {
		t.Fatal(err)
	}
	engine := testEngine(w.h)
	for _, tc := range []struct{ method, path, body string }{{"PUT", "published", `{"url":"https://a.com/a","published_at":1700000000}`}, {"POST", "sitemaps", `{"url":"https://a.com/sitemap.xml"}`}} {
		path := "/api/projects/alpha/indexing/" + tc.path
		if res := call(engine, tc.method, path, tc.body, w.viewer); res.Code != 403 {
			t.Fatalf("viewer %d", res.Code)
		}
		if res := call(engine, tc.method, path, tc.body, w.outsider); res.Code != 404 {
			t.Fatalf("outsider %d", res.Code)
		}
		if res := call(engine, tc.method, path, tc.body, w.editor); res.Code != 200 {
			t.Fatalf("editor %d %s", res.Code, res.Body.String())
		}
	}
	if res := call(engine, "POST", "/api/projects/alpha/indexing/sitemaps", `{"url":"https://outside.example/sitemap.xml"}`, w.editor); res.Code != 400 {
		t.Fatal(res.Code)
	}
	if res := call(engine, "PUT", "/api/projects/alpha/indexing/published", `{"url":"https://a.com/a","published_at":9999999999}`, w.editor); res.Code != 400 {
		t.Fatal(res.Code)
	}
}
