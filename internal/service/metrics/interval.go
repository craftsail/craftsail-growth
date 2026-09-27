// SPDX-License-Identifier: AGPL-3.0-or-later

package metrics

import "math"

// z for a two-sided 95% interval.
const z95 = 1.959963984540054

// Interval is a closed range on the 0..1 scale (or -1..1 for differences).
type Interval struct {
	Lo float64 `json:"lo"`
	Hi float64 `json:"hi"`
}

// Wilson returns the Wilson score interval for x successes in n trials
// (Wilson 1927). It returns nil when there is nothing to estimate.
func Wilson(x, n int) *Interval {
	if n <= 0 || x < 0 || x > n {
		return nil
	}
	fn := float64(n)
	p := float64(x) / fn
	z2 := z95 * z95
	d := 1 + z2/fn
	center := (p + z2/(2*fn)) / d
	half := z95 * math.Sqrt(p*(1-p)/fn+z2/(4*fn*fn)) / d
	return &Interval{Lo: math.Max(0, center-half), Hi: math.Min(1, center+half)}
}

// NewcombeDiff returns the interval for p1 - p2 using Newcombe's hybrid
// score method (method 10 in Newcombe 1998), built from two Wilson intervals.
func NewcombeDiff(x1, n1, x2, n2 int) *Interval {
	a, b := Wilson(x1, n1), Wilson(x2, n2)
	if a == nil || b == nil {
		return nil
	}
	p1 := float64(x1) / float64(n1)
	p2 := float64(x2) / float64(n2)
	d := p1 - p2
	return &Interval{
		Lo: d - math.Sqrt(sq(p1-a.Lo)+sq(b.Hi-p2)),
		Hi: d + math.Sqrt(sq(a.Hi-p1)+sq(p2-b.Lo)),
	}
}

func sq(v float64) float64 { return v * v }

const (
	ChangeUp      = "up"
	ChangeDown    = "down"
	ChangeFlat    = "flat"
	ChangeUnknown = "unknown"
)

// Change compares an earlier period (x1/n1) with a later one (x2/n2).
// It only reports up or down when the 95% interval of the difference
// excludes zero.
func Change(x1, n1, x2, n2 int) string {
	iv := NewcombeDiff(x2, n2, x1, n1)
	if iv == nil {
		return ChangeUnknown
	}
	switch {
	case iv.Lo > 0:
		return ChangeUp
	case iv.Hi < 0:
		return ChangeDown
	default:
		return ChangeFlat
	}
}
