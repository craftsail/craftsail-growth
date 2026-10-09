// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
	"time"
)

func TestSampleLegacyIndexMigrationKeepsVersionsAndOverrides(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	store := &Samples{DB: db}
	if err := db.Migrator().DropIndex(&model.Sample{}, "uk_sample_v2"); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX uk_sample ON samples(project_id,sampled_on,platform,qid,round,sample_mode)").Error; err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	old := model.Sample{ProjectID: 1, SampledOn: day, Platform: "x", QID: "q", Round: 1, SampleMode: "api", Answer: "legacy", ManualOverride: true}
	if err := db.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := model.AutoMigrate(db); err != nil {
			t.Fatal(err)
		}
	}
	if db.Migrator().HasIndex(&model.Sample{}, "uk_sample") {
		t.Fatal("old restrictive index remains")
	}
	newer := old
	newer.ID = 0
	newer.PromptRevision = "new"
	newer.SamplingLanguage = "pt"
	newer.Answer = "new answer"
	newer.ManualOverride = false
	if err := store.Upsert(ctx, []model.Sample{newer}); err != nil {
		t.Fatal(err)
	}
	old.Answer = "overwrite"
	old.ManualOverride = false
	if err := store.Upsert(ctx, []model.Sample{old}); err != nil {
		t.Fatal(err)
	}
	var rows []model.Sample
	db.Order("id").Find(&rows)
	if len(rows) != 2 || rows[0].Answer != "legacy" || rows[0].PromptRevision != "" || rows[0].SamplingLanguage != "" || rows[1].Answer != "new answer" {
		t.Fatalf("lost version/override %+v", rows)
	}
}
