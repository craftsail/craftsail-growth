// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func TestSearchSectionUsesOfficialTotals(t *testing.T) {
	var b strings.Builder
	board := &webstats.SearchBoard{
		Period:   webstats.Period{Clicks: 100, Impressions: 1000, Position: 4.2, CTR: 0.1, Measured: true},
		Keywords: []webstats.KeywordRow{{Query: "acme login", Position: 1.2, Clicks: 20, Impressions: 80}},
	}
	writeSearchSection(&b, board)
	s := b.String()
	if !strings.Contains(s, "acme login") || !strings.Contains(s, "100 clicks") || !strings.Contains(s, "anonymized") {
		t.Fatalf("%s", s)
	}
}

func TestSearchSectionSkipsUnmeasured(t *testing.T) {
	var b strings.Builder
	writeSearchSection(&b, &webstats.SearchBoard{})
	if b.Len() != 0 {
		t.Fatalf("unmeasured search data must not render: %q", b.String())
	}
}

func TestSearchSectionObservationAndIncompleteCoverage(t *testing.T) {
	var b strings.Builder
	board := &webstats.SearchBoard{Period: webstats.Period{Measured: true, Clicks: 2, Impressions: 100, Position: 90}, Observation: &webstats.SearchObservation{Mode: "new_site", Coverage: webstats.GrainCoverage{From: "2026-08-24", Through: "2026-09-20", CoveredDays: 4}}}
	writeSearchSection(&b, board)
	out := b.String()
	if !strings.Contains(out, "2026-08-24") || !strings.Contains(out, "comparison is unavailable") || !strings.Contains(out, "Observation mode") || strings.Contains(out, "average position") {
		t.Fatal(out)
	}
}
