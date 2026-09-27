// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Task struct {
	ID            uint64         `gorm:"primaryKey" json:"id"`
	ProjectID     uint64         `gorm:"uniqueIndex:uk_task_code;uniqueIndex:uk_task_source,priority:1;not null" json:"project_id"`
	Code          string         `gorm:"size:16;uniqueIndex:uk_task_code;not null" json:"code"`
	Priority      string         `gorm:"size:8" json:"priority"`
	Package       string         `gorm:"size:32" json:"package"`
	Market        string         `gorm:"size:16" json:"market"`
	Title         string         `gorm:"size:255" json:"title"`
	Why           string         `gorm:"type:text" json:"why"`
	Action        string         `gorm:"type:text" json:"action"`
	Owner         string         `gorm:"size:32" json:"owner"`
	Effort        string         `gorm:"size:8" json:"effort"`
	Window        string         `gorm:"size:32" json:"window"`
	Risk          string         `gorm:"size:16" json:"risk"`
	Acceptance    map[string]any `gorm:"serializer:json" json:"acceptance"`
	Status        string         `gorm:"size:16;not null;default:open" json:"status"`
	Source        string         `gorm:"size:16;not null;default:legacy" json:"source"` // audit | citation | search | metric | legacy
	SourceKey     *string        `gorm:"size:191;uniqueIndex:uk_task_source,priority:2" json:"source_key"`
	Baseline      map[string]any `gorm:"serializer:json" json:"baseline"`
	VerifiedAt    *int64         `json:"verified_at"`
	Affected      []string       `gorm:"serializer:json" json:"affected"`
	Assets        []string       `gorm:"serializer:json" json:"assets"`
	Evidence      []string       `gorm:"serializer:json" json:"evidence"`
	Progress      map[string]any `gorm:"serializer:json" json:"progress"`
	ProgressFirst map[string]any `gorm:"serializer:json" json:"progress_first"`
	BaselineCount int            `json:"baseline_count"`
	ClosedAt      *int64         `json:"closed_at"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
}

const (
	TaskOpen      = "open"
	TaskDoing     = "doing"
	TaskDone      = "done"
	TaskVerified  = "verified"
	TaskRegressed = "regressed"
	TaskDismissed = "dismissed"
)
