// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// SitemapScan persists discovery independently of HTML crawling and Google history.
type SitemapScan struct {
	ID          uint64 `gorm:"primaryKey" json:"id"`
	ProjectID   uint64 `gorm:"uniqueIndex:uk_sitemap_scan;not null" json:"project_id"`
	KeyHash     string `gorm:"size:64;uniqueIndex:uk_sitemap_scan;not null" json:"-"`
	Property    string `gorm:"type:text" json:"property"`
	URL         string `gorm:"type:text" json:"url"`
	IsIndex     bool   `json:"is_index"`
	Discovered  int    `json:"discovered"`
	FetchedAt   int64  `json:"fetched_at"`
	NextFetchAt int64  `json:"next_fetch_at"`
	LastError   string `gorm:"type:text" json:"last_error"`
}

type SitemapURL struct {
	ID           uint64 `gorm:"primaryKey" json:"id"`
	ProjectID    uint64 `gorm:"uniqueIndex:uk_sitemap_url;not null" json:"project_id"`
	KeyHash      string `gorm:"size:64;uniqueIndex:uk_sitemap_url;not null" json:"-"`
	URLKey       string `gorm:"size:64;index" json:"-"`
	SitemapID    uint64 `gorm:"index" json:"sitemap_id"`
	URL          string `gorm:"type:text" json:"url"`
	LastModified string `gorm:"size:64" json:"last_modified"`
	Present      bool   `json:"present"`
	SeenAt       int64  `json:"seen_at"`
}

// GoogleQuota is shared by every local project using one Google property.
// Reservations survive crashes; uncertain requests are conservatively counted.
type GoogleQuota struct {
	KeyHash      string `gorm:"size:64;primaryKey" json:"-"`
	Property     string `gorm:"type:text" json:"property"`
	Family       string `gorm:"size:32" json:"family"`
	Day          int64  `json:"-"`
	DailyUsed    int    `json:"daily_used"`
	Minute       int64  `json:"-"`
	MinuteUsed   int    `json:"minute_used"`
	BlockedUntil int64  `json:"blocked_until"`
	Reason       string `gorm:"size:32" json:"reason"`
}
