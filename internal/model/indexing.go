// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// IndexURL is a durable inventory entry. FirstSeenAt is discovery, never publication.
// KeyHash includes property and URL so changing properties cannot inherit a verdict.
type IndexURL struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ProjectID      uint64    `gorm:"uniqueIndex:uk_index_url;not null" json:"project_id"`
	KeyHash        string    `gorm:"size:64;uniqueIndex:uk_index_url;not null" json:"key_hash"`
	Property       string    `gorm:"type:text" json:"property"`
	URL            string    `gorm:"type:text" json:"url"`
	FromCrawl      bool      `json:"from_crawl"`
	FromSearch     bool      `json:"from_search"`
	FirstSeenAt    int64     `json:"first_seen_at"`
	LastSeenAt     int64     `json:"last_seen_at"`
	LastAttemptAt  int64     `json:"last_attempt_at"`
	LastSuccessAt  int64     `json:"last_success_at"`
	FirstIndexedAt *int64    `json:"first_indexed_at"`
	NextInspectAt  int64     `gorm:"index" json:"next_inspect_at"`
	Verdict        string    `gorm:"size:32" json:"verdict"`
	LastError      string    `gorm:"type:text" json:"last_error"`
	Latest         *GscIndex `gorm:"-" json:"latest"`
}

// IndexInspection keeps each attempt, including failures which never replace a
// successful result. Result retains the complete provider response for later analysis.
type IndexInspection struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	ProjectID uint64    `gorm:"index:idx_inspection_url;not null" json:"project_id"`
	KeyHash   string    `gorm:"size:64;index:idx_inspection_url;not null" json:"-"`
	Property  string    `gorm:"type:text" json:"property"`
	URL       string    `gorm:"type:text" json:"url"`
	CheckedAt int64     `json:"checked_at"`
	Error     string    `gorm:"type:text" json:"error"`
	Result    *GscIndex `gorm:"serializer:json;type:longtext" json:"result"`
}
