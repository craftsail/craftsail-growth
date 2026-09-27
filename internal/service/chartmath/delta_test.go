// SPDX-License-Identifier: AGPL-3.0-or-later

package chartmath

import "testing"

func TestCountDelta(t *testing.T) {
	d := CountDelta(12, 0)
	if d.Kind != "new" || d.Dir != "up" {
		t.Fatalf("%+v", d)
	}
	d = CountDelta(0, 0)
	if d.Kind != "unchanged" || d.Dir != "flat" {
		t.Fatalf("%+v", d)
	}
	d = CountDelta(110, 100)
	if d.Kind != "changed" || d.Dir != "up" || d.Value != 0.1 {
		t.Fatalf("%+v", d)
	}
	d = CountDelta(100.0004, 100)
	if d.Kind != "unchanged" {
		t.Fatalf("tiny change should not sign: %+v", d)
	}
}

func TestTrendTakeawayHigherIsBetter(t *testing.T) {
	if Takeaway(nil, true) != (TakeawayResult{}) {
		t.Fatal("empty")
	}
	short := []float64{0.2, 0.4}
	if Takeaway(short, true).Kind != "" {
		t.Fatal("under 7 points is not a takeaway")
	}
	steady := []float64{0.50, 0.50, 0.50, 0.50, 0.50, 0.50, 0.50}
	got := Takeaway(steady, true)
	if got.Kind != "steady" || got.Days != 7 {
		t.Fatalf("%+v", got)
	}
	up := []float64{0.10, 0.10, 0.10, 0.20, 0.40, 0.40, 0.40}
	got = Takeaway(up, true)
	if got.Kind != "improved" || got.Days != 7 {
		t.Fatalf("%+v", got)
	}
	down := []float64{0.40, 0.40, 0.40, 0.20, 0.10, 0.10, 0.10}
	got = Takeaway(down, false)
	if got.Kind != "improved" {
		t.Fatalf("lower-is-better should treat a drop as improved: %+v", got)
	}
}
