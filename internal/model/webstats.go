// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// RowKey is the hex SHA-256 of parts joined by newline.
// It is the exact last-write-wins key so MySQL unique indexes stay under 3072 bytes.
func RowKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

// GscSitemap is one Search Console sitemap. Indexed is deprecated by Google and often zero.
type GscSitemap struct {
	ID        uint64  `gorm:"primaryKey" json:"id"`
	ProjectID uint64  `gorm:"uniqueIndex:uk_gsc_sitemap;not null" json:"project_id"`
	KeyHash   string  `gorm:"size:64;uniqueIndex:uk_gsc_sitemap;not null" json:"key_hash"`
	Path      string  `gorm:"type:text;not null" json:"path"`
	Submitted float64 `json:"submitted"`
	Errors    float64 `json:"errors"`
	Warnings  float64 `json:"warnings"`
	Pending   bool    `json:"pending"`
	Raw       string  `gorm:"type:longtext" json:"raw"`
	FetchedAt int64   `json:"fetched_at"`
}

func (GscSitemap) TableName() string { return "gsc_sitemaps" }

// GscIndex is one URL Inspection result. Verdict PASS means Google indexed it.
type GscIndex struct {
	ID            uint64 `gorm:"primaryKey" json:"id"`
	ProjectID     uint64 `gorm:"uniqueIndex:uk_gsc_index;not null" json:"project_id"`
	KeyHash       string `gorm:"size:64;uniqueIndex:uk_gsc_index;not null" json:"key_hash"`
	URL           string `gorm:"type:text;not null" json:"url"`
	Verdict       string `gorm:"size:32" json:"verdict"`
	CoverageState string `gorm:"size:255" json:"coverage_state"`
	IndexingState string `gorm:"size:64" json:"indexing_state"`
	LastCrawl     string `gorm:"size:40" json:"last_crawl"`
	Raw           string `gorm:"type:longtext" json:"raw"`
	FetchedAt     int64  `json:"fetched_at"`
}

func (GscIndex) TableName() string { return "gsc_indexes" }

type GscFact struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	ProjectID        uint64    `gorm:"uniqueIndex:uk_gsc_fact;not null" json:"project_id"`
	Property         string    `gorm:"size:255;uniqueIndex:uk_gsc_fact;not null;default:''" json:"property"`
	Slice            string    `gorm:"size:64;uniqueIndex:uk_gsc_fact;not null" json:"slice"`
	SearchType       string    `gorm:"size:32;not null" json:"search_type"`
	Day              time.Time `gorm:"type:date;uniqueIndex:uk_gsc_fact;not null" json:"day"`
	Hour             string    `gorm:"size:2" json:"hour"`
	KeyHash          string    `gorm:"size:64;uniqueIndex:uk_gsc_fact;not null" json:"key_hash"`
	Query            string    `gorm:"type:text" json:"query"`
	Page             string    `gorm:"type:text" json:"page"`
	Country          string    `gorm:"size:8" json:"country"`
	Device           string    `gorm:"size:16" json:"device"`
	SearchAppearance string    `gorm:"size:64" json:"search_appearance"`
	Clicks           float64   `json:"clicks"`
	Impressions      float64   `json:"impressions"`
	CTR              float64   `json:"ctr"`
	Position         float64   `json:"position"`
	FetchedAt        int64     `json:"fetched_at"`
}

func (GscFact) TableName() string { return "gsc_facts" }

