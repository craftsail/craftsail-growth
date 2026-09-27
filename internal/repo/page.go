// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Pages struct{ DB *gorm.DB }

func (r *Pages) Replace(ctx context.Context, projectID uint64, pages []model.Page) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.Page{}).Error; err != nil {
			return err
		}
		if len(pages) == 0 {
			return nil
		}
		return tx.Create(&pages).Error
	})
}

func (r *Pages) List(ctx context.Context, projectID uint64) ([]model.Page, error) {
	var out []model.Page
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id").Find(&out).Error
	return out, err
}

type SiteSignals struct{ DB *gorm.DB }

func (r *SiteSignals) Upsert(ctx context.Context, projectID uint64, payload map[string]any) error {
	var row model.SiteSignal
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).First(&row).Error
	now := time.Now().Unix()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.DB.WithContext(ctx).Create(&model.SiteSignal{ProjectID: projectID, Payload: payload, CrawledAt: now}).Error
	}
	if err != nil {
		return err
	}
	row.Payload = payload
	row.CrawledAt = now
	return r.DB.WithContext(ctx).Save(&row).Error
}

func (r *SiteSignals) Get(ctx context.Context, projectID uint64) (*model.SiteSignal, error) {
	var row model.SiteSignal
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

type Audits struct{ DB *gorm.DB }

func (r *Audits) SaveRun(ctx context.Context, a *model.Audit, pages []model.AuditPage) error {
	return r.SaveRunWithIssues(ctx, a, pages, nil)
}

// SaveRunWithIssues stores the audit, its pages and one row per finding in a
// single transaction. Issues are matched to pages by URL.
func (r *Audits) SaveRunWithIssues(ctx context.Context, a *model.Audit, pages []model.AuditPage, issues []model.AuditIssue) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(a).Error; err != nil {
			return err
		}
		byURL := map[string]uint64{}
		for i := range pages {
			pages[i].AuditID = a.ID
			byURL[pages[i].URL] = pages[i].PageID
		}
		if len(pages) > 0 {
			if err := tx.Create(&pages).Error; err != nil {
				return err
			}
		}
		if len(issues) == 0 {
			return nil
		}
		now := time.Now().Unix()
		for i := range issues {
			issues[i].AuditID = a.ID
			issues[i].CreatedAt = now
			if id, ok := byURL[issues[i].URL]; ok && issues[i].URL != "" {
				pid := id
				issues[i].PageID = &pid
			}
		}
		return tx.CreateInBatches(issues, 200).Error
	})
}

// LatestIssues returns the issue rows of the most recent audit.
func (r *Audits) LatestIssues(ctx context.Context, projectID uint64) ([]model.AuditIssue, error) {
	var a model.Audit
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []model.AuditIssue
	err = r.DB.WithContext(ctx).Where("audit_id = ?", a.ID).Order("id").Find(&out).Error
	return out, err
}

func (r *Audits) Latest(ctx context.Context, projectID uint64) (*model.Audit, []model.AuditPage, error) {
	var a model.Audit
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("id desc").First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var pages []model.AuditPage
	if err := r.DB.WithContext(ctx).Where("audit_id = ?", a.ID).Order("score").Find(&pages).Error; err != nil {
		return nil, nil, err
	}
	return &a, pages, nil
}
