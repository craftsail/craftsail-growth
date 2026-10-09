// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
	"time"
)

// Lock the project row before checking for an active job. This also serializes
// starts by separate API workers on MySQL; SQLite serializes the write.
func (r *Jobs) CreateExclusive(ctx context.Context, j *model.Job) (bool, error) {
	created := false
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lock := tx.Model(&model.Project{}).Where("id = ?", j.ProjectID).UpdateColumn("updated_at", gorm.Expr("updated_at"))
		if lock.Error != nil {
			return lock.Error
		}
		var project model.Project
		if err := tx.Select("id").First(&project, j.ProjectID).Error; err != nil {
			return err
		}
		var n int64
		if err := tx.Model(&model.Job{}).Where("project_id = ? AND status IN ?", j.ProjectID, []string{"running", "queued"}).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		if err := tx.Create(j).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}
func (r *Jobs) Due(ctx context.Context, now int64, onlyID uint64) ([]model.Job, error) {
	var rows []model.Job
	q := r.DB.WithContext(ctx).Where("status = ? AND resumable = ? AND resume_at <= ?", "queued", true, now)
	if onlyID != 0 {
		q = q.Where("id = ?", onlyID)
	}
	err := q.Order("resume_at, id").Limit(20).Find(&rows).Error
	return rows, err
}
func (r *Jobs) Claim(ctx context.Context, id uint64, now int64) (bool, error) {
	result := r.DB.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND status = ? AND resume_at <= ?", id, "queued", now).Updates(map[string]any{"status": "running", "resume_at": nil, "error": ""})
	return result.RowsAffected == 1, result.Error
}
func (r *Jobs) FinishRunning(ctx context.Context, j *model.Job) (bool, error) {
	result := r.DB.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND status = ?", j.ID, "running").Updates(map[string]any{"status": j.Status, "finished_at": j.FinishedAt, "resume_at": j.ResumeAt, "error": j.Error})
	return result.RowsAffected == 1, result.Error
}
func (r *Jobs) StopActive(ctx context.Context, id uint64) (bool, error) {
	result := r.DB.WithContext(ctx).Model(&model.Job{}).Where("id = ? AND status IN ?", id, []string{"running", "queued"}).Updates(map[string]any{"status": "stopped", "resume_at": nil, "finished_at": time.Now().Unix(), "error": "stopped"})
	return result.RowsAffected == 1, result.Error
}
func (r *Jobs) AppendLog(ctx context.Context, id uint64, line string) error {
	expression := "COALESCE(log, '') || ?"
	if r.DB.Dialector.Name() == "mysql" {
		expression = "CONCAT(COALESCE(log, ''), ?)"
	}
	return r.DB.WithContext(ctx).Model(&model.Job{}).Where("id = ?", id).UpdateColumn("log", gorm.Expr(expression, line)).Error
}
