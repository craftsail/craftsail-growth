// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestDiscoveryResumesDeduplicatesAndPreservesFailedMap(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	db := testDB(t)
	s := New(db)
	s.Now = func() time.Time { return now }
	bad := false
	var origin string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			fmt.Fprint(w, "Sitemap: "+origin+"/sitemap.xml")
		case "/sitemap.xml":
			fmt.Fprint(w, "<sitemapindex>")
			for i := 0; i < 9; i++ {
				fmt.Fprintf(w, "<sitemap><loc>%s/child-%d.xml</loc></sitemap>", origin, i)
			}
			fmt.Fprint(w, "</sitemapindex>")
		case "/sitemap_index.xml":
			w.WriteHeader(404)
		default:
			if bad {
				fmt.Fprint(w, "<urlset><url>")
				return
			}
			fmt.Fprintf(w, "<urlset><url><loc>%s/shared</loc><lastmod>2020-01-01</lastmod></url><url><loc>%s%s/page</loc></url><url><loc>https://outside.example/a</loc></url></urlset>", origin, origin, r.URL.Path)
		}
	}))
	defer server.Close()
	origin = server.URL
	p := model.Project{Slug: "maps", Name: "Maps", Site: origin, GscSite: origin + "/"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		s = New(db)
		s.SiteHTTP = server.Client()
		if err := s.discoverIndexSitemaps(ctx, &p, p.GscSite, now); err != nil {
			t.Fatal(err)
		}
	}
	inv, err := s.rows.IndexInventory(ctx, p.ID, p.GscSite, "", "", 1, 50, now.Unix())
	if err != nil || inv.Known != 10 {
		t.Fatalf("inventory %#v %v", inv, err)
	}
	for _, row := range inv.Items {
		if strings.HasSuffix(row.URL, "/shared") && len(row.Sitemaps) != 9 {
			t.Fatal("membership lost")
		}
		if row.PublishedAt != nil {
			t.Fatal("lastmod treated as publication")
		}
	}
	var scans int64
	db.Model(&model.SitemapScan{}).Count(&scans)
	if scans != 11 {
		t.Fatal(scans)
	}
	bad = true
	if err := s.discoverIndexSitemaps(ctx, &p, p.GscSite, now.Add(8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	after, _ := s.rows.IndexInventory(ctx, p.ID, p.GscSite, "", "", 1, 50, now.Unix())
	if after.Known != 10 {
		t.Fatal("failed map destroyed URLs")
	}
}
func TestPublicationCohortAndImpressions(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	s := New(db)
	now := time.Unix(1800000000, 0)
	s.Now = func() time.Time { return now }
	p := model.Project{Slug: "cohort", Name: "Cohort", GscSite: "sc-domain:a.com"}
	db.Create(&p)
	old, recent := now.Add(-10*24*time.Hour).Unix(), now.Add(-2*24*time.Hour).Unix()
	indexed := old + 86400
	rows := []model.IndexURL{{ProjectID: p.ID, Property: p.GscSite, URL: "https://a.com/old", FromCrawl: true, PublishedAt: &old, FirstIndexedAt: &indexed}, {ProjectID: p.ID, Property: p.GscSite, URL: "https://a.com/recent", FromCrawl: true, PublishedAt: &recent, FirstIndexedAt: &indexed}, {ProjectID: p.ID, Property: p.GscSite, URL: "https://a.com/unknown", FromCrawl: true}}
	if err := s.rows.DiscoverIndexURLs(ctx, rows); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.GscFact{ProjectID: p.ID, Property: p.GscSite, Slice: "page", SearchType: "web", Page: rows[0].URL, Day: day, KeyHash: "page", Impressions: 3}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.rows.RefreshIndexImpressions(ctx, p.ID, p.GscSite, now.Unix()); err != nil {
		t.Fatal(err)
	}
	out, err := s.IndexInventory(ctx, p.Slug, "", "", 1, 50)
	if err != nil || out.PublishedMature != 1 || out.IndexedWithinWeek != 1 {
		t.Fatalf("%#v %v", out, err)
	}
	for _, row := range out.Items {
		if row.URL == rows[0].URL && (row.FirstImpressionAt == nil || *row.FirstImpressionAt != day.Unix()) {
			t.Fatal(row)
		}
	}
	future := now.Add(time.Hour).Unix()
	if err := s.SetURLPublished(ctx, p.Slug, rows[0].URL, &future); err == nil {
		t.Fatal("future accepted")
	}
}

func TestIndependentDiscoveryWithoutGoogle(t *testing.T) {
	t.Setenv("GOOGLE_SA_JSON", "")
	t.Setenv("GOOGLE_REFRESH_TOKEN", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			return
		}
		if r.URL.Path == "/sitemap.xml" {
			fmt.Fprintf(w, "<urlset><url><loc>http://%s/a</loc></url></urlset>", r.Host)
			return
		}
		w.WriteHeader(404)
	}))
	defer server.Close()
	db := testDB(t)
	p := model.Project{Slug: "independent", Name: "Independent", Site: server.URL}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	s := New(db)
	s.SiteHTTP = server.Client()
	out, err := s.RunIndexBatch(context.Background(), p.Slug, time.Time{})
	if err != nil || out.Connected || out.Pending {
		t.Fatalf("%#v %v", out, err)
	}
	inv, err := s.IndexInventory(context.Background(), p.Slug, "", "", 1, 50)
	if err != nil || inv.Known != 1 || inv.Inspected != 0 {
		t.Fatalf("%#v %v", inv, err)
	}
	var reports int64
	db.Model(&model.WebSyncReport{}).Count(&reports)
	if reports != 0 {
		t.Fatal("discovery started traffic backfill")
	}
}
