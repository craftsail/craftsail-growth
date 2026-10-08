// SPDX-License-Identifier: AGPL-3.0-or-later

package model

// GoogleQuality describes successfully returned data, not request coverage.
// Provider metadata remains in raw responses; this summary survives raw pruning.
type GoogleQuality struct {
	Known        bool     `json:"known"`
	Sampled      bool     `json:"sampled"`
	Thresholded  bool     `json:"thresholded"`
	OtherRow     bool     `json:"other_row"`
	Restricted   bool     `json:"restricted"`
	EmptyReason  bool     `json:"empty_reason"`
	Currencies   []string `json:"currencies,omitempty"`
	TimeZones    []string `json:"time_zones,omitempty"`
	Aggregations []string `json:"aggregations,omitempty"`
}
