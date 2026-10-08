// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

type rotationRT struct {
	urls   map[string]int
	status int
	body   string
}

func (rt *rotationRT) RoundTrip(req *http.Request) (*http.Response, error) {
	var input inspectBody
	if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
		return nil, err
	}
	rt.urls[input.InspectionURL]++
	status := rt.status
	if status == 0 {
		status = 200
	}
	body := rt.body
	if body == "" {
		body = `{"inspectionResult":{"indexStatusResult":{"verdict":"PASS","googleCanonical":"https://a.com/canonical","userCanonical":"https://a.com/declared","robotsTxtState":"ALLOWED","pageFetchState":"SUCCESSFUL"}}}`
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
}
func TestIndexRotationAcrossThousandURLsAndRestart(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	pages := make([]model.Page, 1000)
	for i := range pages {
		pages[i] = model.Page{ProjectID: 1, URL: fmt.Sprintf("https://a.com/%04d", i)}
	}
	if err := db.CreateInBatches(pages, 100).Error; err != nil {
		t.Fatal(err)
	}
	// Search-only URLs must join the inventory; duplicates merge and other properties do not leak.
	facts := []model.GscFact{
		{ProjectID: 1, Property: "sc-domain:a.com", Slice: "page", SearchType: "web", Day: now, Page: pages[0].URL, KeyHash: "dup"},
		{ProjectID: 1, Property: "sc-domain:a.com", Slice: "page", SearchType: "web", Day: now, Page: "https://a.com/search-only", KeyHash: "new"},
		{ProjectID: 1, Property: "sc-domain:old.com", Slice: "page", SearchType: "web", Day: now, Page: "https://a.com/old", KeyHash: "old"},
		{ProjectID: 1, Property: "sc-domain:a.com", Slice: "page", SearchType: "web", Day: now, Page: "https://other.com/no", KeyHash: "outside"},
	}
	if err := db.Create(&facts).Error; err != nil {
		t.Fatal(err)
	}
	rt := &rotationRT{urls: map[string]int{}}
	s.Client = &Client{HTTP: &http.Client{Transport: rt}}
	// Exhaust a request budget after successful inspections, then reconstruct the service.
	batch, cancel, _ := withSyncBudget(ctx, BatchOptions{MaxRequests: 5})
	defer cancel()
	if _, err := s.inspectDue(batch, "tok", 1, "sc-domain:a.com", now); !errors.Is(err, ErrSyncBudget) {
		t.Fatalf("budget %v", err)
	}
	for i := 0; i < 34; i++ {
		s = New(db)
		s.Now = func() time.Time { return now.Add(time.Duration(i+1) * time.Minute) }
		s.Client = &Client{HTTP: &http.Client{Transport: rt}}
		if _, err := s.inspectDue(ctx, "tok", 1, "sc-domain:a.com", now); err != nil {
			t.Fatal(err)
		}
	}
	if len(rt.urls) != 1001 {
		t.Fatalf("only %d URLs inspected", len(rt.urls))
	}
	for url, n := range rt.urls {
		if n != 1 {
			t.Fatalf("%s inspected %d times", url, n)
		}
	}
	inventory, err := s.rows.IndexInventory(ctx, 1, "sc-domain:a.com", "", "", 1, 50, now.Unix())
	if err != nil || inventory.Known != 1001 || inventory.Inspected != 1001 || inventory.Due != 0 {
		t.Fatalf("%#v %v", inventory, err)
	}
	if !inventory.Items[0].FromCrawl || !inventory.Items[0].FromSearch {
		t.Fatal("provenance lost")
	}
	history, _ := s.rows.IndexHistory(ctx, 1, "sc-domain:a.com", pages[0].URL, 1, 50)
	if history.Total != 1 || history.Items[0].Result.GoogleCanonical != "https://a.com/canonical" {
		t.Fatalf("%#v", history)
	}
	// Switching property must cause a new inspection, not inherit previous freshness.
	if _, err := s.inspectDue(ctx, "tok", 1, "https://a.com/", now); err != nil {
		t.Fatal(err)
	}
	current, _ := s.rows.IndexInventory(ctx, 1, "https://a.com/", "", "", 1, 50, now.Unix())
	if current.Known != 1000 || current.Inspected != 30 {
		t.Fatalf("new property %#v", current)
	}
}
func TestInspectionFailurePreservesVerdictAndStopsQuotaBurst(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	if err := db.Create(&[]model.Page{{ProjectID: 1, URL: "https://a.com/a"}, {ProjectID: 1, URL: "https://a.com/b"}}).Error; err != nil {
		t.Fatal(err)
	}
	rt := &rotationRT{urls: map[string]int{}}
	s.Client = &Client{HTTP: &http.Client{Transport: rt}}
	if _, err := s.inspectDue(ctx, "tok", 1, "sc-domain:a.com", now); err != nil {
		t.Fatal(err)
	}
	rt.status = 429
	rt.body = `{"error":{"message":"Quota exceeded"}}`
	note, err := s.inspectDue(ctx, "tok", 1, "sc-domain:a.com", now.Add(8*24*time.Hour))
	var quotaWait *QuotaWait
	if !errors.As(err, &quotaWait) || note == "" {
		t.Fatalf("%s %v", note, err)
	}
	if rt.urls["https://a.com/a"]+rt.urls["https://a.com/b"] != 3 {
		t.Fatal("quota failure caused repeated requests")
	}
	r := &repo.Webstats{DB: db}
	rows, _ := r.ListIndex(ctx, 1, "sc-domain:a.com")
	if len(rows) != 2 || rows[0].Verdict != "PASS" || rows[0].FetchedAt != now.Unix() {
		t.Fatal(rows)
	}
	history, _ := r.IndexHistory(ctx, 1, "sc-domain:a.com", "https://a.com/a", 1, 50)
	if history.Total != 2 || history.Items[0].Error == "" {
		t.Fatal(history)
	}
}
func TestInspectRejectsMissingResultAndPropertyPort(t *testing.T) {
	rt := &rotationRT{urls: map[string]int{}, body: `{}`}
	c := &Client{HTTP: &http.Client{Transport: rt}}
	if _, err := c.InspectURL("tok", "sc-domain:a.com", "https://a.com/a"); err == nil {
		t.Fatal("empty result counted as success")
	}
	if urlInProperty("https://a.com:444/", "https://a.com/a") || urlInProperty("sc-domain:a.com", "ftp://a.com/a") {
		t.Fatal("invalid scope")
	}
}
