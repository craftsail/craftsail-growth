// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// When AutoMigrate aborts on a table, the fallback must add the columns it
// did not reach.
func TestAddMissingColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE projects (id integer primary key, slug text, name text)").Error; err != nil {
		t.Fatal(err)
	}
	if err := addMissingColumns(db, &Project{}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"monitor_runs_per_day", "monitor_every_days", "gsc_site", "brand"} {
		if !db.Migrator().HasColumn(&Project{}, c) {
			t.Fatalf("column %s was not added", c)
		}
	}
}
