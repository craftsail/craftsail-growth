// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"context"
	"errors"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

func TestReviewProgressVersionsAndExplicitEvents(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	p, err := s.Create(ctx, CreateInput{Name: "Review", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Progress(ctx, p.Slug)
	if err != nil || status.FirstValueAt != nil || status.BrandConfirmed || status.QuestionsConfirmed {
		t.Fatalf("%#v %v", status, err)
	}
	var n int64
	db.Model(&model.ProjectProgress{}).Count(&n)
	if n != 0 {
		t.Fatal("GET wrote progress")
	}
	if err := s.ConfirmReview(ctx, p.Slug, "brand", model.BrandReviewRevision(p), 1); !errors.Is(err, ErrReviewUnavailable) {
		t.Fatalf("empty brand %v", err)
	}
	p.Brand.Definition = "A tool for teams"
	s.Save(ctx, p)
	revision := model.BrandReviewRevision(p)
	if err := s.ConfirmReview(ctx, p.Slug, "brand", revision, 1); err != nil {
		t.Fatal(err)
	}
	status, _ = s.Progress(ctx, p.Slug)
	if !status.BrandConfirmed || status.FirstValueAt != nil {
		t.Fatalf("%#v", status)
	}
	if err := s.ConfirmReview(ctx, p.Slug, "brand", revision, 2); err != nil {
		t.Fatal(err)
	}
	status, _ = s.Progress(ctx, p.Slug)
	if status.BrandConfirmedBy != 1 {
		t.Fatal("duplicate confirmation replaced actor")
	}
	p.Brand.Definition = "Changed description"
	s.Save(ctx, p)
	status, _ = s.Progress(ctx, p.Slug)
	if status.BrandConfirmed || status.BrandConfirmedAt == nil {
		t.Fatal("changed content kept current confirmation")
	}
	if err := s.ConfirmReview(ctx, p.Slug, "brand", revision, 1); !errors.Is(err, ErrReviewChanged) {
		t.Fatalf("stale %v", err)
	}
	qs := []model.Question{{QID: "q1", Text: "Which tool?", Enabled: true, Market: model.MarketAll}}
	if err := (&repo.Questions{DB: db}).Replace(ctx, p.ID, qs); err != nil {
		t.Fatal(err)
	}
	rows, _ := (&repo.Questions{DB: db}).List(ctx, p.ID)
	if err := s.ConfirmReview(ctx, p.Slug, "questions", model.QuestionsReviewRevision(rows), 1); err != nil {
		t.Fatal(err)
	}
	rows[0].ID = 500
	rows[0].UpdatedAt = 1234
	rows[0].SystemTags = []string{"unbranded"}
	if err := s.ConfirmReview(ctx, p.Slug, "questions", model.QuestionsReviewRevision(rows), 1); err != nil {
		t.Fatalf("derived fields affected revision %v", err)
	}
	db.Model(&model.Question{}).Where("project_id = ?", p.ID).Update("text", "A different question?")
	status, _ = s.Progress(ctx, p.Slug)
	if status.QuestionsConfirmed || status.FirstValueAt != nil {
		t.Fatal("edit or confirmation counted as value")
	}
	a := model.Audit{ProjectID: p.ID, PageCount: 1}
	db.Create(&a)
	if err := s.AuditHelpful(ctx, p.Slug, a.ID, 1); err != nil {
		t.Fatal(err)
	}
	first, _ := s.Progress(ctx, p.Slug)
	if first.FirstValueAt == nil || first.FirstValueKind != "audit_helpful" || first.FirstValueRef != a.ID {
		t.Fatalf("%#v", first)
	}
	if err := s.AuditHelpful(ctx, p.Slug, a.ID, 2); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Progress(ctx, p.Slug)
	if after.FirstValueBy != 1 || *after.FirstValueAt != *first.FirstValueAt {
		t.Fatal("first value was overwritten")
	}
	// Whole-project saves cannot erase the independent progress record.
	s.Save(ctx, p)
	after, _ = s.Progress(ctx, p.Slug)
	if after.FirstValueAt == nil {
		t.Fatal("project save erased progress")
	}
	other, _ := s.Create(ctx, CreateInput{Name: "Other", NoSite: true})
	if err := s.AuditHelpful(ctx, other.Slug, a.ID, 1); !errors.Is(err, ErrReviewUnavailable) {
		t.Fatalf("cross project evidence %v", err)
	}
	empty := model.Audit{ProjectID: p.ID, NoSite: true}
	db.Create(&empty)
	if err := s.AuditHelpful(ctx, p.Slug, empty.ID, 1); !errors.Is(err, ErrReviewUnavailable) {
		t.Fatalf("empty audit %v", err)
	}
	if err := s.AuditHelpful(ctx, p.Slug, a.ID, 0); !errors.Is(err, ErrReviewUnavailable) {
		t.Fatalf("automated event %v", err)
	}
}
