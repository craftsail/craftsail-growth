// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Assets struct{ DB *gorm.DB }

func (r *Assets) Upsert(ctx context.Context, a *model.Asset) error {
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "path"}},
		UpdateAll: true,
	}).Create(a).Error
}

type Verifies struct{ DB *gorm.DB }

func (r *Verifies) Save(ctx context.Context, rep *model.VerifyReport, rows []model.VerifyResult) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rep).Error; err != nil {
			return err
		}
		for i := range rows {
			rows[i].VerifyReportID = rep.ID
		}
		if len(rows) > 0 {
			return tx.Create(&rows).Error
		}
		return nil
	})
}

type Reports struct{ DB *gorm.DB }

func (r *Reports) Upsert(ctx context.Context, row *model.Report) error {
	var old model.Report
	err := r.DB.WithContext(ctx).Where("project_id = ? AND report_on = ?", row.ProjectID, row.ReportOn.Format("2006-01-02")).First(&old).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.DB.WithContext(ctx).Create(row).Error
	}
	if err != nil {
		return err
	}
	old.Markdown, old.HTML = row.Markdown, row.HTML
	return r.DB.WithContext(ctx).Save(&old).Error
}

func (r *Reports) Latest(ctx context.Context, projectID uint64) (*model.Report, error) {
	var row model.Report
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Verifies) Latest(ctx context.Context, projectID uint64) (*model.VerifyReport, []model.VerifyResult, error) {
	var rep model.VerifyReport
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").First(&rep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var rows []model.VerifyResult
	if err := r.DB.WithContext(ctx).Where("verify_report_id = ?", rep.ID).Order("id").Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	return &rep, rows, nil
}
