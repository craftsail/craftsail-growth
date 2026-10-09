// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Samples struct{ DB *gorm.DB }

// Upsert writes samples keyed by day, engine, prompt, round and mode. Rows a
// person corrected (manual_override) are never overwritten by a re-run.
func (r *Samples) Upsert(ctx context.Context, rows []model.Sample) error {
	if len(rows) == 0 {
		return nil
	}
	rows, err := r.withoutOverridden(ctx, rows)
	if err != nil || len(rows) == 0 {
		return err
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "project_id"}, {Name: "sampled_on"}, {Name: "platform"},
			{Name: "qid"}, {Name: "round"}, {Name: "sample_mode"}, {Name: "prompt_revision"},
		},
		UpdateAll: true,
	}).Create(&rows).Error
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

type SampleDayAgg struct {
	Day       string
	Platform  string
	OK        int
	Mentioned int
	Failed    int
}

func (r *Samples) AggregateDays(ctx context.Context, projectID uint64) ([]SampleDayAgg, error) {
	// SQLite stores the date as local midnight with an offset, and its date()
	// converts to UTC first, which moves east-of-UTC days back by one. The
	// first ten characters are the local day.
	day := "date(sampled_on)"
	if r.DB.Dialector.Name() == "sqlite" {
		day = "substr(sampled_on, 1, 10)"
	}
	var rows []SampleDayAgg
	err := r.DB.WithContext(ctx).Raw(`
		SELECT `+day+` AS day, platform,
			SUM(CASE WHEN ok THEN 1 ELSE 0 END) AS ok,
			SUM(CASE WHEN ok AND mentioned THEN 1 ELSE 0 END) AS mentioned,
			SUM(CASE WHEN NOT ok THEN 1 ELSE 0 END) AS failed
		FROM samples WHERE project_id = ? GROUP BY `+day+`, platform`, projectID).Scan(&rows).Error
	return rows, err
}

func (r *Samples) CountRounds(ctx context.Context, projectID uint64, day time.Time, platform, qid, mode string, revisions ...string) (int, error) {
	var n int64
	q := r.DB.WithContext(ctx).Model(&model.Sample{})
	if len(revisions) > 0 {
		q = q.Where("prompt_revision = ?", revisions[0])
	}
	err := q.
		Where("project_id = ? AND sampled_on >= ? AND sampled_on < ? AND platform = ? AND qid = ? AND sample_mode = ? AND ok = ?",
			projectID, dateOnly(day), dateOnly(day).Add(24*time.Hour), platform, qid, mode, true).
		Count(&n).Error
	return int(n), err
}

func (r *Samples) ReplaceCitations(ctx context.Context, projectID uint64, sampleIDs []uint64, rows []model.SampleCitation) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(sampleIDs) > 0 {
			if err := tx.Where("project_id = ? AND sample_id IN ?", projectID, sampleIDs).Delete(&model.SampleCitation{}).Error; err != nil {
				return err
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func (r *Samples) ListCitations(ctx context.Context, projectID uint64, since time.Time) ([]model.SampleCitation, error) {
	q := r.DB.WithContext(ctx).Where("project_id = ?", projectID)
	if !since.IsZero() {
		q = q.Where("sampled_on >= ?", since.Format("2006-01-02"))
	}
	var out []model.SampleCitation
	err := q.Order("sampled_on, citation_index").Find(&out).Error
	return out, err
}

func (r *Samples) List(ctx context.Context, projectID uint64, day time.Time, platform, qid string, limit int) ([]model.Sample, error) {
	q := r.DB.WithContext(ctx).Where("project_id = ?", projectID)
	if !day.IsZero() {
		q = q.Where("sampled_on >= ? AND sampled_on < ?", dateOnly(day), dateOnly(day).Add(24*time.Hour))
	}
	if platform != "" {
		q = q.Where("platform = ?", platform)
	}
	if qid != "" {
		q = q.Where("qid = ?", qid)
	}
	if limit <= 0 {
		limit = 300
	}
	var out []model.Sample
	err := q.Order("sampled_on desc, platform, qid").Limit(limit).Find(&out).Error
	return out, err
}

type Metrics struct{ DB *gorm.DB }

func (r *Metrics) Upsert(ctx context.Context, m *model.Metric) error {
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "metric_on"}, {Name: "market"}},
		UpdateAll: true,
	}).Create(m).Error
}

func (r *Metrics) Latest(ctx context.Context, projectID uint64) (*model.Metric, error) {
	var m model.Metric
	err := r.DB.WithContext(ctx).Where("project_id = ?", projectID).Order("metric_on desc").First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func sampleKey(s model.Sample) string {
	return fmt.Sprintf("%d|%s|%s|%s|%d|%s|%s", s.ProjectID, s.SampledOn.Format("2006-01-02"), s.Platform, s.QID, s.Round, s.SampleMode, s.PromptRevision)
}

func (r *Samples) withoutOverridden(ctx context.Context, rows []model.Sample) ([]model.Sample, error) {
	projects := map[uint64]bool{}
	for _, s := range rows {
		projects[s.ProjectID] = true
	}
	ids := make([]uint64, 0, len(projects))
	for id := range projects {
		ids = append(ids, id)
	}
	var locked []model.Sample
	if err := r.DB.WithContext(ctx).Select("project_id, sampled_on, platform, qid, round, sample_mode, prompt_revision").
		Where("project_id IN ? AND manual_override = ?", ids, true).Find(&locked).Error; err != nil {
		return nil, err
	}
	if len(locked) == 0 {
		return rows, nil
	}
	skip := map[string]bool{}
	for _, s := range locked {
		skip[sampleKey(s)] = true
	}
	out := rows[:0:0]
	for _, s := range rows {
		if !skip[sampleKey(s)] || s.ManualOverride {
			out = append(out, s)
		}
	}
	return out, nil
}
