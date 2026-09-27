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
