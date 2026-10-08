// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"time"
)

func (r *Samples) ScopeFacets(ctx context.Context, pid uint64, since, until time.Time) ([]model.Sample, error) {
	q := r.DB.WithContext(ctx).Model(&model.Sample{}).Where("project_id = ?", pid)
	if !since.IsZero() {
		q = q.Where("sampled_on >= ?", since.Format("2006-01-02"))
	}
	if !until.IsZero() {
		q = q.Where("sampled_on < ?", until.Format("2006-01-02"))
	}
	var rows []model.Sample
	err := q.Distinct("sampling_language", "target_region", "prompt_revision").Find(&rows).Error
	return rows, err
}
