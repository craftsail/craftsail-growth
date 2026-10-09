// SPDX-License-Identifier: AGPL-3.0-or-later

package plan

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

type Service struct {
	projects     *project.Service
	audit        *audit.Service
	sample       *sample.Service
	tasks        *repo.Tasks
	observations *repo.Observations
	web          *webstats.Service
	Now          func() time.Time
}

func New(db *gorm.DB) *Service {
	return &Service{
		projects:     project.New(db),
		observations: &repo.Observations{DB: db}, web: webstats.New(db), Now: time.Now,
		audit:  audit.New(db),
		sample: sample.New(db, sample.NewAsker()),
		tasks:  &repo.Tasks{DB: db},
	}
}

type Board struct {
	Tasks   []model.Task   `json:"tasks"`
	Summary map[string]any `json:"summary"`
}

func (s *Service) Get(ctx context.Context, slug, code string) (*model.Task, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	t, err := s.tasks.ByCode(ctx, p.ID, code)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("action %s not found", code)
	}
	return t, nil
}

func (s *Service) SetStatus(ctx context.Context, slug, code, status, note string) (*model.Task, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	t, err := s.tasks.ByCode(ctx, p.ID, code)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, fmt.Errorf("action %s not found", code)
	}
	switch status {
	case model.TaskOpen, model.TaskDoing, model.TaskDone, model.TaskDismissed:
	default:
		return nil, fmt.Errorf("invalid status %s", status)
	}
	t.Status = status
	if note != "" {
		t.Evidence = append(t.Evidence, time.Now().Format(time.RFC3339)+" "+note)
	}
	if status == "done" {
		now := time.Now().Unix()
		t.ClosedAt = &now
	}
	if err := s.tasks.Save(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}
