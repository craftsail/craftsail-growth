// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"context"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type stubSources struct{ items []Item }

func (s stubSources) Collect(ctx context.Context, slug string) ([]Item, error) { return s.items, nil }

type memTasks struct{ rows map[string]model.Task }

func (m *memTasks) BySourceKeys(ctx context.Context, projectID uint64) (map[string]model.Task, error) {
	return m.rows, nil
}
func (m *memTasks) Create(ctx context.Context, t *model.Task) error {
	m.rows[*t.SourceKey] = *t
	return nil
}
func (m *memTasks) NextCode(ctx context.Context, projectID uint64, prefix string) (string, error) {
	return prefix + "-001", nil
}

func pid(context.Context, string) (uint64, error) { return 1, nil }

func TestListMergesTaskStatusAndHidesDismissed(t *testing.T) {
	key, gone := "audit:NO_JSONLD", "search:low_ctr:x"
	tasks := &memTasks{rows: map[string]model.Task{
		key:  {Code: "A-001", Status: model.TaskDoing, SourceKey: &key},
		gone: {Code: "S-001", Status: model.TaskDismissed, SourceKey: &gone},
	}}
	svc := New(stubSources{items: []Item{{Key: key, Priority: "P1"}, {Key: gone, Priority: "P2"}, {Key: "audit:X", Priority: "P0"}}}, tasks, pid)
	got, err := svc.List(context.Background(), "p", ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Key != "audit:X" || got[1].Status != model.TaskDoing || got[1].TaskCode != "A-001" {
		t.Fatalf("%+v", got)
	}
	fresh, _ := svc.List(context.Background(), "p", ListFilter{Status: "new"})
	if len(fresh) != 1 || fresh[0].Key != "audit:X" {
		t.Fatalf("new = %+v", fresh)
	}
}

func TestAcceptCreatesTaskOnceWithBaseline(t *testing.T) {
	tasks := &memTasks{rows: map[string]model.Task{}}
	svc := New(stubSources{items: []Item{{Key: "audit:SPA_SHELL", Source: "audit", Priority: "P0", Title: "t",
		URLs: []string{"https://e.com/a"}, Baseline: map[string]any{"count": 1},
		Acceptance: map[string]any{"type": "auto", "check": "issue.absent:SPA_SHELL"}}}}, tasks, pid)
	task, err := svc.Accept(context.Background(), "p", "audit:SPA_SHELL")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != model.TaskOpen || task.Source != "audit" || task.Code != "A-001" ||
		task.Acceptance["check"] != "issue.absent:SPA_SHELL" || task.Baseline["count"] != 1 {
		t.Fatalf("%+v", task)
	}
	if _, err := svc.Accept(context.Background(), "p", "audit:SPA_SHELL"); err != ErrAccepted {
		t.Fatalf("accepting twice must fail with ErrAccepted, got %v", err)
	}
	if _, err := svc.Accept(context.Background(), "p", "audit:NOPE"); err != ErrNotFound {
		t.Fatalf("unknown key must fail with ErrNotFound, got %v", err)
	}
}

func TestListKeepsAcceptedActionAfterSourceDrops(t *testing.T) {
	key := "audit:NO_JSONLD"
	tasks := &memTasks{rows: map[string]model.Task{
		key: {Code: "A-001", Status: model.TaskVerified, Source: "audit", Title: "No structured data", Priority: "P1", SourceKey: &key},
	}}
	svc := New(stubSources{}, tasks, pid)
	got, err := svc.List(context.Background(), "p", ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != key || got[0].Status != model.TaskVerified || got[0].Kind != "NO_JSONLD" || got[0].Title != "No structured data" {
		t.Fatalf("a verified action must stay listed, got %+v", got)
	}
}
