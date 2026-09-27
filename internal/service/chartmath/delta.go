// SPDX-License-Identifier: AGPL-3.0-or-later

package chartmath

import "math"

type Delta struct {
	Kind  string
	Dir   string
	Value float64
}

type TakeawayResult struct {
	Kind  string
	Days  int
	Value float64
}

func CountDelta(current, previous float64) Delta {
	if previous <= 0 {
		if current > 0 {
			return Delta{Kind: "new", Dir: "up"}
		}
		return Delta{Kind: "unchanged", Dir: "flat"}
	}
	change := round((current-previous)/previous, 3)
	if change == 0 {
		return Delta{Kind: "unchanged", Dir: "flat"}
	}
	dir := "down"
	if change > 0 {
		dir = "up"
	}
	return Delta{Kind: "changed", Dir: dir, Value: change}
}

// Takeaway compares the first three points with the last three.
// higherIsBetter is false for rank. Fewer than 7 points returns an empty result.
func Takeaway(points []float64, higherIsBetter bool) TakeawayResult {
	if len(points) < 7 {
		return TakeawayResult{}
	}
	start := avg(points[:3])
	end := avg(points[len(points)-3:])
	signed := end - start
	if !higherIsBetter {
		signed = start - end
	}
	delta := round(math.Abs(signed), 1)
	if delta < 0.1 {
		return TakeawayResult{Kind: "steady", Days: len(points)}
	}
	kind := "slipped"
	if signed > 0 {
		kind = "improved"
	}
	return TakeawayResult{Kind: kind, Days: len(points), Value: delta}
}

func avg(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func round(value float64, digits int) float64 {
	pow := math.Pow(10, float64(digits))
	next := math.Round(value*pow) / pow
	if next == 0 {
		return 0
	}
	return next
}
