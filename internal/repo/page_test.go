// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestSaveRunWritesIssues(t *testing.T) {
	db := testDB(t)
	audits := &Audits{DB: db}
	a := &model.Audit{ProjectID: 1}
	pages := []model.AuditPage{{PageID: 7, URL: "https://e.com/a"}}
	issues := []model.AuditIssue{
		{ProjectID: 1, URL: "https://e.com/a", Code: "NO_JSONLD", Severity: "warning", Layer: "understand"},
		{ProjectID: 1, Code: "NO_SITEMAP", Severity: "warning", Layer: "discover"},
	}
	if err := audits.SaveRunWithIssues(context.Background(), a, pages, issues); err != nil {
		t.Fatal(err)
	}
	var got []model.AuditIssue
	db.Where("audit_id = ?", a.ID).Order("id").Find(&got)
	if len(got) != 2 || got[0].PageID == nil || *got[0].PageID != 7 || got[1].PageID != nil {
		t.Fatalf("issues = %+v", got)
	}
	latest, err := audits.LatestIssues(context.Background(), 1)
	if err != nil || len(latest) != 2 {
		t.Fatalf("latest = %v err=%v", latest, err)
	}
}

func TestNextCodeSkipsTaken(t *testing.T) {
	db := testDB(t)
	r := &Tasks{DB: db}
	ctx := context.Background()
	for _, c := range []string{"A-001", "A-007", "T-003"} {
		if err := r.Create(ctx, &model.Task{ProjectID: 1, Code: c, Status: model.TaskOpen}); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := r.NextCode(ctx, 1, "A"); got != "A-008" {
		t.Fatalf("got %s", got)
	}
	if got, _ := r.NextCode(ctx, 1, "S"); got != "S-001" {
		t.Fatalf("got %s", got)
	}
}
