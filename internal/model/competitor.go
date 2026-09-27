// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Competitor struct {
	ID        uint64   `gorm:"primaryKey" json:"id"`
	ProjectID uint64   `gorm:"index;not null" json:"project_id"`
	Name      string   `gorm:"size:80;not null" json:"name"`
	Site      string   `gorm:"size:512" json:"site"`
	Aliases   []string `gorm:"serializer:json" json:"aliases"`
	Market    string   `gorm:"size:16;not null" json:"market"`
	Confirmed *bool    `json:"confirmed"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

func CompetitorConfirmed(c Competitor) bool {
	return c.Confirmed == nil || *c.Confirmed
}
