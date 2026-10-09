// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// OpportunityScore is an explicit human estimate, separate from observations.
type OpportunityScore struct {
	ID          uint64  `gorm:"primaryKey" json:"-"`
	ProjectID   uint64  `gorm:"uniqueIndex:uk_opportunity_score;not null" json:"-"`
	KeyHash     string  `gorm:"size:64;uniqueIndex:uk_opportunity_score;not null" json:"-"`
	Key         string  `gorm:"type:text" json:"key"`
	Impact      int     `json:"impact"`
	Confidence  int     `json:"confidence"`
	Ease        int     `json:"ease"`
	EffortHours float64 `json:"effort_hours"`
	UpdatedAt   int64   `json:"updated_at"`
}
