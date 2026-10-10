// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestPlaybookStageAndConfirmations(t *testing.T) {
	ctx := context.Background()
	r := &Playbook{DB: testDB(t)}
	if err := r.SetStage(ctx, 7, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := r.SetStage(ctx, 7, "s2"); err != nil {
		t.Fatal(err)
	}
	row, err := (&ProjectProgress{DB: r.DB}).Get(ctx, 7)
	if err != nil || row.ProductStage != "s2" {
		t.Fatalf("stage = %q, %v", row.ProductStage, err)
	}
	for _, on := range []bool{true, true} { // confirming twice is an upsert
		if err := r.Confirm(ctx, 7, "crawlUp", 3, on); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Confirm(ctx, 7, "brandFilter", 3, true); err != nil {
		t.Fatal(err)
	}
	if err := r.Confirm(ctx, 7, "brandFilter", 3, false); err != nil {
		t.Fatal(err)
	}
	got, err := r.Confirmations(ctx, 7)
	if err != nil || len(got) != 1 || !got["crawlUp"] {
		t.Fatalf("confirmations = %v, %v", got, err)
	}
}

func TestPlaybookIndexFacts(t *testing.T) {
	ctx := context.Background()
	r := &Webstats{DB: testDB(t)}
	ts := func(v int64) *int64 { return &v }
	rows := []model.IndexURL{
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "a", URL: "https://x.test/a", Verdict: "PASS", PublishedAt: ts(1000), FirstIndexedAt: ts(2000), LastImpressionAt: ts(1500), LastSuccessAt: 5000},
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "b", URL: "https://x.test/b", Verdict: "NEUTRAL", PublishedAt: ts(50), LastSuccessAt: 6000},
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "c", URL: "https://x.test/c", Verdict: "PASS"},
		{ProjectID: 1, Property: "sc-domain:x.test", KeyHash: "e", URL: "https://x.test/e", Verdict: "PASS", LastImpressionAt: ts(100)},
		{ProjectID: 1, Property: "other", KeyHash: "d", URL: "https://x.test/d", Verdict: "PASS", LastSuccessAt: 9000},
	}
	if err := r.DB.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	f, err := r.PlaybookIndex(ctx, 1, "sc-domain:x.test", 500)
	if err != nil {
		t.Fatal(err)
	}
	// "e" is PASS and indexed, but its only impression (100) predates the
	// since cutoff (500), so it does not count toward IndexedWithImpressions.
	if len(f.NewPages) != 1 || f.NewPages[0].KeyHash != "a" || f.Indexed != 3 || f.IndexedWithImpressions != 1 ||
		f.LastSuccess == nil || *f.LastSuccess != 6000 {
		t.Fatalf("facts = %+v", f)
	}
}

// A property whose only PASS rows never had a successful inspection (the
// zero value, not a timestamp) must report no last-success time, not epoch 0.
func TestPlaybookIndexFactsNoSuccess(t *testing.T) {
	ctx := context.Background()
	r := &Webstats{DB: testDB(t)}
	rows := []model.IndexURL{
		{ProjectID: 2, Property: "sc-domain:y.test", KeyHash: "a", URL: "https://y.test/a", Verdict: "PASS"},
		{ProjectID: 2, Property: "sc-domain:y.test", KeyHash: "b", URL: "https://y.test/b", Verdict: "PASS"},
	}
	if err := r.DB.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	f, err := r.PlaybookIndex(ctx, 2, "sc-domain:y.test", 500)
	if err != nil {
		t.Fatal(err)
	}
	if f.LastSuccess != nil {
		t.Fatalf("last success = %v, want nil", f.LastSuccess)
	}
}
