// SPDX-License-Identifier: AGPL-3.0-or-later

// Package opportunity merges the four sources of "what to do next" into one
// ranked list: audit findings, AI citation gaps, search opportunities and
// significant metric drops. It does not compute metrics itself.
package opportunity

import "github.com/craftsail/craftsail-growth/internal/model"

// Item is one thing worth doing. Key is stable across runs so an accepted
// item maps to exactly one task.
type Item struct {
	Score       *model.OpportunityScore `json:"score,omitempty"`
	Recommended bool                    `json:"recommended"`
	Stage       string                  `json:"stage,omitempty"`
	Key         string                  `json:"key"`
	Source      string                  `json:"source"` // audit | citation | search | metric
	Kind        string                  `json:"kind"`
	Priority    string                  `json:"priority"`
	Title       string                  `json:"title"`
	Why         string                  `json:"why"`
	Fix         string                  `json:"fix"`
	Evidence    string                  `json:"evidence,omitempty"`
	Refs        []string                `json:"refs,omitempty"`
	QID         string                  `json:"qid,omitempty"`
	URLs        []string                `json:"urls,omitempty"`
	Acceptance  map[string]any          `json:"acceptance"`
	Detail      map[string]any          `json:"detail,omitempty"`
	// Baseline is copied onto the task when the item is accepted.
	Baseline map[string]any `json:"-"`
	// filled from tasks by Service.List
	Status   string `json:"status,omitempty"`
	TaskCode string `json:"task_code,omitempty"`
}
