// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Fact struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	ProjectID uint64 `gorm:"uniqueIndex;not null" json:"project_id"`
	Markdown  string `gorm:"type:longtext" json:"markdown"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type Asset struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	ProjectID uint64 `gorm:"uniqueIndex:uk_project_apath;not null" json:"project_id"`
	Path      string `gorm:"size:255;uniqueIndex:uk_project_apath;not null" json:"path"`
	Content   string `gorm:"type:longtext" json:"content"`
	Kind      string `gorm:"size:32" json:"kind"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
