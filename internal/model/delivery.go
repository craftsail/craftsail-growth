// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "time"

type Report struct {
	Language  string    `gorm:"size:8" json:"language"`
	ID        uint64    `gorm:"primaryKey" json:"id"`
	ProjectID uint64    `gorm:"uniqueIndex:uk_report;not null" json:"project_id"`
	ReportOn  time.Time `gorm:"type:date;uniqueIndex:uk_report;not null" json:"report_on"`
	Markdown  string    `gorm:"type:longtext" json:"markdown"`
	HTML      string    `gorm:"type:longtext" json:"html"`
	CreatedAt int64     `json:"created_at"`
	UpdatedAt int64     `json:"updated_at"`
}
