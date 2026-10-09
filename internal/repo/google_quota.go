// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// ReserveGoogleRequest locks one property's counter before testing either cap.
// dayStart/dayEnd are computed in the provider's quota calendar by the service.
func (r *Webstats) ReserveGoogleRequest(ctx context.Context, property, family string, now, dayStart, dayEnd int64, daily, minute int) (int64, error) {
	var retry int64
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := model.GoogleQuota{KeyHash: model.RowKey(family, property), Property: property, Family: family}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.GoogleQuota{}).Where("key_hash = ?", row.KeyHash).UpdateColumn("daily_used", gorm.Expr("daily_used")).Error; err != nil {
			return err
		}
		if err := tx.Where("key_hash = ?", row.KeyHash).First(&row).Error; err != nil {
			return err
		}
		if row.BlockedUntil > now {
			retry = row.BlockedUntil
			return nil
		}
		if row.Day != dayStart {
			row.Day = dayStart
			row.DailyUsed = 0
		}
		if row.Minute != now/60 {
			row.Minute = now / 60
			row.MinuteUsed = 0
		}
		if row.DailyUsed >= daily {
			retry = dayEnd
			row.Reason = "daily_quota"
		} else if row.MinuteUsed >= minute {
			retry = (now/60 + 1) * 60
			row.Reason = "minute_quota"
		}
		if retry == 0 {
			row.DailyUsed++
			row.MinuteUsed++
			row.Reason = ""
		}
		row.BlockedUntil = retry
		return tx.Save(&row).Error
	})
	return retry, err
}
func (r *Webstats) BackoffGoogle(ctx context.Context, property, family string, until int64) error {
	row := model.GoogleQuota{KeyHash: model.RowKey(family, property), Property: property, Family: family, BlockedUntil: until, Reason: "provider_quota"}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key_hash"}}, DoUpdates: clause.Assignments(map[string]any{"blocked_until": gorm.Expr("CASE WHEN blocked_until > ? THEN blocked_until ELSE ? END", until, until), "reason": "provider_quota"})}).Create(&row).Error
}
func (r *Webstats) GoogleQuota(ctx context.Context, property, family string) (*model.GoogleQuota, error) {
	var out model.GoogleQuota
	err := r.DB.WithContext(ctx).Where("key_hash = ?", model.RowKey(family, property)).Find(&out).Error
	if err != nil {
		return nil, err
	}
	if out.KeyHash == "" {
		return nil, nil
	}
	return &out, nil
}
func (r *Webstats) PruneIndexHistory(ctx context.Context, days int) error {
	if days <= 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Where("checked_at < ?", time.Now().AddDate(0, 0, -days).Unix()).Delete(&model.IndexInspection{}).Error
}
