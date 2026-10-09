// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm/clause"
)

func (r *Tasks) OpportunityScores(ctx context.Context, pid uint64) ([]model.OpportunityScore, error) {
	var rows []model.OpportunityScore
	err := r.DB.WithContext(ctx).Where("project_id = ?", pid).Find(&rows).Error
	return rows, err
}
func (r *Tasks) SaveOpportunityScore(ctx context.Context, row *model.OpportunityScore) error {
	row.KeyHash = model.RowKey(row.Key)
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "key_hash"}}, DoUpdates: clause.AssignmentColumns([]string{"impact", "confidence", "ease", "effort_hours", "updated_at"})}).Create(row).Error
}
