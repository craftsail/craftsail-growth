// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm/clause"
)

func (r *Questions) Snapshot(ctx context.Context, p *model.Project, rows []model.Question) (*model.QuestionLibrary, error) {
	copyRows := append([]model.Question{}, rows...)
	for i := range copyRows {
		copyRows[i].ID = 0
		copyRows[i].ProjectID = 0
		copyRows[i].CreatedAt = 0
		copyRows[i].UpdatedAt = 0
		copyRows[i].SystemTags = nil
	}
	row := model.QuestionLibrary{ProjectID: p.ID, Revision: model.LibraryRevision(copyRows, p.SamplingLanguage, p.TargetRegion), Language: p.SamplingLanguage, Region: p.TargetRegion, Questions: copyRows}
	err := r.DB.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "project_id"}, {Name: "revision"}}, DoNothing: true}).Create(&row).Error
	return &row, err
}
func (r *Questions) Libraries(ctx context.Context, pid uint64) ([]model.QuestionLibrary, error) {
	var rows []model.QuestionLibrary
	err := r.DB.WithContext(ctx).Where("project_id = ?", pid).Order("id DESC").Limit(100).Find(&rows).Error
	return rows, err
}
