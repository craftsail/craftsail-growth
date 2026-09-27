// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestUpsertKeepsManualOverride(t *testing.T) {
	db := testDB(t)
	r := &Samples{DB: db}
	ctx := context.Background()
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	base := model.Sample{ProjectID: 1, SampledOn: day, Platform: "openai", QID: "q1", Round: 1, SampleMode: "api", OK: true}
	first := base
	first.Mentioned = true
	first.ManualOverride = true
	if err := r.Upsert(ctx, []model.Sample{first}); err != nil {
		t.Fatal(err)
	}
	again := base
	again.Mentioned = false
	other := base
	other.QID = "q2"
	if err := r.Upsert(ctx, []model.Sample{again, other}); err != nil {
		t.Fatal(err)
	}
	var got model.Sample
	db.Where("qid = ?", "q1").First(&got)
	if !got.Mentioned || !got.ManualOverride {
		t.Fatalf("a re-run must not overwrite a manual override: %+v", got)
	}
	var n int64
	db.Model(&model.Sample{}).Where("qid = ?", "q2").Count(&n)
	if n != 1 {
		t.Fatal("other rows are still written")
	}
}
