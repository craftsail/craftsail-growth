// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (r *Webstats) QueueSitemaps(ctx context.Context, projectID uint64, property string, urls []string) error {
	rows := []model.SitemapScan{}
	seen := map[string]bool{}
	for _, u := range urls {
		if seen[u] {
			continue
		}
		seen[u] = true
		rows = append(rows, model.SitemapScan{ProjectID: projectID, Property: property, URL: u, KeyHash: model.RowKey(property, u)})
	}
	if len(rows) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&rows, upsertBatchSize).Error
}
func (r *Webstats) DueSitemaps(ctx context.Context, projectID uint64, property string, now int64, limit int) ([]model.SitemapScan, error) {
	rows := []model.SitemapScan{}
	err := r.DB.WithContext(ctx).Where("project_id = ? AND "+r.indexBinary("property")+" = ? AND next_fetch_at <= ?", projectID, property, now).Order("next_fetch_at, id").Limit(limit).Find(&rows).Error
	return rows, err
}
func (r *Webstats) SaveSitemap(ctx context.Context, scan model.SitemapScan, children []string, urls []model.IndexURL, sources []model.SitemapURL, now int64, failure string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rr := &Webstats{DB: tx}
		next := now + int64((7 * 24 * time.Hour).Seconds())
		if failure != "" {
			next = now + int64((6 * time.Hour).Seconds())
		} else {
			if err := rr.QueueSitemaps(ctx, scan.ProjectID, scan.Property, children); err != nil {
				return err
			}
			if err := rr.DiscoverIndexURLs(ctx, urls); err != nil {
				return err
			}
			if err := tx.Model(&model.SitemapURL{}).Where("project_id = ? AND sitemap_id = ?", scan.ProjectID, scan.ID).Update("present", false).Error; err != nil {
				return err
			}
			if len(sources) > 0 {
				if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "key_hash"}}, DoUpdates: clause.AssignmentColumns([]string{"last_modified", "present", "seen_at"})}).CreateInBatches(&sources, upsertBatchSize).Error; err != nil {
					return err
				}
			}
		}
		updates := map[string]any{"last_error": failure, "next_fetch_at": next}
		if failure == "" {
			updates["fetched_at"] = now
			updates["is_index"] = scan.IsIndex
			updates["discovered"] = len(sources) + len(children)
		}
		return tx.Model(&model.SitemapScan{}).Where("id = ? AND project_id = ?", scan.ID, scan.ProjectID).Updates(updates).Error
	})
}
func (r *Webstats) SitemapStatus(ctx context.Context, pid uint64, property string) ([]model.SitemapScan, error) {
	rows := []model.SitemapScan{}
	err := r.DB.WithContext(ctx).Where("project_id = ? AND "+r.indexBinary("property")+" = ?", pid, property).Order("last_error desc, id").Limit(200).Find(&rows).Error
	return rows, err
}
func (r *Webstats) SetURLPublished(ctx context.Context, pid uint64, property, url string, published *int64) error {
	var row model.IndexURL
	if err := r.DB.WithContext(ctx).Select("id").Where("project_id = ? AND key_hash = ?", pid, model.RowKey(property, url)).First(&row).Error; err != nil {
		return err
	}
	return r.DB.WithContext(ctx).Model(&model.IndexURL{}).Where("id = ? AND project_id = ?", row.ID, pid).Update("published_at", published).Error
}
