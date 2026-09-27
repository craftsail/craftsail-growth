// SPDX-License-Identifier: AGPL-3.0-or-later

package model

type Page struct {
	ID         uint64            `gorm:"primaryKey" json:"id"`
	ProjectID  uint64            `gorm:"uniqueIndex:uk_project_url;not null" json:"project_id"`
	URL        string            `gorm:"size:512;uniqueIndex:uk_project_url;not null" json:"url"`
	Title      string            `gorm:"size:512" json:"title"`
	StatusCode int               `json:"status_code"`
	FinalURL   string            `gorm:"size:2048" json:"final_url"`
	HTML       string            `gorm:"type:longtext" json:"html"`
	Text       string            `gorm:"type:mediumtext" json:"text"`
	Headers    map[string]string `gorm:"serializer:json" json:"headers"`
	FetchedAt  *int64            `json:"fetched_at"`
	FetchError string            `gorm:"type:text" json:"fetch_error"`
	WordCount  int               `json:"word_count"`
	Analysis   map[string]any    `gorm:"serializer:json" json:"analysis"`
	CreatedAt  int64             `json:"created_at"`
	UpdatedAt  int64             `json:"updated_at"`
}
