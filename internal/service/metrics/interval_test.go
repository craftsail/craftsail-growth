// SPDX-License-Identifier: AGPL-3.0-or-later

package metrics

import (
	"math"
	"testing"
)

func TestWilsonKnownValues(t *testing.T) {
	cases := []struct {
		x, n   int
		lo, hi float64
	}{
		{5, 20, 0.1119, 0.4687},
		{0, 10, 0.0000, 0.2775},
		{10, 10, 0.7225, 1.0000},
		{81, 263, 0.2553, 0.3662},
		{15, 148, 0.0624, 0.1605},
	}
	for _, c := range cases {
		got := Wilson(c.x, c.n)
		if got == nil {
			t.Fatalf("Wilson(%d,%d) = nil", c.x, c.n)
		}
		if math.Abs(got.Lo-c.lo) > 0.0001 || math.Abs(got.Hi-c.hi) > 0.0001 {
			t.Fatalf("Wilson(%d,%d) = [%.4f, %.4f], want [%.4f, %.4f]", c.x, c.n, got.Lo, got.Hi, c.lo, c.hi)
		}
	}
}

func TestWilsonRejectsEmpty(t *testing.T) {
	if Wilson(0, 0) != nil || Wilson(3, 2) != nil || Wilson(-1, 5) != nil {
		t.Fatal("invalid input must return nil, not a fake interval")
	}
}

func TestNewcombeKnownValues(t *testing.T) {
	cases := []struct {
		x1, n1, x2, n2 int
		lo, hi         float64
	}{
		{56, 70, 48, 80, 0.0524, 0.3339},
		{9, 10, 3, 10, 0.1705, 0.8090},
		{5, 20, 6, 20, -0.3089, 0.2178},
	}
	for _, c := range cases {
		got := NewcombeDiff(c.x1, c.n1, c.x2, c.n2)
		if got == nil || math.Abs(got.Lo-c.lo) > 0.0001 || math.Abs(got.Hi-c.hi) > 0.0001 {
			t.Fatalf("NewcombeDiff(%v) = %+v, want [%.4f, %.4f]", c, got, c.lo, c.hi)
		}
	}
}

func TestChangeUsesInterval(t *testing.T) {
	if got := Change(3, 10, 9, 10); got != ChangeUp {
		t.Fatalf("got %q", got)
	}
	if got := Change(9, 10, 3, 10); got != ChangeDown {
		t.Fatalf("got %q", got)
	}
	if got := Change(5, 20, 6, 20); got != ChangeFlat {
		t.Fatalf("got %q", got)
	}
	if got := Change(0, 0, 6, 20); got != ChangeUnknown {
		t.Fatalf("got %q", got)
	}
}
