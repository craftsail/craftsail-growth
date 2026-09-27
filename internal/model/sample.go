// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "time"

type Sample struct {
	ID                   uint64         `gorm:"primaryKey" json:"id"`
	ProjectID            uint64         `gorm:"uniqueIndex:uk_sample;not null" json:"project_id"`
	SampledOn            time.Time      `gorm:"type:date;uniqueIndex:uk_sample;not null" json:"sampled_on"`
	Platform             string         `gorm:"size:32;uniqueIndex:uk_sample;not null" json:"platform"`
	QID                  string         `gorm:"column:qid;size:16;uniqueIndex:uk_sample;not null" json:"qid"`
	Round                int            `gorm:"uniqueIndex:uk_sample;not null;default:1" json:"round"`
	RunID                *uint64        `gorm:"index" json:"run_id"`
	SampleMode           string         `gorm:"size:16;uniqueIndex:uk_sample;not null;default:api" json:"sample_mode"`
	QuestionText         string         `gorm:"type:text" json:"question_text"`
	Answer               string         `gorm:"type:longtext" json:"answer"`
	Cited                []Citation     `gorm:"serializer:json" json:"cited"`
	Mentioned            bool           `json:"mentioned"`
	Rank                 *int           `json:"rank"`
	CompetitorsMentioned []string       `gorm:"serializer:json" json:"competitors_mentioned"`
	Negative             bool           `json:"negative"`
	ManualOverride       bool           `json:"manual_override"`
	Env                  string         `gorm:"size:32" json:"env"`
	Raw                  map[string]any `gorm:"serializer:json" json:"raw"`
	Flags                []string       `gorm:"serializer:json" json:"flags"`
	OK                   bool           `json:"ok"`
	BrandInQuestion      bool           `json:"brand_in_question"`
	NeedsReview          bool           `json:"needs_review"`
	NegativeCues         []string       `gorm:"serializer:json" json:"negative_cues"`
	CitedDomains         []string       `gorm:"serializer:json" json:"cited_domains"`
	WebQueries           []string       `gorm:"serializer:json" json:"web_queries"`
	OwnDomainCited       bool           `json:"own_domain_cited"`
	Candidates           []string       `gorm:"serializer:json" json:"candidates"`
	Error                string         `gorm:"type:text" json:"error"`
	PlatformName         string         `gorm:"size:64" json:"platform_name"`
	Market               string         `gorm:"size:16" json:"market"`
	CreatedAt            int64          `json:"created_at"`
	UpdatedAt            int64          `json:"updated_at"`
}

type Citation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// SampleCitation is one cited URL from one sample, queryable apart from the sample JSON.
type SampleCitation struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	ProjectID     uint64    `gorm:"index:idx_cite_project_day;not null" json:"project_id"`
	SampleID      uint64    `gorm:"index;not null" json:"sample_id"`
	SampledOn     time.Time `gorm:"type:date;index:idx_cite_project_day;not null" json:"sampled_on"`
	Platform      string    `gorm:"size:32;not null" json:"platform"`
	QID           string    `gorm:"column:qid;size:16" json:"qid"`
	Round         int       `json:"round"`
	SampleMode    string    `gorm:"size:16" json:"sample_mode"`
	URL           string    `gorm:"type:text" json:"url"`
	Domain        string    `gorm:"size:255;index" json:"domain"`
	Title         string    `gorm:"size:512" json:"title"`
	CitationIndex int       `json:"citation_index"`
	Category      string    `gorm:"size:32" json:"category"`
	PageType      string    `gorm:"size:32" json:"page_type"`
	CreatedAt     int64     `json:"created_at"`
}

type Metric struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	ProjectID uint64         `gorm:"uniqueIndex:uk_metric;not null" json:"project_id"`
	MetricOn  time.Time      `gorm:"type:date;uniqueIndex:uk_metric;not null" json:"metric_on"`
	Market    string         `gorm:"size:16;uniqueIndex:uk_metric;not null" json:"market"`
	Payload   map[string]any `gorm:"serializer:json" json:"payload"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}
