// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"strings"

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
			rows[i].KeyHash = model.RowKey(rows[i].Path)
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
			rows[i].KeyHash = model.RowKey(rows[i].URL)
		}
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "key_hash"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) ListSitemaps(ctx context.Context, projectID uint64) ([]model.GscSitemap, error) {
	var out []model.GscSitemap
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("submitted desc").Find(&out).Error
	if out == nil {
		out = []model.GscSitemap{}
	}
	return out, err
}

func (r *Webstats) ListIndex(ctx context.Context, projectID uint64) ([]model.GscIndex, error) {
	var out []model.GscIndex
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("url").Find(&out).Error
	if out == nil {
		out = []model.GscIndex{}
	}
	return out, err
}

func (r *Webstats) IndexTargets(ctx context.Context, projectID uint64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 30
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		u := strings.TrimSpace(raw)
		if u == "" || !strings.Contains(u, "://") {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		if len(out) >= limit {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	var pages []string
	if err := r.DB.WithContext(ctx).Model(&model.Page{}).Where("project_id = ?", projectID).Pluck("url", &pages).Error; err != nil {
		return nil, err
	}
	for _, u := range pages {
		add(u)
	}
	type pageHit struct {
		Page        string
		Impressions float64
	}
	var hits []pageHit
	err := r.DB.WithContext(ctx).Model(&model.GscFact{}).
		Select("page, SUM(impressions) AS impressions").
		Where("project_id = ? AND slice = ? AND page <> ''", projectID, "query_page").
		Group("page").
		Order("impressions desc").
		Limit(limit).
		Scan(&hits).Error
	if err != nil {
		return nil, err
	}
	for _, h := range hits {
		add(h.Page)
	}
	return out, nil
}
