// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "fmt"

// AlertInput uses CrawlSEO's signs: PositionDelta is previous-current, so
// negative means the average position got worse.
type AlertInput struct {
	ClicksDeltaPct float64
	CurrentClicks  float64
	PositionDelta  float64
}

type AlertHit struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// EvaluateAlerts uses the thresholds from crawlseo lib/alerts/evaluate.ts
// (rules of thumb): clicks down 20% or more with at least 5 clicks, or the
// average position worse by 2 or more. Site health is not an alert here;
// audit findings already cover it.
func EvaluateAlerts(in AlertInput) []AlertHit {
	var fires []AlertHit
	if in.ClicksDeltaPct <= -20 && in.CurrentClicks >= 5 {
		fires = append(fires, AlertHit{
			Type:    "traffic_drop",
			Message: fmt.Sprintf("Search clicks changed %+.1f%% versus the previous window (%.0f this window)", in.ClicksDeltaPct, in.CurrentClicks),
		})
	}
	if in.PositionDelta <= -2 {
		fires = append(fires, AlertHit{
			Type:    "position_change",
			Message: fmt.Sprintf("Average search position got worse by %.1f", -in.PositionDelta),
		})
	}
	return fires
}

// alertOps turns period-over-period changes in the official totals into
// search opportunities. Unmeasured or first-window periods raise nothing.
func alertOps(p Period) []SearchOp {
	if !p.Measured || !p.HasPrevious || p.ClicksDelta == nil || p.PositionDelta == nil {
		return nil
	}
	var out []SearchOp
	for _, hit := range EvaluateAlerts(AlertInput{ClicksDeltaPct: *p.ClicksDelta, CurrentClicks: p.Clicks, PositionDelta: *p.PositionDelta}) {
		out = append(out, SearchOp{Type: hit.Type, Title: hit.Message, Detail: "Based on Search Console daily totals for the last 28 days versus the 28 days before.", Severity: "high", Metric: p.Clicks})
	}
	return out
}
