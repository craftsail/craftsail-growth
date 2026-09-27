// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTimestampColumnsAreInt64(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	var cols []struct {
		Name string `gorm:"column:name"`
		Type string `gorm:"column:type"`
	}
	if err := db.Raw("PRAGMA table_info(jobs)").Scan(&cols).Error; err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range cols {
		got[c.Name] = strings.ToLower(c.Type)
	}
	for _, name := range []string{"started_at", "finished_at", "created_at", "updated_at"} {
		typ := got[name]
		if typ == "" {
			t.Fatalf("jobs.%s missing", name)
		}
		if strings.Contains(typ, "date") || strings.Contains(typ, "time") || strings.Contains(typ, "char") {
			t.Fatalf("jobs.%s type = %s, want integer", name, typ)
		}
		if !strings.Contains(typ, "int") {
			t.Fatalf("jobs.%s type = %s, want integer", name, typ)
		}
	}

	var dayType string
	if err := db.Raw("SELECT type FROM pragma_table_info('gsc_dailies') WHERE name = 'day'").Scan(&dayType).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(dayType), "date") {
		t.Fatalf("gsc_dailies.day type = %s, want date", dayType)
	}

	before := time.Now().Unix()
	j := Job{Action: "crawl", Status: "queued"}
	if err := db.Create(&j).Error; err != nil {
		t.Fatal(err)
	}
	if j.CreatedAt < before || j.CreatedAt > time.Now().Unix()+1 {
		t.Fatalf("created_at = %d, want unix seconds", j.CreatedAt)
	}
}
