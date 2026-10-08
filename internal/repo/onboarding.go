// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type ProjectProgress struct{ DB *gorm.DB }

func (r *ProjectProgress) Get(ctx context.Context, pid uint64) (*model.ProjectProgress, error) {
	row := &model.ProjectProgress{ProjectID: pid}
	err := r.DB.WithContext(ctx).First(row, "project_id = ?", pid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.ProjectProgress{ProjectID: pid}, nil
	}
	return row, err
}
func (r *ProjectProgress) ensure(ctx context.Context, pid uint64) error {
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model.ProjectProgress{ProjectID: pid}).Error
}
func (r *ProjectProgress) Confirm(ctx context.Context, pid, user uint64, kind, revision string) error {
	if kind != "brand" && kind != "questions" {
		return errors.New("invalid confirmation kind")
	}
	if err := r.ensure(ctx, pid); err != nil {
		return err
	}
	now := time.Now().Unix()
	return r.DB.WithContext(ctx).Model(&model.ProjectProgress{}).Where("project_id = ? AND ("+kind+"_revision IS NULL OR "+kind+"_revision <> ?)", pid, revision).Updates(map[string]any{kind + "_revision": revision, kind + "_confirmed_at": now, kind + "_confirmed_by": user}).Error
}
func (r *ProjectProgress) FirstValue(ctx context.Context, pid, user uint64, kind string, ref uint64) error {
	if err := r.ensure(ctx, pid); err != nil {
		return err
	}
	return r.DB.WithContext(ctx).Model(&model.ProjectProgress{}).Where("project_id = ? AND first_value_at IS NULL", pid).Updates(map[string]any{"first_value_at": time.Now().Unix(), "first_value_by": user, "first_value_kind": kind, "first_value_ref": ref}).Error
}
func (r *ProjectProgress) Audit(ctx context.Context, pid, id uint64) (*model.Audit, error) {
	var a model.Audit
	err := r.DB.WithContext(ctx).Where("project_id = ? AND id = ?", pid, id).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

// Task acceptance and its explicit review event succeed or roll back together.
func (r *Tasks) CreateReviewed(ctx context.Context, task *model.Task, user uint64) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		return (&ProjectProgress{DB: tx}).FirstValue(ctx, task.ProjectID, user, "action_accepted", task.ID)
	})
}
