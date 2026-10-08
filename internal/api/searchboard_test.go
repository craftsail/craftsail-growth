// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func TestSearchReadPermissionsValidationAndCSV(t *testing.T) {
	w := newWorld(t)
	w.h.web = webstats.New(w.db)
	w.h.web.Now = func() time.Time { return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC) }
	ctx := context.Background()
	p, err := w.h.projects.Get(ctx, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	p.GscSite = "sc-domain:a.com"
	if err := w.h.projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 205; i++ {
		name := fmt.Sprintf("term-%03d", i)
		if i == 204 {
			name = "=DANGEROUS()"
		}
		row := model.GscFact{ProjectID: p.ID, Property: p.GscSite, Slice: "query", SearchType: "web", Day: day, Query: name, KeyHash: fmt.Sprint(i), Clicks: 1, Impressions: 10}
		if err := w.db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := testEngine(w.h)
	base := "/api/projects/alpha/keywords?from=2026-09-20&through=2026-09-20"
	for _, path := range []string{base, base + "&format=csv", "/api/projects/alpha/search-detail?kind=query&value=term-000&from=2026-09-20&through=2026-09-20"} {
		if res := call(r, "GET", path, "", w.viewer); res.Code != 200 {
			t.Fatalf("viewer %s: %d %s", path, res.Code, res.Body.String())
		}
		if res := call(r, "GET", path, "", w.outsider); res.Code != 404 {
			t.Fatalf("outsider %s: %d", path, res.Code)
		}
	}
	for _, query := range []string{"&page=abc", "&page=-1", "&sort=DROP", "&country=us", "&device=bad"} {
		if res := call(r, "GET", base+query, "", w.viewer); res.Code != 400 {
			t.Fatalf("invalid query %s: %d", query, res.Code)
		}
	}
	res := call(r, "GET", base+"&page_size=1&page=2", "", w.viewer)
	var envelope struct {
		Data webstats.SearchExplore `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Total != 205 || len(envelope.Data.Items) != 1 || envelope.Data.Page != 2 {
		t.Fatalf("paged %s", res.Body.String())
	}
	res = call(r, "GET", base+"&page_size=1&page=2&format=csv", "", w.viewer)
	records, err := csv.NewReader(strings.NewReader(res.Body.String())).ReadAll()
	if err != nil || len(records) != 206 {
		t.Fatalf("export %d %v %s", len(records), err, res.Body.String())
	}
	if records[1][0] != "'=DANGEROUS()" || records[1][9] != "" {
		t.Fatalf("csv safety and unknown comparison %#v", records[1])
	}
	var syncs, raw int64
	w.db.Model(&model.WebSyncReport{}).Count(&syncs)
	w.db.Model(&model.GoogleRaw{}).Count(&raw)
	if syncs != 0 || raw != 0 {
		t.Fatal("read endpoints triggered sync")
	}
}
