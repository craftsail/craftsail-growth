// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Job struct {
	ID         uint64         `gorm:"primaryKey" json:"id"`
	ProjectID  *uint64        `gorm:"index" json:"project_id"`
	Action     string         `gorm:"size:32;not null" json:"action"`
	Status     string         `gorm:"size:16;not null" json:"status"`
	Args       map[string]any `gorm:"serializer:json" json:"args"`
	Log        string         `gorm:"type:longtext" json:"log"`
	StartedAt  *int64         `json:"started_at"`
	FinishedAt *int64         `json:"finished_at"`
	Error      string         `gorm:"type:text" json:"error"`
	CreatedAt  int64          `json:"created_at"`
	UpdatedAt  int64          `json:"updated_at"`
}

type VerifyReport struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	ProjectID uint64         `gorm:"index;not null" json:"project_id"`
	RunAt     int64          `json:"run_at"`
	Changed   int            `json:"changed"`
	Summary   map[string]any `gorm:"serializer:json" json:"summary"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

type VerifyResult struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	VerifyReportID uint64         `gorm:"index;not null" json:"verify_report_id"`
	TaskCode       string         `gorm:"size:16" json:"task_code"`
	Title          string         `gorm:"size:255" json:"title"`
	Priority       string         `gorm:"size:8" json:"priority"`
	Verdict        string         `gorm:"size:16" json:"verdict"`
	Note           string         `gorm:"type:text" json:"note"`
	Was            string         `gorm:"size:16" json:"was"`
	Now            string         `gorm:"size:16" json:"now"`
	Progress       map[string]any `gorm:"serializer:json" json:"progress"`
	CreatedAt      int64          `json:"created_at"`
	UpdatedAt      int64          `json:"updated_at"`
}
