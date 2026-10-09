// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
)

func TestRankingProtectsTechnicalEvidenceAndStableTies(t *testing.T) {
	rows := []Item{{Key: "z", Source: "search", Priority: "P1", Evidence: "observational", Score: &model.OpportunityScore{Impact: 10, Confidence: 10, Ease: 10}}, {Key: "fault", Source: "audit", Priority: "P1", Evidence: "standard"}, {Key: "b", Source: "search", Priority: "P2"}, {Key: "a", Source: "search", Priority: "P2"}}
	Sort(rows)
	for i, key := range []string{"fault", "z", "a", "b"} {
		if rows[i].Key != key {
			t.Fatalf("rank %+v", rows)
		}
	}
}
