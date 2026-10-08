// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
	"time"
)

func (r *Webstats) SearchProperty(ctx context.Context, projectID uint64, property string) (*model.WebProperty, error) {
	var row model.WebProperty
	err := r.DB.WithContext(ctx).Where("project_id = ? AND source = ? AND property_key = ?", projectID, "gsc", property).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}
func (r *Webstats) PromoteSearchStage(ctx context.Context, projectID uint64, property string, through time.Time, version int) error {
	return r.DB.WithContext(ctx).Model(&model.WebProperty{}).Where("project_id = ? AND source = ? AND property_key = ? AND (search_established_through IS NULL OR search_stage_version <> ?)", projectID, "gsc", property, version).Updates(map[string]any{"search_established_through": through, "search_stage_version": version}).Error
}
func (r *Webstats) PagesWithImpressions(ctx context.Context, projectID uint64, property string, from, through time.Time) (int64, error) {
	group := "page COLLATE BINARY"
	if r.DB.Dialector.Name() == "mysql" {
		group = "CAST(page AS BINARY)"
	}
	q := r.DB.WithContext(ctx).Model(&model.GscFact{}).Select(group).Where("project_id = ? AND property = ? AND slice = ? AND search_type = ? AND page <> '' AND date(day) >= ? AND date(day) <= ?", projectID, property, "page", "web", from.Format("2006-01-02"), through.Format("2006-01-02")).Group(group).Having("SUM(impressions) > 0")
	var count int64
	err := r.DB.WithContext(ctx).Table("(?) AS visible_pages", q).Count(&count).Error
	return count, err
}