type GaFact struct {
	EngagementDuration *float64  `json:"engagement_duration"`
	EventValue         *float64  `json:"event_value"`
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	ProjectID          uint64    `gorm:"uniqueIndex:uk_ga_fact;not null" json:"project_id"`
	Property           string    `gorm:"size:64;uniqueIndex:uk_ga_fact;not null;default:''" json:"property"`
	Report             string    `gorm:"size:32;uniqueIndex:uk_ga_fact;not null" json:"report"`
	Day                time.Time `gorm:"type:date;uniqueIndex:uk_ga_fact;not null" json:"day"`
	Hour               string    `gorm:"size:2" json:"hour"`
	KeyHash            string    `gorm:"size:64;uniqueIndex:uk_ga_fact;not null" json:"key_hash"`
	Source             string    `gorm:"type:text" json:"source"`
	Medium             string    `gorm:"type:text" json:"medium"`
	Campaign           string    `gorm:"type:text" json:"campaign"`
	Channel            string    `gorm:"size:128" json:"channel"`
	Landing            string    `gorm:"type:text" json:"landing"`
	PagePath           string    `gorm:"type:text" json:"page_path"`
	PageTitle          string    `gorm:"type:text" json:"page_title"`
	Country            string    `gorm:"size:128" json:"country"`
	Device             string    `gorm:"size:32" json:"device"`
	EventName          string    `gorm:"size:255" json:"event_name"`
	Sessions           float64   `json:"sessions"`
	Engaged            float64   `json:"engaged"`
	ActiveUsers        float64   `json:"active_users"`
	NewUsers           float64   `json:"new_users"`
	Views              float64   `json:"views"`
	EventCount         float64   `json:"event_count"`
	KeyEvents          float64   `json:"key_events"`
	Revenue            float64   `json:"revenue"`
	EngagementRate     float64   `json:"engagement_rate"`
	BounceRate         float64   `json:"bounce_rate"`
	FetchedAt          int64     `json:"fetched_at"`
}

func (GaFact) TableName() string { return "ga_facts" }

type GoogleRaw struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	ProjectID uint64 `gorm:"index;not null" json:"project_id"`
	Source    string `gorm:"size:32;index;not null" json:"source"`
	Report    string `gorm:"size:64;not null" json:"report"`
	Request   string `gorm:"type:longtext" json:"request"`
	Body      string `gorm:"type:longtext" json:"body"`
	FetchedAt int64  `json:"fetched_at"`
}

func (GoogleRaw) TableName() string { return "google_raws" }

type WebSyncState struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	ProjectID uint64    `gorm:"uniqueIndex:uk_web_sync;not null" json:"project_id"`
	Source    string    `gorm:"size:64;uniqueIndex:uk_web_sync;not null" json:"source"`
	LastEnd   time.Time `gorm:"type:date" json:"last_end"`
	UpdatedAt int64     `json:"updated_at"`
}

// GscDaily is the date-only Search Console total. It is never rebuilt by summing query rows.
type GscDaily struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	ProjectID   uint64    `gorm:"uniqueIndex:uk_gsc_daily;not null" json:"project_id"`
	Property    string    `gorm:"size:512;uniqueIndex:uk_gsc_daily;not null" json:"property"`
	SearchType  string    `gorm:"size:32;uniqueIndex:uk_gsc_daily;not null;default:web" json:"search_type"`
	Day         time.Time `gorm:"type:date;uniqueIndex:uk_gsc_daily;not null" json:"day"`
	Clicks      float64   `json:"clicks"`
	Impressions float64   `json:"impressions"`
	CTR         float64   `json:"ctr"`
	Position    float64   `json:"position"`
	FetchedAt   int64     `json:"fetched_at"`
}

func (GscDaily) TableName() string { return "gsc_dailies" }

// GaDaily is the date-only GA4 total. Nil metrics were not in that response; they are not zero.
type GaDaily struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	ProjectID   uint64    `gorm:"uniqueIndex:uk_ga_daily;not null" json:"project_id"`
	Property    string    `gorm:"size:32;uniqueIndex:uk_ga_daily;not null" json:"property"`
	Day         time.Time `gorm:"type:date;uniqueIndex:uk_ga_daily;not null" json:"day"`
	Sessions    float64   `json:"sessions"`
	Engaged     *float64  `json:"engaged,omitempty"`
	KeyEvents   *float64  `json:"key_events,omitempty"`
	Conversions *float64  `json:"conversions,omitempty"`
	Timezone    string    `gorm:"size:64" json:"timezone"`
	FetchedAt   int64     `json:"fetched_at"`
}

