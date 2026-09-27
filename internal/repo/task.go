// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Tasks struct{ DB *gorm.DB }

func (r *Tasks) List(ctx context.Context, projectID uint64) ([]model.Task, error) {
	var out []model.Task
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("code").Find(&out).Error
	return out, err
}

func (r *Tasks) Replace(ctx context.Context, projectID uint64, rows []model.Task) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.Task{}).Error; err != nil {
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

func (r *Tasks) ByCode(ctx context.Context, projectID uint64, code string) (*model.Task, error) {
	var t model.Task
	err := r.DB.WithContext(ctx).Where("project_id = ? AND code = ?", projectID, code).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Tasks) Save(ctx context.Context, t *model.Task) error {
	return r.DB.WithContext(ctx).Save(t).Error
}

// BySourceKeys returns tasks created from opportunities, keyed by source key.
func (r *Tasks) BySourceKeys(ctx context.Context, projectID uint64) (map[string]model.Task, error) {
	var rows []model.Task
	if err := r.DB.WithContext(ctx).Where("project_id = ? AND source_key IS NOT NULL", projectID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]model.Task, len(rows))
	for _, t := range rows {
		out[*t.SourceKey] = t
	}
	return out, nil
}

func (r *Tasks) Create(ctx context.Context, t *model.Task) error {
	return r.DB.WithContext(ctx).Create(t).Error
}

// NextCode returns the next free code with the prefix, e.g. A-004.
func (r *Tasks) NextCode(ctx context.Context, projectID uint64, prefix string) (string, error) {
	var codes []string
	if err := r.DB.WithContext(ctx).Model(&model.Task{}).Where("project_id = ? AND code LIKE ?", projectID, prefix+"-%").Pluck("code", &codes).Error; err != nil {
		return "", err
	}
	max := 0
	for _, c := range codes {
		var n int
		if _, err := fmt.Sscanf(strings.TrimPrefix(c, prefix+"-"), "%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s-%03d", prefix, max+1), nil
}
