// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
)

func TestOpportunityScoreIsolationAndReplacement(t *testing.T) {
	db := testDB(t)
	r := &Tasks{DB: db}
	ctx := context.Background()
	for _, row := range []model.OpportunityScore{{ProjectID: 1, Key: "search:x", Impact: 2}, {ProjectID: 2, Key: "search:x", Impact: 9}, {ProjectID: 1, Key: "search:x", Impact: 7}} {
		if err := r.SaveOpportunityScore(ctx, &row); err != nil {
			t.Fatal(err)
		}
	}
	a, err := r.OpportunityScores(ctx, 1)
	if err != nil || len(a) != 1 || a[0].Impact != 7 {
		t.Fatalf("%+v %v", a, err)
	}
	b, err := r.OpportunityScores(ctx, 2)
	if err != nil || len(b) != 1 || b[0].Impact != 9 {
		t.Fatalf("%+v %v", b, err)
	}
}
