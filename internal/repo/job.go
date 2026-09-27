// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Jobs struct{ DB *gorm.DB }

func (r *Jobs) Create(ctx context.Context, j *model.Job) error {
	return r.DB.WithContext(ctx).Create(j).Error
}

func (r *Jobs) Save(ctx context.Context, j *model.Job) error {
	return r.DB.WithContext(ctx).Save(j).Error
}

func (r *Jobs) ByID(ctx context.Context, id uint64) (*model.Job, error) {
	var j model.Job
	err := r.DB.WithContext(ctx).First(&j, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Jobs) Running(ctx context.Context, projectID uint64) (*model.Job, error) {
	var j model.Job
	err := r.DB.WithContext(ctx).Where("project_id = ? AND status = ?", projectID, "running").Order("id desc").First(&j).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Jobs) Recent(ctx context.Context, projectID uint64, limit int) ([]model.Job, error) {
	if limit <= 0 {
		limit = 12
	}
	q := r.DB.WithContext(ctx).Order("id desc").Limit(limit)
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	var out []model.Job
	err := q.Find(&out).Error
	return out, err
}

func (r *Jobs) InterruptRunning(ctx context.Context) error {
	now := time.Now().Unix()
	return r.DB.WithContext(ctx).Model(&model.Job{}).Where("status = ?", "running").
		Updates(map[string]any{"status": "interrupted", "finished_at": now, "error": "interrupted by a server restart"}).Error
}

func (r *Metrics) Previous(ctx context.Context, projectID uint64) (*model.Metric, error) {
	var rows []model.Metric
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("metric_on desc, id desc").Limit(2).Find(&rows).Error
	if err != nil || len(rows) < 2 {
		return nil, err
	}
	return &rows[1], nil
}

func (r *Audits) Previous(ctx context.Context, projectID uint64) (*model.Audit, error) {
	var rows []model.Audit
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").Limit(2).Find(&rows).Error
	if err != nil || len(rows) < 2 {
		return nil, err
	}
	return &rows[1], nil
}
