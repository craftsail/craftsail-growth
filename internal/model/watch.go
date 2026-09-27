// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type SavedKeyword struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	ProjectID uint64 `gorm:"uniqueIndex:uk_saved_kw;not null" json:"project_id"`
	Query     string `gorm:"size:512;uniqueIndex:uk_saved_kw;not null" json:"query"`
	Notes     string `gorm:"type:text" json:"notes"`
	CreatedAt int64  `json:"created_at"`
}

func (SavedKeyword) TableName() string { return "saved_keywords" }
