// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// upsertBatchSize stays under MySQL's 65535-placeholder limit for a GSC day.
const upsertBatchSize = 200

type Webstats struct{ DB *gorm.DB }

func (r *Webstats) UpsertSitemaps(ctx context.Context, rows []model.GscSitemap) error {
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		if rows[i].KeyHash == "" {
			rows[i].KeyHash = model.RowKey(rows[i].Property, rows[i].Path)
		}
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "key_hash"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) UpsertIndex(ctx context.Context, rows []model.GscIndex) error {
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		if rows[i].KeyHash == "" {
			rows[i].KeyHash = model.RowKey(rows[i].Property, rows[i].URL)
		}
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "key_hash"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) ListSitemaps(ctx context.Context, projectID uint64, property string) ([]model.GscSitemap, error) {
	var out []model.GscSitemap
	err := r.DB.WithContext(ctx).Where("project_id = ? AND "+r.indexBinary("property")+" = ?", projectID, property).Order("submitted desc").Find(&out).Error
	if out == nil {
		out = []model.GscSitemap{}
	}
	return out, err
}

func (r *Webstats) ListIndex(ctx context.Context, projectID uint64, property string) ([]model.GscIndex, error) {
	var out []model.GscIndex
	err := r.DB.WithContext(ctx).Where("project_id = ? AND "+r.indexBinary("property")+" = ?", projectID, property).Order("url").Find(&out).Error
	if out == nil {
		out = []model.GscIndex{}
	}
	return out, err
}
