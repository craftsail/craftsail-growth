// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// Playbook stores the two things a user tells the playbook: the product
// stage and which manual pass signals they have checked.
type Playbook struct{ DB *gorm.DB }

func (r *Playbook) SetStage(ctx context.Context, pid uint64, stage string) error {
	if err := (&ProjectProgress{DB: r.DB}).ensure(ctx, pid); err != nil {
		return err
	}
	return r.DB.WithContext(ctx).Model(&model.ProjectProgress{}).Where("project_id = ?", pid).Update("product_stage", stage).Error
}

func (r *Playbook) Confirm(ctx context.Context, pid uint64, signal string, user uint64, on bool) error {
	if !on {
		return r.DB.WithContext(ctx).Where("project_id = ? AND `signal` = ?", pid, signal).Delete(&model.PlaybookConfirmation{}).Error
	}
	row := model.PlaybookConfirmation{ProjectID: pid, Signal: signal, ConfirmedAt: time.Now().Unix(), ConfirmedBy: user}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
}

func (r *Playbook) Confirmations(ctx context.Context, pid uint64) (map[string]bool, error) {
	var rows []model.PlaybookConfirmation
	if err := r.DB.WithContext(ctx).Where("project_id = ?", pid).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[row.Signal] = true
	}
	return out, nil
}

// IndexFacts is what the playbook reads from the URL inventory of one
// Search Console property.
type IndexFacts struct {
	NewPages               []model.IndexURL // publish date at or after since
	Indexed                int64            // last verdict PASS
	IndexedWithImpressions int64            // last verdict PASS, with an impression since the same cutoff
	LastSuccess            *int64           // latest successful inspection
}

func (r *Webstats) PlaybookIndex(ctx context.Context, pid uint64, property string, since int64) (*IndexFacts, error) {
	q := func() *gorm.DB {
		return r.DB.WithContext(ctx).Model(&model.IndexURL{}).Where("project_id = ? AND property = ?", pid, property)
	}
	out := &IndexFacts{}
	if err := q().Where("published_at >= ?", since).Find(&out.NewPages).Error; err != nil {
		return nil, err
	}
	if err := q().Where("verdict = ?", "PASS").Count(&out.Indexed).Error; err != nil {
		return nil, err
	}
	if err := q().Where("verdict = ? AND last_impression_at >= ?", "PASS", since).Count(&out.IndexedWithImpressions).Error; err != nil {
		return nil, err
	}
	var last sql.NullInt64
	if err := q().Where("last_success_at > 0").Select("MAX(last_success_at)").Row().Scan(&last); err != nil {
		return nil, err
	}
	if last.Valid {
		out.LastSuccess = &last.Int64
	}
	return out, nil
}
