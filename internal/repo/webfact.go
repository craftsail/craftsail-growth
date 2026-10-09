// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func (r *Webstats) UpsertGscFacts(ctx context.Context, rows []model.GscFact) error {
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		if rows[i].KeyHash == "" {
			rows[i].KeyHash = model.RowKey(rows[i].SearchType, rows[i].Hour, rows[i].Query, rows[i].Page, rows[i].Country, rows[i].Device, rows[i].SearchAppearance)
		}
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "property"}, {Name: "slice"}, {Name: "day"}, {Name: "key_hash"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) UpsertGaFacts(ctx context.Context, rows []model.GaFact) error {
	if len(rows) == 0 {
		return nil
	}
	for i := range rows {
		if rows[i].KeyHash == "" {
			rows[i].KeyHash = model.RowKey(rows[i].Hour, rows[i].Source, rows[i].Medium, rows[i].Campaign, rows[i].Channel, rows[i].Landing, rows[i].PagePath, rows[i].PageTitle, rows[i].Country, rows[i].Device, rows[i].EventName)
			if rows[i].Hostname != "" {
				rows[i].KeyHash = model.RowKey(rows[i].KeyHash, rows[i].Hostname)
			}
		}
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "property"}, {Name: "report"}, {Name: "day"}, {Name: "key_hash"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) AppendRaw(ctx context.Context, row *model.GoogleRaw) error {
	return r.DB.WithContext(ctx).Create(row).Error
}

func (r *Webstats) GetSync(ctx context.Context, projectID uint64, source string) (time.Time, error) {
	var st model.WebSyncState
	err := r.DB.WithContext(ctx).Where("project_id = ? AND source = ?", projectID, source).First(&st).Error
	if err == gorm.ErrRecordNotFound {
		return time.Time{}, nil
	}
	return st.LastEnd, err
}

func (r *Webstats) ActivateProperty(ctx context.Context, projectID uint64, source, key, timezone string) error {
	now := time.Now()
	var existing []model.WebProperty
	if err := r.DB.WithContext(ctx).Where("project_id = ? AND source = ? AND status = ?", projectID, source, "active").Find(&existing).Error; err != nil {
		return err
	}
	for i := range existing {
		if existing[i].PropertyKey == key {
			continue
		}
		archived := now.Unix()
		existing[i].Status = "archived"
		existing[i].ArchivedAt = &archived
		if err := r.DB.WithContext(ctx).Save(&existing[i]).Error; err != nil {
			return err
		}
	}
	row := model.WebProperty{
		ProjectID: projectID, Source: source, PropertyKey: key, Status: "active", Timezone: timezone, ActivatedAt: now.Unix(),
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "source"}, {Name: "property_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "timezone", "activated_at", "archived_at"}),
	}).Create(&row).Error
}

func (r *Webstats) ActiveProperty(ctx context.Context, projectID uint64, source string) (string, error) {
	var row model.WebProperty
	err := r.DB.WithContext(ctx).Where("project_id = ? AND source = ? AND status = ?", projectID, source, "active").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return row.PropertyKey, err
}

func (r *Webstats) UpsertGscDaily(ctx context.Context, rows []model.GscDaily) error {
	if len(rows) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "property"}, {Name: "search_type"}, {Name: "day"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) UpsertGaDaily(ctx context.Context, rows []model.GaDaily) error {
	if len(rows) == 0 {
		return nil
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "property"}, {Name: "day"}},
		UpdateAll: true,
	}).CreateInBatches(&rows, upsertBatchSize).Error
}

func (r *Webstats) ListGscDaily(ctx context.Context, projectID uint64, property string, from, to time.Time) ([]model.GscDaily, error) {
	var out []model.GscDaily
	err := r.DB.WithContext(ctx).
		Where("project_id = ? AND property = ? AND date(day) >= ? AND date(day) <= ?", projectID, property, from.Format("2006-01-02"), to.Format("2006-01-02")).
		Order("day").
		Find(&out).Error
	return out, err
}

func (r *Webstats) ListGaDaily(ctx context.Context, projectID uint64, property string, from, to time.Time) ([]model.GaDaily, error) {
	var out []model.GaDaily
	err := r.DB.WithContext(ctx).
		Where("project_id = ? AND property = ? AND date(day) >= ? AND date(day) <= ?", projectID, property, from.Format("2006-01-02"), to.Format("2006-01-02")).
		Order("day").
		Find(&out).Error
	return out, err
}

func (r *Webstats) UpsertImport(ctx context.Context, row *model.WebImport) error {
	if row.UpdatedAt == 0 {
		row.UpdatedAt = time.Now().Unix()
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "source"}},
		UpdateAll: true,
	}).Create(row).Error
}

func (r *Webstats) GetImport(ctx context.Context, projectID uint64, source string) (*model.WebImport, error) {
	var row model.WebImport
	err := r.DB.WithContext(ctx).Where("project_id = ? AND source = ?", projectID, source).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Webstats) UpsertWindow(ctx context.Context, row *model.WebWindow) error {
	if row.ComputedAt == 0 {
		row.ComputedAt = time.Now().Unix()
	}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "source"}, {Name: "property"}, {Name: "window_days"}, {Name: "finalized_through"}},
		UpdateAll: true,
	}).Create(row).Error
}

func (r *Webstats) LatestWindow(ctx context.Context, projectID uint64, source, property string) (*model.WebWindow, error) {
	var row model.WebWindow
	err := r.DB.WithContext(ctx).
		Where("project_id = ? AND source = ? AND property = ?", projectID, source, property).
		Order("finalized_through desc").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Webstats) PutSync(ctx context.Context, projectID uint64, source string, lastEnd time.Time) error {
	st := model.WebSyncState{ProjectID: projectID, Source: source, LastEnd: lastEnd, UpdatedAt: time.Now().Unix()}
	return r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "source"}},
		UpdateAll: true,
	}).Create(&st).Error
}

// ListQueryPage returns the web query x page slice for one property.
func (r *Webstats) ListQueryPage(ctx context.Context, projectID uint64, property string, from, to time.Time) ([]model.GscFact, error) {
	return r.ListGscSlice(ctx, projectID, property, "query_page", from, to)
}

// ListGscSlice keeps report grain explicit; no fallback to query/page details.
func (r *Webstats) ListGscSlice(ctx context.Context, projectID uint64, property, slice string, from, to time.Time) ([]model.GscFact, error) {
	var out []model.GscFact
	err := r.DB.WithContext(ctx).
		Where("project_id = ? AND property = ? AND slice = ? AND search_type = ? AND date(day) >= ? AND date(day) <= ?",
			projectID, property, slice, "web", from.Format("2006-01-02"), to.Format("2006-01-02")).
		Find(&out).Error
	return out, err
}

// ListGaSessionFacts returns the GA4 session report rows for one property.
func (r *Webstats) ListGaSessionFacts(ctx context.Context, projectID uint64, property string, from, to time.Time) ([]model.GaFact, error) {
	var out []model.GaFact
	err := r.DB.WithContext(ctx).
		Where("project_id = ? AND property = ? AND report = ? AND date(day) >= ? AND date(day) <= ?",
			projectID, property, "session", from.Format("2006-01-02"), to.Format("2006-01-02")).
		Find(&out).Error
	return out, err
}

// PruneRaw deletes raw Google responses older than days. 0 keeps all.
func (r *Webstats) PruneRaw(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		return 0, nil
	}
	cut := time.Now().AddDate(0, 0, -days).Unix()
	res := r.DB.WithContext(ctx).Where("fetched_at < ?", cut).Delete(&model.GoogleRaw{})
	return res.RowsAffected, res.Error
}
