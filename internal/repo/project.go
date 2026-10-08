// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Projects struct {
	DB *gorm.DB
}

func (r *Projects) List(ctx context.Context) ([]model.Project, error) {
	var out []model.Project
	err := r.DB.WithContext(ctx).Order("id desc").Find(&out).Error
	return out, err
}

func (r *Projects) BySlug(ctx context.Context, slug string) (*model.Project, error) {
	var p model.Project
	err := r.DB.WithContext(ctx).Where("slug = ?", slug).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Projects) ByID(ctx context.Context, id uint64) (*model.Project, error) {
	var p model.Project
	err := r.DB.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Projects) Create(ctx context.Context, p *model.Project) error {
	return r.DB.WithContext(ctx).Create(p).Error
}

// DeleteBySlug removes the project and its memberships, so a forced
// re-create does not leave access rows pointing at a reused project ID.
func (r *Projects) DeleteBySlug(ctx context.Context, slug string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p model.Project
		err := tx.Where("slug = ?", slug).First(&p).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", p.ID).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", p.ID).Delete(&model.ProjectProgress{}).Error; err != nil {
			return err
		}
		for _, table := range []any{&model.Observation{}, &model.ObservationResult{}, &model.OpportunityScore{}, &model.IndexURL{}, &model.IndexInspection{}, &model.SitemapScan{}, &model.SitemapURL{}, &model.GscIndex{}, &model.GscSitemap{}} {
			if err := tx.Where("project_id = ?", p.ID).Delete(table).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&model.Project{}, p.ID).Error
	})
}

func (r *Projects) Save(ctx context.Context, p *model.Project) error {
	return r.DB.WithContext(ctx).Save(p).Error
}

func (r *Projects) SetReportLanguage(ctx context.Context, pid uint64, language string) error {
	return r.DB.WithContext(ctx).Model(&model.Project{}).Where("id = ?", pid).Update("report_language", language).Error
}
