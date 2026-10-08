// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
	"strings"
	"testing"
	"time"
)

func TestGAReadPermissionsAndCSV(t *testing.T) {
	w := newWorld(t)
	w.h.web = webstats.New(w.db)
	w.h.web.Now = func() time.Time { return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC) }
	p, err := w.h.projects.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	if err := w.h.projects.Save(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 205; i++ {
		name := fmt.Sprintf("source-%03d", i)
		if i == 0 {
			name = "=DANGEROUS()"
		}
		row := model.GaFact{ProjectID: p.ID, Property: "123", Report: "channel", Day: day, Source: name, KeyHash: fmt.Sprint(i), Sessions: 1}
		if err := w.db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := testEngine(w.h)
	base := "/api/projects/alpha/ga-channels?from=2026-09-20&through=2026-09-20"
	for _, path := range []string{base, base + "&format=csv", "/api/projects/alpha/ga-landings"} {
		if res := call(r, "GET", path, "", w.viewer); res.Code != 200 {
			t.Fatalf("viewer %s %d %s", path, res.Code, res.Body.String())
		}
		if res := call(r, "GET", path, "", w.outsider); res.Code != 404 {
			t.Fatalf("outsider %d", res.Code)
		}
	}
	for _, query := range []string{"&country=usa", "&device=mobile", "&page=abc", "&page=-1", "&sort=revenue"} {
		if res := call(r, "GET", base+query, "", w.viewer); res.Code != 400 {
			t.Fatalf("invalid %s %d", query, res.Code)
		}
	}
	res := call(r, "GET", base+"&page_size=1&page=2", "", w.viewer)
	var envelope struct {
		Data webstats.GAExplore `json:"data"`
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
		t.Fatalf("CSV %d %v %s", len(records), err, res.Body.String())
	}
	if records[1][1] != "'=DANGEROUS()" || records[1][8] != "" || records[1][11] != "" {
		t.Fatalf("formula/null/comparison %#v", records[1])
	}
	var syncs int64
	w.db.Model(&model.WebSyncReport{}).Count(&syncs)
	if syncs != 0 {
		t.Fatal("read triggered sync")
	}
}
