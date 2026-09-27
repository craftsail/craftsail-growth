// SPDX-License-Identifier: AGPL-3.0-or-later

package invoker

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

func openSQLiteFile(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "data", "app.db")
	if err := ensureSQLiteDir(dsn); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open((&sqliteParser{}).GetDialector(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	tune(db)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	// A second run must be a no-op, as it is on every server start.
	if err := model.AutoMigrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	return db
}

func TestSQLiteDSN(t *testing.T) {
	if got := sqliteDSN("data/app.db"); got != "data/app.db?"+sqliteDefaults {
		t.Fatalf("defaults not added: %s", got)
	}
	if got := sqliteDSN("file:x.db?cache=shared"); got != "file:x.db?cache=shared" {
		t.Fatalf("explicit query must be kept: %s", got)
	}
	if got := sqlitePath("file:data/x.db?cache=shared"); got != "data/x.db" {
		t.Fatalf("path: %s", got)
	}
	if got := sqlitePath("file::memory:?cache=shared"); got != "" {
		t.Fatalf("memory path: %s", got)
	}
	cfg, _ := (&sqliteParser{}).ParseDSN("data/app.db")
	if cfg.DBName != "app.db" {
		t.Fatalf("db name: %s", cfg.DBName)
	}
}

func TestSQLiteWALEnabled(t *testing.T) {
	db := openSQLiteFile(t)
	var mode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

// Every ON CONFLICT target must match a unique index on SQLite, which,
// unlike MySQL, rejects the statement otherwise. Each upsert runs twice so
// the second call takes the conflict path.
func TestSQLiteUpserts(t *testing.T) {
	db := openSQLiteFile(t)
	ctx := context.Background()
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	ws := &repo.Webstats{DB: db}
	steps := map[string]func() error{
		"samples": func() error {
			return (&repo.Samples{DB: db}).Upsert(ctx, []model.Sample{{ProjectID: 1, SampledOn: day, Platform: "openai", QID: "q1", Round: 1, SampleMode: "api", OK: true}})
		},
		"metrics": func() error {
			return (&repo.Metrics{DB: db}).Upsert(ctx, &model.Metric{ProjectID: 1, MetricOn: day, Market: "global"})
		},
		"assets": func() error {
			return (&repo.Assets{DB: db}).Upsert(ctx, &model.Asset{ProjectID: 1, Path: "llms.txt"})
		},
		"members": func() error {
			return (&repo.Members{DB: db}).Set(ctx, 1, 1, "view")
		},
		"gsc_sitemaps": func() error {
			return ws.UpsertSitemaps(ctx, []model.GscSitemap{{ProjectID: 1, KeyHash: "k"}})
		},
		"gsc_index": func() error {
			return ws.UpsertIndex(ctx, []model.GscIndex{{ProjectID: 1, KeyHash: "k"}})
		},
		"gsc_facts": func() error {
			return ws.UpsertGscFacts(ctx, []model.GscFact{{ProjectID: 1, Property: "p", Slice: "query", Day: day, KeyHash: "k"}})
		},
		"ga_facts": func() error {
			return ws.UpsertGaFacts(ctx, []model.GaFact{{ProjectID: 1, Property: "p", Report: "r", Day: day, KeyHash: "k"}})
		},
		"web_properties": func() error {
			return ws.ActivateProperty(ctx, 1, "gsc", "sc-domain:example.com", "UTC")
		},
		"gsc_daily": func() error {
			return ws.UpsertGscDaily(ctx, []model.GscDaily{{ProjectID: 1, Property: "p", SearchType: "web", Day: day}})
		},
		"ga_daily": func() error {
			return ws.UpsertGaDaily(ctx, []model.GaDaily{{ProjectID: 1, Property: "p", Day: day}})
		},
		"web_imports": func() error {
			return ws.UpsertImport(ctx, &model.WebImport{ProjectID: 1, Source: "gsc"})
		},
		"web_windows": func() error {
			return ws.UpsertWindow(ctx, &model.WebWindow{ProjectID: 1, Source: "gsc", Property: "p", WindowDays: 28, FinalizedThrough: day})
		},
		"web_sync": func() error {
			return ws.PutSync(ctx, 1, "gsc", day)
		},
	}
	for name, step := range steps {
		for i := 0; i < 2; i++ {
			if err := step(); err != nil {
				t.Errorf("%s (run %d): %v", name, i+1, err)
			}
		}
	}
}

// Days east of UTC must not slide back one day in the monitor aggregate.
func TestSQLiteAggregateDaysKeepsLocalDay(t *testing.T) {
	prev := time.Local
	time.Local = time.FixedZone("UTC+8", 8*3600)
	t.Cleanup(func() { time.Local = prev })
	db := openSQLiteFile(t)
	ctx := context.Background()
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	r := &repo.Samples{DB: db}
	if err := r.Upsert(ctx, []model.Sample{{ProjectID: 1, SampledOn: day, Platform: "openai", QID: "q1", Round: 1, SampleMode: "api", OK: true, Mentioned: true}}); err != nil {
		t.Fatal(err)
	}
	rows, err := r.AggregateDays(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Day != "2026-09-01" || rows[0].OK != 1 || rows[0].Mentioned != 1 {
		t.Fatalf("got %+v", rows)
	}
}

func TestSQLiteMigrate(t *testing.T) {
	prev := DB
	t.Cleanup(func() { DB = prev })
	DB = openSQLiteFile(t)
	if err := Migrate(); err != nil {
		t.Fatal(err)
	}
}

// A sampling run can write more rows than fit in one SQLite statement.
func TestSQLiteLargeSampleInsert(t *testing.T) {
	db := openSQLiteFile(t)
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	rows := make([]model.Sample, 2000)
	for i := range rows {
		rows[i] = model.Sample{ProjectID: 1, SampledOn: day, Platform: "openai", QID: "q", Round: i + 1, SampleMode: "api", OK: true}
	}
	if err := (&repo.Samples{DB: db}).Upsert(context.Background(), rows); err != nil {
		t.Fatal(err)
	}
}
