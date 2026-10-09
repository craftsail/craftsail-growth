// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Audit struct {
	ID                uint64           `gorm:"primaryKey" json:"id"`
	ProjectID         uint64           `gorm:"index;not null" json:"project_id"`
	RunAt             int64            `json:"run_at"`
	AvgScore          float64          `json:"avg_score"`
	PageCount         int              `json:"page_count"`
	GradeDistribution map[string]int   `gorm:"serializer:json" json:"grade_distribution"`
	Site              map[string]any   `gorm:"serializer:json" json:"site"`
	Layers            []map[string]any `gorm:"serializer:json" json:"layers"`
	NoSite            bool             `json:"no_site"`
	BlockGap          []map[string]any `gorm:"serializer:json" json:"block_gap"`
	LanguageCoverage  map[string]any   `gorm:"serializer:json" json:"language_coverage"`
	KeywordsUsed      []string         `gorm:"serializer:json" json:"keywords_used"`
	Market            string           `gorm:"size:16" json:"market"`
	CreatedAt         int64            `json:"created_at"`
	UpdatedAt         int64            `json:"updated_at"`
}

type AuditPage struct {
	CrawledAt   int64          `json:"crawled_at"`
	ID          uint64         `gorm:"primaryKey" json:"id"`
	AuditID     uint64         `gorm:"index;not null" json:"audit_id"`
	PageID      uint64         `gorm:"index;not null" json:"page_id"`
	Score       float64        `json:"score"`
	Grade       string         `gorm:"size:8" json:"grade"`
	Dimensions  map[string]any `gorm:"serializer:json" json:"dimensions"`
	IssueCodes  []string       `gorm:"serializer:json" json:"issue_codes"`
	Blocks      map[string]any `gorm:"serializer:json" json:"blocks"`
	URL         string         `gorm:"size:512" json:"url"`
	Title       string         `gorm:"size:255" json:"title"`
	WordCount   int            `json:"word_count"`
	JSONLDTypes []string       `gorm:"serializer:json" json:"jsonld_types"`
	CreatedAt   int64          `json:"created_at"`
	UpdatedAt   int64          `json:"updated_at"`
}
