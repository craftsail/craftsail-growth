// SPDX-License-Identifier: AGPL-3.0-or-later

package opportunity

import (
	"context"
	"errors"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// Sources produces the current opportunity items for a project.
type Sources interface {
	Collect(ctx context.Context, slug string) ([]Item, error)
}

type TaskStore interface {
	BySourceKeys(ctx context.Context, projectID uint64) (map[string]model.Task, error)
	Create(ctx context.Context, t *model.Task) error
	NextCode(ctx context.Context, projectID uint64, prefix string) (string, error)
}

type Service struct {
	sources   Sources
	tasks     TaskStore
	projectID func(ctx context.Context, slug string) (uint64, error)
}

func New(sources Sources, tasks TaskStore, projectID func(context.Context, string) (uint64, error)) *Service {
	return &Service{sources: sources, tasks: tasks, projectID: projectID}
}

type ListFilter struct {
	Source string
	Status string // "" = everything not dismissed; "new" = not accepted yet
}

var (
	ErrAccepted = errors.New("opportunity already accepted or dismissed")
	ErrNotFound = errors.New("opportunity not found")
)

// List returns current items merged with the status of tasks created from
// them. Dismissed items are hidden unless asked for.
func (s *Service) List(ctx context.Context, slug string, f ListFilter) ([]Item, error) {
	pid, err := s.projectID(ctx, slug)
	if err != nil {
		return nil, err
	}
	items, err := s.sources.Collect(ctx, slug)
	if err != nil {
		return nil, err
	}
	tasks, err := s.tasks.BySourceKeys(ctx, pid)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	present := map[string]bool{}
	for _, it := range items {
		present[it.Key] = true
	}
	// An accepted action stays listed after its source stops producing the
	// item, for example once the audit issue is fixed and verified.
	for key, t := range tasks {
		if present[key] || t.Status == model.TaskDismissed {
			continue
		}
		items = append(items, itemFromTask(t))
	}
	for _, it := range items {
		if t, ok := tasks[it.Key]; ok {
			it.Status, it.TaskCode = t.Status, t.Code
		}
		if it.Status == model.TaskDismissed && f.Status != model.TaskDismissed {
			continue
		}
		if f.Source != "" && it.Source != f.Source {
			continue
		}
		if f.Status == "new" && it.Status != "" {
			continue
		}
		if f.Status != "" && f.Status != "new" && it.Status != f.Status {
			continue
		}
		out = append(out, it)
	}
	Sort(out)
	return out, nil
}

func (s *Service) Accept(ctx context.Context, slug, key string) (*model.Task, error) {
	return s.materialize(ctx, slug, key, model.TaskOpen)
}

func (s *Service) Dismiss(ctx context.Context, slug, key string) (*model.Task, error) {
	return s.materialize(ctx, slug, key, model.TaskDismissed)
}

func (s *Service) materialize(ctx context.Context, slug, key, status string) (*model.Task, error) {
	pid, err := s.projectID(ctx, slug)
	if err != nil {
		return nil, err
	}
	existing, err := s.tasks.BySourceKeys(ctx, pid)
	if err != nil {
		return nil, err
	}
	if _, ok := existing[key]; ok {
		return nil, ErrAccepted
	}
	items, err := s.sources.Collect(ctx, slug)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if it.Key != key {
			continue
		}
		code, err := s.tasks.NextCode(ctx, pid, strings.ToUpper(it.Source[:1]))
		if err != nil {
			return nil, err
		}
		k := it.Key
		t := &model.Task{
			ProjectID: pid, Code: code, Priority: it.Priority, Title: it.Title, Why: it.Why, Action: it.Fix,
			Status: status, Source: it.Source, SourceKey: &k, Acceptance: it.Acceptance, Affected: it.URLs,
			Baseline: it.Baseline, BaselineCount: len(it.URLs),
		}
		if err := s.tasks.Create(ctx, t); err != nil {
			return nil, err
		}
		return t, nil
	}
	return nil, ErrNotFound
}

func itemFromTask(t model.Task) Item {
	key := ""
	if t.SourceKey != nil {
		key = *t.SourceKey
	}
	kind := key
	if parts := strings.SplitN(key, ":", 3); len(parts) >= 2 {
		kind = parts[1]
	}
	return Item{
		Key: key, Source: t.Source, Kind: kind, Priority: t.Priority, Title: t.Title, Why: t.Why, Fix: t.Action,
		URLs: t.Affected, Acceptance: t.Acceptance, Status: t.Status, TaskCode: t.Code,
	}
}
