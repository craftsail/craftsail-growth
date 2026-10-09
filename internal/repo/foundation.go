// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Questions struct{ DB *gorm.DB }

func (r *Questions) List(ctx context.Context, projectID uint64) ([]model.Question, error) {
	var out []model.Question
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("qid").Find(&out).Error
	return out, err
}

func (r *Questions) Replace(ctx context.Context, projectID uint64, rows []model.Question) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.Question{}).Error; err != nil {
			return err
		}
		var p model.Project
		if err := tx.Where("id = ?", projectID).First(&p).Error; err != nil {
			return err
		}
		if _, err := (&Questions{DB: tx}).Snapshot(ctx, &p, rows); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		disabled := []string{}
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].ID = 0
			if !rows[i].Enabled {
				disabled = append(disabled, rows[i].QID)
			}
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		if len(disabled) > 0 {
			if err := tx.Model(&model.Question{}).Where("project_id = ? AND qid IN ?", projectID, disabled).Update("enabled", false).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type Competitors struct{ DB *gorm.DB }

func (r *Competitors) List(ctx context.Context, projectID uint64) ([]model.Competitor, error) {
	var out []model.Competitor
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id").Find(&out).Error
	return out, err
}

func (r *Competitors) Replace(ctx context.Context, projectID uint64, rows []model.Competitor) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.Competitor{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].ID = 0
		}
		return tx.Create(&rows).Error
	})
}

type Facts struct{ DB *gorm.DB }

func (r *Facts) Get(ctx context.Context, projectID uint64) (*model.Fact, error) {
	var row model.Fact
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Facts) Upsert(ctx context.Context, projectID uint64, markdown string) error {
	var row model.Fact
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.DB.WithContext(ctx).Create(&model.Fact{ProjectID: projectID, Markdown: markdown}).Error
	}
	if err != nil {
		return err
	}
	row.Markdown = markdown
	return r.DB.WithContext(ctx).Save(&row).Error
}
