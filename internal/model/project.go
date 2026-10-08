// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "time"

type Brand struct {
	Aliases        []string    `json:"aliases"`
	Products       []string    `json:"products"`
	Industry       string      `json:"industry"`
	TargetUsers    string      `json:"target_users"`
	BusinessGoal   string      `json:"business_goal"`
	Definition     string      `json:"definition"`
	Disambiguation []string    `json:"disambiguation"`
	Offers         []Offer     `json:"offers"`
	KeyNumbers     []KeyNumber `json:"key_numbers"`
	Suitable       []string    `json:"suitable"`
	Unsuitable     []string    `json:"unsuitable"`
	Uncertain      []string    `json:"uncertain"`
}

type Offer struct {
	Name     string `json:"name"`
	Price    string `json:"price"`
	Currency string `json:"currency"`
	Desc     string `json:"desc"`
}

type KeyNumber struct {
	Fact   string `json:"fact"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

type BootstrapMeta struct {
	At          string   `json:"at"`
	Source      string   `json:"source"`
	Uncertain   []string `json:"uncertain"`
	NeedsReview bool     `json:"needs_review"`
}

type Targets struct {
	MentionRate  float64 `json:"mention_rate"`
	Top3Rate     float64 `json:"top3_rate"`
	AvgPageScore float64 `json:"avg_page_score"`
}

type Project struct {
	SamplingLanguage     string         `gorm:"size:8;not null;default:''" json:"sampling_language"`
	SiteLanguage         string         `gorm:"size:8;not null;default:''" json:"site_language"`
	TargetRegion         string         `gorm:"size:64;not null;default:''" json:"target_region"`
	ReportLanguage       string         `gorm:"size:8;not null;default:en" json:"report_language"`
	SearchMode           string         `gorm:"size:16;not null;default:auto" json:"search_mode"`
	SearchMinImpressions int            `gorm:"not null;default:500" json:"search_min_impressions"`
	ID                   uint64         `gorm:"primaryKey" json:"id"`
	Slug                 string         `gorm:"size:48;uniqueIndex;not null" json:"slug"`
	Name                 string         `gorm:"size:80;not null" json:"name"`
	Site                 string         `gorm:"size:512" json:"site"`
	GscSite              string         `gorm:"size:512" json:"gsc_site"`
	GA4Property          string         `gorm:"size:32" json:"ga4_property"`
	Market               string         `gorm:"size:16;not null" json:"market"`
	NoSite               bool           `json:"no_site"`
	Brand                Brand          `gorm:"serializer:json" json:"brand"`
	Platforms            []string       `gorm:"serializer:json" json:"platforms"`
	PagesSeed            []string       `gorm:"serializer:json" json:"pages_seed"`
	PagesMax             int            `json:"pages_max"`
	Targets              Targets        `gorm:"serializer:json" json:"targets"`
	Materials            string         `gorm:"type:longtext" json:"materials"`
	Notes                string         `gorm:"type:text" json:"notes"`
	Bootstrap            *BootstrapMeta `gorm:"serializer:json" json:"bootstrap"`
	MonitorEveryDays     *int           `json:"monitor_every_days"`
	MonitorNextRun       *time.Time     `gorm:"type:date" json:"monitor_next_run"`
	// MonitorRunsPerDay is the target number of runs per prompt and engine per day.
	MonitorRunsPerDay int   `gorm:"not null;default:3" json:"monitor_runs_per_day"`
	CreatedAt         int64 `json:"created_at"`
	UpdatedAt         int64 `json:"updated_at"`
}

// MarketAll is stored in the legacy market columns. Projects are no longer
// split by market: every enabled prompt goes to every engine.
const MarketAll = "all"

// DefaultPlatforms lists every engine a project samples. Engines without a
// key are skipped at run time.
func DefaultPlatforms() []string {
	return []string{
		"glm", "doubao", "deepseek", "kimi", "minimax",
		"gemini", "openai", "claude", "grok", "perplexity",
		"nano_ai", "baidu", "chatgpt", "google_aio",
	}
}

func DefaultTargets() Targets {
	return Targets{MentionRate: 0.5, Top3Rate: 0.3, AvgPageScore: 75}
}

// MentionTarget returns the project's target mention rate, falling back to
// the single default used everywhere.
func (t Targets) MentionTarget() float64 {
	if t.MentionRate > 0 {
		return t.MentionRate
	}
	return DefaultTargets().MentionRate
}
