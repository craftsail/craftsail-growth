// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"time"
)

type AlertInput struct {
	CurrentClicks, PreviousClicks float64
	Covered, Sustained            bool
}
type AlertHit struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// These are conservative product heuristics, not significance tests or Google
// rules. Aggregate position changes are never evidence of a traffic incident.
func EvaluateAlerts(in AlertInput) []AlertHit {
	if !in.Covered || !in.Sustained || !meaningfulClickDrop(in.CurrentClicks, in.PreviousClicks) {
		return nil
	}
	return []AlertHit{{Type: "traffic_drop", Message: fmt.Sprintf("Search clicks fell from %.0f to %.0f across fully covered 28-day windows, with declines in both recent weeks. Inspect affected pages before deciding on a change.", in.PreviousClicks, in.CurrentClicks)}}
}
func meaningfulClickDrop(current, previous float64) bool {
	return previous >= 100 && current >= 0 && previous-current >= 50 && current <= previous*0.75
}
func sustainedDrop(current, previous func(time.Time, time.Time) float64, through time.Time) bool {
	for i := 0; i < 2; i++ {
		end := through.AddDate(0, 0, -7*i)
		start := end.AddDate(0, 0, -6)
		cur := current(start, end)
		prev := previous(start.AddDate(0, 0, -28), end.AddDate(0, 0, -28))
		if !weeklyDrop(cur, prev) {
			return false
		}
	}
	return true
}
func alertOps(p Period, rows []model.GscDaily, through time.Time) []SearchOp {
	sum := func(from, to time.Time) float64 { return sumOfficial(rows, from, to).clicks }
	var out []SearchOp
	for _, hit := range EvaluateAlerts(AlertInput{CurrentClicks: p.Clicks, PreviousClicks: p.PreviousClicks, Covered: p.Comparable, Sustained: sustainedDrop(sum, sum, through)}) {
		out = append(out, SearchOp{Type: hit.Type, Title: hit.Message, Detail: hit.Message, Severity: "medium", Metric: p.Clicks, Reason: "sustained_decline", Facts: map[string]float64{"current": p.Clicks, "previous": p.PreviousClicks}})
	}
	return out
}

func weeklyDrop(current, previous float64) bool {
	return previous >= 20 && previous-current >= 10 && current <= previous*0.75
}
