// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "time"

// WebSyncReport is scoped by property and request version. Legacy cursors
// cannot establish coverage and are deliberately not migrated into this table.
type WebSyncReport struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	ProjectID  uint64    `gorm:"uniqueIndex:uk_sync_report;not null" json:"project_id"`
	KeyHash    string    `gorm:"size:64;uniqueIndex:uk_sync_report;not null" json:"-"`
	Property   string    `gorm:"type:text;not null" json:"property"`
	Source     string    `gorm:"size:16;not null" json:"source"`
	Report     string    `gorm:"size:64;not null" json:"report"`
	SearchType string    `gorm:"size:32" json:"search_type"`
	Version    int       `json:"version"`
	Token      string    `gorm:"size:64" json:"-"`
	Revision   int64     `json:"-"`
	State      string    `gorm:"size:32" json:"state"`
	ErrorClass string    `gorm:"size:32" json:"error_class"`
	From       time.Time `gorm:"type:date" json:"from"`
	Through    time.Time `gorm:"type:date" json:"through"`
	UpdatedAt  int64     `json:"updated_at"`
}

// WebSyncDay records successful requests, including empty results. A missing
// row means unmeasured, never zero. Facts and coverage commit together.
type WebSyncDay struct {
	QualityJSON string    `gorm:"type:text" json:"-"`
	ID          uint64    `gorm:"primaryKey" json:"-"`
	ReportID    uint64    `gorm:"uniqueIndex:uk_sync_day;not null" json:"-"`
	Day         time.Time `gorm:"type:date;uniqueIndex:uk_sync_day;not null" json:"day"`
	FetchedAt   int64     `json:"fetched_at"`
}
