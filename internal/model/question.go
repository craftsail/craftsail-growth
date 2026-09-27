// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Question struct {
	ID         uint64         `gorm:"primaryKey" json:"id"`
	ProjectID  uint64         `gorm:"index;uniqueIndex:uk_project_qid;not null" json:"project_id"`
	QID        string         `gorm:"column:qid;size:16;uniqueIndex:uk_project_qid;not null" json:"qid"`
	GroupName  string         `gorm:"size:32" json:"group"`
	Tags       []string       `gorm:"serializer:json" json:"tags"`
	SystemTags []string       `gorm:"-" json:"system_tags"`
	Market     string         `gorm:"size:16;not null" json:"market"`
	Text       string         `gorm:"type:text;not null" json:"text"`
	Intent     string         `gorm:"size:16" json:"intent"`
	Enabled    bool           `gorm:"not null;default:true" json:"enabled"`
	Diagnosis  *string        `gorm:"size:32" json:"diagnosis"`
	Extra      map[string]any `gorm:"serializer:json" json:"extra"`
	CreatedAt  int64          `json:"created_at"`
	UpdatedAt  int64          `json:"updated_at"`
}

func IntentOf(group string) string {
	switch group {
	case "品牌验证":
		return "probe"
	case "场景", "风险":
		return "educate"
	default:
		return "buyer"
	}
}