func (GaDaily) TableName() string { return "ga_dailies" }

// WebImport is the freshness row the monitoring report reads. Disconnecting does not delete dailies.
type WebImport struct {
	ID               uint64     `gorm:"primaryKey" json:"id"`
	ProjectID        uint64     `gorm:"uniqueIndex:uk_web_import;not null" json:"project_id"`
	Source           string     `gorm:"size:16;uniqueIndex:uk_web_import;not null" json:"source"`
	Property         string     `gorm:"size:512" json:"property"`
	State            string     `gorm:"size:32" json:"state"`
	PausedReason     string     `gorm:"size:32" json:"paused_reason"`
	CursorDate       *time.Time `gorm:"type:date" json:"cursor_date,omitempty"`
	FinalizedThrough *time.Time `gorm:"type:date" json:"finalized_through,omitempty"`
	BoundarySource   string     `gorm:"size:16" json:"boundary_source"`
	LastError        string     `gorm:"type:text" json:"last_error"`
	LastErrorClass   string     `gorm:"size:32" json:"last_error_class"`
	RowsRequested    int        `json:"rows_requested"`
	RowsReturned     int        `json:"rows_returned"`
	CapHit           bool       `json:"cap_hit"`
	UpdatedAt        int64      `json:"updated_at"`
}

func (WebImport) TableName() string { return "web_imports" }

// WebProperty archives a previous resource key so its daily rows stay readable.
type WebProperty struct {
	SearchEstablishedThrough *time.Time `gorm:"type:date" json:"search_established_through,omitempty"`
	SearchStageVersion       int        `json:"search_stage_version"`
	ID                       uint64     `gorm:"primaryKey" json:"id"`
	ProjectID                uint64     `gorm:"uniqueIndex:uk_web_property;not null" json:"project_id"`
	Source                   string     `gorm:"size:16;uniqueIndex:uk_web_property;not null" json:"source"`
	PropertyKey              string     `gorm:"size:512;uniqueIndex:uk_web_property;not null" json:"property_key"`
	Status                   string     `gorm:"size:16;index" json:"status"`
	Timezone                 string     `gorm:"size:64" json:"timezone"`
	ActivatedAt              int64      `json:"activated_at"`
	ArchivedAt               *int64     `json:"archived_at,omitempty"`
}

func (WebProperty) TableName() string { return "web_properties" }

// WebWindow is the 28-day monitoring total written from date-only rows.
type WebWindow struct {
	ID                  uint64    `gorm:"primaryKey" json:"id"`
	ProjectID           uint64    `gorm:"uniqueIndex:uk_web_window;not null" json:"project_id"`
	Source              string    `gorm:"size:16;uniqueIndex:uk_web_window;not null" json:"source"`
	Property            string    `gorm:"size:512;uniqueIndex:uk_web_window;not null" json:"property"`
	WindowDays          int       `gorm:"uniqueIndex:uk_web_window;not null" json:"window_days"`
	FinalizedThrough    time.Time `gorm:"type:date;uniqueIndex:uk_web_window;not null" json:"finalized_through"`
	Clicks              float64   `json:"clicks"`
	Impressions         float64   `json:"impressions"`
	Sessions            float64   `json:"sessions"`
	PreviousClicks      float64   `json:"previous_clicks"`
	PreviousImpressions float64   `json:"previous_impressions"`
	PreviousSessions    float64   `json:"previous_sessions"`
	CoveredDays         int       `json:"covered_days"`
	ComputedAt          int64     `json:"computed_at"`
}

func (WebWindow) TableName() string { return "web_windows" }

func (WebSyncState) TableName() string { return "web_sync_states" }
