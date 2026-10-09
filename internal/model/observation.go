// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// Observation is an immutable release contract. A correction creates a new
// record; task completion never implies a measured effect.
type Observation struct {
	ID              uint64              `gorm:"primaryKey" json:"id"`
	ProjectID       uint64              `gorm:"index;not null" json:"project_id"`
	TaskID          uint64              `gorm:"index;not null" json:"task_id"`
	TaskCode        string              `gorm:"size:16" json:"task_code"`
	Hypothesis      string              `gorm:"type:text" json:"hypothesis"`
	Metric          string              `gorm:"size:32" json:"metric"`
	Guardrails      string              `gorm:"type:text" json:"guardrails"`
	URLs            []string            `gorm:"serializer:json" json:"urls"`
	QIDs            []string            `gorm:"serializer:json" json:"qids"`
	ControlURLs     []string            `gorm:"serializer:json" json:"control_urls"`
	Owner           string              `gorm:"size:128" json:"owner"`
	EffortHours     float64             `json:"effort_hours"`
	Notes           string              `gorm:"type:text" json:"notes"`
	ReleasedAt      int64               `json:"released_at"`
	WaitDays        int                 `json:"wait_days"`
	WindowDays      int                 `json:"window_days"`
	BaselineFrom    string              `gorm:"size:10" json:"baseline_from"`
	BaselineThrough string              `gorm:"size:10" json:"baseline_through"`
	FollowupFrom    string              `gorm:"size:10" json:"followup_from"`
	FollowupThrough string              `gorm:"size:10" json:"followup_through"`
	Property        string              `gorm:"type:text" json:"property"`
	Engine          string              `gorm:"size:32" json:"engine"`
	Access          string              `gorm:"size:16" json:"access"`
	PromptRevision  string              `gorm:"size:64" json:"prompt_revision"`
	Baseline        Measurement         `gorm:"serializer:json" json:"baseline"`
	ControlBaseline Measurement         `gorm:"serializer:json" json:"control_baseline"`
	CreatedAt       int64               `json:"created_at"`
	Results         []ObservationResult `gorm:"-" json:"results"`
}
type Measurement struct {
	Valid     bool              `json:"valid"`
	Reason    string            `json:"reason"`
	Value     float64           `json:"value"`
	Exposure  float64           `json:"exposure"`
	Count     int               `json:"count"`
	Signature string            `json:"signature"`
	Strata    map[string][2]int `json:"strata,omitempty"`
	AuditAt   int64             `json:"audit_at,omitempty"`
}
type ObservationResult struct {
	ID              uint64      `gorm:"primaryKey" json:"id"`
	ProjectID       uint64      `gorm:"index;not null" json:"project_id"`
	ObservationID   uint64      `gorm:"index;not null" json:"observation_id"`
	Conclusion      string      `gorm:"size:32" json:"conclusion"`
	Reason          string      `gorm:"size:64" json:"reason"`
	Followup        Measurement `gorm:"serializer:json" json:"followup"`
	ControlFollowup Measurement `gorm:"serializer:json" json:"control_followup"`
	Notes           string      `gorm:"type:text" json:"notes"`
	CreatedAt       int64       `json:"created_at"`
}
