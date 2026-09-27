// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type SiteSignal struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	ProjectID uint64         `gorm:"uniqueIndex;not null" json:"project_id"`
	Payload   map[string]any `gorm:"serializer:json" json:"payload"`
	CrawledAt int64          `json:"crawled_at"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}
