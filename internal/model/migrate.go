// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	models := []any{
		&Project{},
		&Question{},
		&Competitor{},
		&Fact{},
		&Page{},
		&SiteSignal{},
		&Audit{},
		&AuditPage{},
		&AuditIssue{},
		&Sample{},
		&SampleCitation{},
		&SampleRun{},
		&Metric{},
		&Task{},
		&Asset{},
		&Report{},
		&Job{},
		&VerifyReport{},
		&VerifyResult{},
		&GscSitemap{},
		&GscIndex{},
		&GscFact{},
		&GaFact{},
		&GoogleRaw{},
		&WebSyncState{},
		&GscDaily{},
		&GaDaily{},
		&WebImport{},
		&WebProperty{},
		&WebWindow{},
		&SavedKeyword{},
		&User{},
		&ProjectMember{},
		&Session{},
	}
	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			if strings.Contains(err.Error(), "Can't DROP") || strings.Contains(err.Error(), "1091") {
				// On MySQL, GORM stops a table's migration when it tries to
				// drop a unique constraint that was created as an index
				// (uni_<table>_<col>). Columns after that point would never
				// be added, so add them here.
				if err := addMissingColumns(db, m); err != nil {
					return err
				}
				continue
			}
			return err
		}
	}
	return nil
}

// addMissingColumns adds every model field that has no column yet. It is the
// fallback when AutoMigrate aborts early on a table.
func addMissingColumns(db *gorm.DB, m any) error {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(m); err != nil {
		return err
	}
	mig := db.Migrator()
	for _, f := range stmt.Schema.Fields {
		if f.DBName == "" || mig.HasColumn(m, f.DBName) {
			continue
		}
		if err := mig.AddColumn(m, f.Name); err != nil {
			return fmt.Errorf("add column %s.%s: %w", stmt.Schema.Table, f.DBName, err)
		}
	}
	return nil
}
