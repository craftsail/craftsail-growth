// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// SampleRun is one sampling batch. Status says how far it got; Outcome says
// how it went. A partial run can be retried for its failed pairs only.
type SampleRun struct {
	ID         uint64  `gorm:"primaryKey" json:"id"`
	ProjectID  uint64  `gorm:"index:idx_run_project,priority:1;not null" json:"project_id"`
	JobID      *uint64 `json:"job_id"`
	ParentID   *uint64 `json:"parent_id"`                         // set when this run retries another run's failures
	Trigger    string  `gorm:"size:16;not null" json:"trigger"`   // manual | schedule | import | retry
	Status     string  `gorm:"size:16;not null" json:"status"`    // running | completed | cancelled
	Outcome    string  `gorm:"size:16" json:"outcome"`            // succeeded | partial | failed
	Planned    int     `gorm:"not null;default:0" json:"planned"` // prompt x engine x round calls
	Succeeded  int     `gorm:"not null;default:0" json:"succeeded"`
	Failed     int     `gorm:"not null;default:0" json:"failed"`
	EstTokens  int     `gorm:"not null;default:0" json:"est_tokens"`  // planning estimate, not a bill
	UsedTokens int     `gorm:"not null;default:0" json:"used_tokens"` // 0 when providers do not report usage
	StartedAt  int64   `json:"started_at"`
	FinishedAt *int64  `json:"finished_at"`
	CreatedAt  int64   `gorm:"index:idx_run_project,priority:2" json:"created_at"`
}
