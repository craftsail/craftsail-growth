// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// The demo runs crawl, audit, sampling, citations, opportunities and the
// report end to end, so it doubles as a smoke test for that whole path.
func TestDemoBuildsEveryPage(t *testing.T) {
	if testing.Short() {
		t.Skip("end-to-end demo run")
	}
	for _, lang := range []string{"en", "zh"} {
		t.Run(lang, func(t *testing.T) {
			loc = locales[lang]
			path := filepath.Join(t.TempDir(), "demo.db")
			if err := run(path, "demo", "quillpad-demo-2026"); err != nil {
				t.Fatal(err)
			}
			db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			count := func(m any) int64 {
				var n int64
				db.Model(m).Count(&n)
				return n
			}
			if n := count(&model.Page{}); n < 8 {
				t.Errorf("pages crawled = %d", n)
			}
			if n := count(&model.AuditIssue{}); n == 0 {
				t.Error("no audit issues")
			}
			if n := count(&model.SampleRun{}); n != days {
				t.Errorf("runs = %d, want %d", n, days)
			}
			if n := count(&model.SampleCitation{}); n == 0 {
				t.Error("no citations")
			}
			if n := count(&model.Task{}); n != 2 {
				t.Errorf("accepted tasks = %d, want 2", n)
			}
			if n := count(&model.Report{}); n != 1 {
				t.Errorf("reports = %d", n)
			}
			var demos []model.Project
			if err := db.Where("demo_scenario <> ?", "").Find(&demos).Error; err != nil || len(demos) != 3 {
				t.Fatalf("demo scenarios %d %v", len(demos), err)
			}
			for _, scenario := range []string{"new-site", "established"} {
				board, err := webstats.New(db).SearchBoard(context.Background(), "quillpad-"+scenario)
				if err != nil {
					t.Fatal(err)
				}
				expected := strings.ReplaceAll(scenario, "-", "_")
				if board.Observation.Mode != expected || !board.Period.Comparable {
					t.Fatalf("scenario %s %#v", scenario, board)
				}
				decline := false
				for _, op := range board.Ops {
					if op.Type == "traffic_drop" {
						decline = true
					}
				}
				if decline != (scenario == "established") {
					t.Fatalf("scenario %s decline=%v", scenario, decline)
				}
			}
			var sampled int64
			db.Model(&model.Sample{}).Distinct("sampled_on").Count(&sampled)
			if sampled != days {
				t.Errorf("sampled days = %d", sampled)
			}
		})
	}
}
