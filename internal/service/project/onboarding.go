// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

var ErrReviewChanged = errors.New("reviewed content changed; reload before confirming")
var ErrReviewUnavailable = errors.New("save brand details or enabled questions before confirming; helpful feedback requires a saved website audit")

type Progress struct {
	model.ProjectProgress
	FirstValueSeconds        *int64 `json:"first_value_seconds"`
	BrandCurrentRevision     string `json:"brand_current_revision"`
	QuestionsCurrentRevision string `json:"questions_current_revision"`
	BrandConfirmed           bool   `json:"brand_confirmed"`
	QuestionsConfirmed       bool   `json:"questions_confirmed"`
	BrandReady               bool   `json:"brand_ready"`
	QuestionsReady           bool   `json:"questions_ready"`
}

func (s *Service) Progress(ctx context.Context, slug string) (*Progress, error) {
	p, err := s.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	qs, err := (&repo.Questions{DB: s.projects.DB}).List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	row, err := (&repo.ProjectProgress{DB: s.projects.DB}).Get(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	brandReady := !model.IsPendingFact(p.Brand.Definition)
	questionsReady := false
	for _, q := range qs {
		if q.Enabled && strings.TrimSpace(q.Text) != "" {
			questionsReady = true
		}
	}
	var elapsed *int64
	if row.FirstValueAt != nil && p.CreatedAt > 0 && *row.FirstValueAt >= p.CreatedAt {
		seconds := *row.FirstValueAt - p.CreatedAt
		elapsed = &seconds
	}
	return &Progress{
		ProjectProgress: *row, FirstValueSeconds: elapsed,
		BrandCurrentRevision: model.BrandReviewRevision(p), QuestionsCurrentRevision: model.QuestionsReviewRevision(qs),
		BrandReady: brandReady, QuestionsReady: questionsReady,
		BrandConfirmed:     row.BrandConfirmedAt != nil && row.BrandRevision == model.BrandReviewRevision(p),
		QuestionsConfirmed: row.QuestionsConfirmedAt != nil && row.QuestionsRevision == model.QuestionsReviewRevision(qs),
	}, nil
}
func (s *Service) ConfirmReview(ctx context.Context, slug, kind, revision string, user uint64) error {
	if user == 0 || (kind != "brand" && kind != "questions") || len(revision) != 64 {
		return ErrReviewUnavailable
	}
	return s.projects.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		svc := New(tx)
		p, err := svc.Get(ctx, slug)
		if err != nil {
			return err
		}
		status, err := svc.Progress(ctx, slug)
		if err != nil {
			return err
		}
		current := model.BrandReviewRevision(p)
		ready := status.BrandReady
		if kind == "questions" {
			qs, err := (&repo.Questions{DB: tx}).List(ctx, p.ID)
			if err != nil {
				return err
			}
			current = model.QuestionsReviewRevision(qs)
			ready = status.QuestionsReady
		}
		if current != revision {
			return ErrReviewChanged
		}
		if !ready {
			return ErrReviewUnavailable
		}
		return (&repo.ProjectProgress{DB: tx}).Confirm(ctx, p.ID, user, kind, revision)
	})
}
func (s *Service) AuditHelpful(ctx context.Context, slug string, id, user uint64) error {
	if id == 0 || user == 0 {
		return ErrReviewUnavailable
	}
	return s.projects.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p, err := New(tx).Get(ctx, slug)
		if err != nil {
			return err
		}
		rows := &repo.ProjectProgress{DB: tx}
		a, err := rows.Audit(ctx, p.ID, id)
		if err != nil {
			return err
		}
		if a == nil || a.NoSite || a.PageCount < 1 {
			return ErrReviewUnavailable
		}
		return rows.FirstValue(ctx, p.ID, user, "audit_helpful", a.ID)
	})
}
