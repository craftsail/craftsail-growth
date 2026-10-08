// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
)

type Observations struct{ DB *gorm.DB }

func (r *Observations) List(ctx context.Context, pid uint64, code string) ([]model.Observation, error) {
	var rows []model.Observation
	q := r.DB.WithContext(ctx).Where("project_id = ?", pid)
	if code != "" {
		q = q.Where("task_code = ?", code)
	}
	if err := q.Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return rows, nil
	}
	ids := make([]uint64, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var results []model.ObservationResult
	if err := r.DB.WithContext(ctx).Where("project_id = ? AND observation_id IN ?", pid, ids).Order("id DESC").Find(&results).Error; err != nil {
		return nil, err
	}
	by := map[uint64][]model.ObservationResult{}
	for _, v := range results {
		by[v.ObservationID] = append(by[v.ObservationID], v)
	}
	for i := range rows {
		rows[i].Results = by[rows[i].ID]
	}
	return rows, nil
}
func (r *Observations) Create(ctx context.Context, o *model.Observation) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(o).Error; err != nil {
			return err
		}
		return tx.Model(&model.Task{}).Where("id = ? AND project_id = ?", o.TaskID, o.ProjectID).Update("released_at", o.ReleasedAt).Error
	})
}
func (r *Observations) Append(ctx context.Context, result *model.ObservationResult) error {
	return r.DB.WithContext(ctx).Create(result).Error
}
func (r *Observations) Samples(ctx context.Context, pid uint64, engine, access, from, through string, qids []string) ([]model.Sample, error) {
	var rows []model.Sample
	day := "date(sampled_on)"
	if r.DB.Dialector.Name() == "sqlite" {
		day = "substr(sampled_on,1,10)"
	}
	modes := []string{access}
	if access == "web" {
		modes = append(modes, "manual")
	}
	err := r.DB.WithContext(ctx).Where("project_id = ? AND platform = ? AND sample_mode IN ? AND qid IN ? AND "+day+" >= ? AND "+day+" <= ?", pid, engine, modes, qids, from, through).Find(&rows).Error
	return rows, err
}
func (r *Observations) Audit(ctx context.Context, pid uint64, urls []string, code string) (model.Measurement, error) {
	var a model.Audit
	err := r.DB.WithContext(ctx).Where("project_id = ?", pid).Order("run_at DESC, id DESC").First(&a).Error
	if err == gorm.ErrRecordNotFound {
		return model.Measurement{Reason: "no_audit"}, nil
	}
	if err != nil {
		return model.Measurement{}, err
	}
	var pages []model.AuditPage
	if err := r.DB.WithContext(ctx).Where("audit_id = ?", a.ID).Find(&pages).Error; err != nil {
		return model.Measurement{}, err
	}
	latest := map[string]model.AuditPage{}
	for _, p := range pages {
		latest[p.URL] = p
	}
	out := model.Measurement{Valid: true, AuditAt: a.RunAt, Signature: code}
	for _, u := range urls {
		p, ok := latest[u]
		if !ok {
			return model.Measurement{Reason: "missing_page", AuditAt: a.RunAt}, nil
		}

		if p.CrawledAt == 0 {
			return model.Measurement{Reason: "fresh_audit"}, nil
		}
		if p.CrawledAt < out.AuditAt {
			out.AuditAt = p.CrawledAt
		}
		out.Count++
		for _, c := range p.IssueCodes {
			if c == code {
				out.Value++
				break
			}
		}
	}
	if out.Count == 0 {
		out.Valid = false
		out.Reason = "missing_page"
	}
	return out, nil
}
