// SPDX-License-Identifier: AGPL-3.0-or-later

package integration

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	sqlmysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

// MySQL always creates a new, randomly named database. A DSN naming an existing
// database is rejected. Cleanup drops only the database created by this test.
func acceptanceDB(t *testing.T, dialect string) *gorm.DB {
	t.Helper()
	var db *gorm.DB
	var err error
	config := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if dialect == "sqlite" {
		db, err = gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "accept.db")), config)
	} else {
		raw := os.Getenv("CRAFTSAIL_TEST_MYSQL_DSN")
		if raw == "" {
			t.Skip("set CRAFTSAIL_TEST_MYSQL_DSN to run isolated MySQL 8 acceptance")
		}
		cfg, e := sqlmysql.ParseDSN(raw)
		if e != nil {
			t.Fatal("invalid MySQL acceptance DSN")
		}
		if cfg.DBName != "" {
			t.Fatal("acceptance DSN must not name a database")
		}
		cfg.ParseTime = true
		cfg.Loc = time.UTC
		cfg.Timeout = 10 * time.Second
		admin, e := gorm.Open(mysql.Open(cfg.FormatDSN()), config)
		if e != nil {
			t.Fatalf("connect acceptance server: %v", e)
		}
		adminSQL, e := admin.DB()
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = adminSQL.Close() })
		var version string
		if e = admin.Raw("SELECT VERSION()").Scan(&version).Error; e != nil {
			t.Fatal(e)
		}
		if !strings.HasPrefix(version, "8.") {
			t.Fatalf("MySQL 8 required, got %s", version)
		}
		t.Log("MySQL", version)
		var token [12]byte
		if _, e = rand.Read(token[:]); e != nil {
			t.Fatal(e)
		}
		name := fmt.Sprintf("craftsail_accept_%x", token)
		if e = admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error; e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() {
			if e := admin.Exec("DROP DATABASE `" + name + "`").Error; e != nil {
				t.Errorf("cleanup isolated database: %v", e)
			}
		})
		cfg.DBName = name
		db, err = gorm.Open(mysql.Open(cfg.FormatDSN()), config)
	}
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if dialect == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
	} else {
		sqlDB.SetMaxOpenConns(12)
	}
	for i := 0; i < 2; i++ {
		if err = model.AutoMigrate(db); err != nil {
			t.Fatalf("migration pass %d: %v", i+1, err)
		}
	}
	return db
}
func TestDatabaseAcceptance(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql"} {
		t.Run(dialect, func(t *testing.T) {
			db := acceptanceDB(t, dialect)
			t.Run("google_replacement", func(t *testing.T) { testGoogleReplacement(t, db) })
			t.Run("analysis_identity", func(t *testing.T) { testAnalysisIdentity(t, db) })
			t.Run("concurrent_quota", func(t *testing.T) { testConcurrentQuota(t, db) })
			t.Run("release_and_report", func(t *testing.T) { testReleaseAndReport(t, db) })
			t.Run("index_history", func(t *testing.T) { testIndexHistory(t, db) })
			t.Run("legacy_sample_version_migration", func(t *testing.T) {
				ctx := context.Background()
				day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
				if err := db.Migrator().DropIndex(&model.Sample{}, "uk_sample_v2"); err != nil {
					t.Fatal(err)
				}
				if err := db.Exec("CREATE UNIQUE INDEX uk_sample ON samples(project_id,sampled_on,platform,qid,`round`,sample_mode)").Error; err != nil {
					t.Fatal(err)
				}
				old := model.Sample{ProjectID: 1, SampledOn: day, Platform: "openai", QID: "q1", Round: 1, SampleMode: "api", Answer: "legacy", ManualOverride: true}
				if err := db.Create(&old).Error; err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 2; i++ {
					if err := model.AutoMigrate(db); err != nil {
						t.Fatal(err)
					}
				}
				if db.Migrator().HasIndex(&model.Sample{}, "uk_sample") || !db.Migrator().HasIndex(&model.Sample{}, "uk_sample_v2") {
					t.Fatal("version index migration incomplete")
				}
				next := old
				next.ID = 0
				next.PromptRevision = "v2"
				next.Answer = "new"
				next.ManualOverride = false
				store := &repo.Samples{DB: db}
				if err := store.Upsert(ctx, []model.Sample{next}); err != nil {
					t.Fatal(err)
				}
				old.Answer = "overwrite"
				old.ManualOverride = false
				if err := store.Upsert(ctx, []model.Sample{old}); err != nil {
					t.Fatal(err)
				}
				var rows []model.Sample
				if err := db.Where("project_id = ?", 1).Order("id").Find(&rows).Error; err != nil {
					t.Fatal(err)
				}
				if len(rows) != 2 || rows[0].Answer != "legacy" || rows[1].Answer != "new" {
					t.Fatalf("version/override lost: %+v", rows)
				}
			})
		})
	}
}
