// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestReviewedAcceptanceAtomicFirstValue(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	r := &Tasks{DB: db}
	task := model.Task{ProjectID: 1, Code: "A-001", Status: model.TaskOpen}
	if err := r.CreateReviewed(ctx, &task, 7); err != nil {
		t.Fatal(err)
	}
	progress, err := (&ProjectProgress{DB: db}).Get(ctx, 1)
	if err != nil || progress.FirstValueKind != "action_accepted" || progress.FirstValueRef != task.ID || progress.FirstValueBy != 7 {
		t.Fatalf("%#v %v", progress, err)
	}
	next := model.Task{ProjectID: 1, Code: "A-002", Status: model.TaskOpen}
	if err := r.CreateReviewed(ctx, &next, 8); err != nil {
		t.Fatal(err)
	}
	after, _ := (&ProjectProgress{DB: db}).Get(ctx, 1)
	if after.FirstValueRef != task.ID || after.FirstValueBy != 7 {
		t.Fatal("first event replaced")
	}
	// Storage failure after task insertion must roll the task back too.
	if err := db.Migrator().DropTable(&model.ProjectProgress{}); err != nil {
		t.Fatal(err)
	}
	bad := model.Task{ProjectID: 2, Code: "A-001", Status: model.TaskOpen}
	if err := r.CreateReviewed(ctx, &bad, 7); err == nil {
		t.Fatal("expected failure")
	}
	var n int64
	db.Model(&model.Task{}).Where("project_id = ?", 2).Count(&n)
	if n != 0 {
		t.Fatal("task survived failed value transaction")
	}
}
