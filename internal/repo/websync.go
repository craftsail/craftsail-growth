// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrSyncSuperseded = errors.New("Google sync superseded by a newer request")

func (r *Webstats) BeginSyncReport(ctx context.Context, row *model.WebSyncReport) error {
	token := row.Token
	row.KeyHash = model.RowKey(row.Property, row.Source, row.Report, row.SearchType, fmt.Sprint(row.Version))
	row.State, row.ErrorClass = "running", ""
	err := r.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "key_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"token", "state", "error_class", "from", "through", "updated_at"}),
	}).Create(row).Error
	if err != nil {
		return err
	}
	row.ID = 0
	if err := r.DB.WithContext(ctx).Where("project_id = ? AND key_hash = ?", row.ProjectID, row.KeyHash).First(row).Error; err != nil {
		return err
	}
	if row.Token != token {
		return ErrSyncSuperseded
	}
	return nil
}

func (r *Webstats) FinishSyncReport(ctx context.Context, row model.WebSyncReport, state, errorClass string) error {
	res := r.DB.WithContext(ctx).Model(&model.WebSyncReport{}).Where("id = ? AND token = ?", row.ID, row.Token).
		Updates(map[string]any{"state": state, "error_class": errorClass, "updated_at": time.Now().Unix(), "revision": gorm.Expr("revision + 1")})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSyncSuperseded
	}
	return nil
}

func (r *Webstats) SyncDays(ctx context.Context, reportID uint64, from, through time.Time) ([]model.WebSyncDay, error) {
	var rows []model.WebSyncDay
	err := r.DB.WithContext(ctx).Where("report_id = ? AND date(day) >= ? AND date(day) <= ?", reportID, from.Format("2006-01-02"), through.Format("2006-01-02")).Order("day").Find(&rows).Error
	return rows, err
}

func (r *Webstats) SyncReports(ctx context.Context, projectID uint64, source, property string) ([]model.WebSyncReport, error) {
	var rows []model.WebSyncReport
	err := r.DB.WithContext(ctx).Where("project_id = ? AND source = ? AND property = ?", projectID, source, property).Order("report, search_type").Find(&rows).Error
	return rows, err
}

// SyncBatch contains exactly one report's complete response. The caller must
// reject incomplete pagination before committing. Empty results are valid.
type SyncBatch struct {
	Quality  model.GoogleQuality
	GSC      []model.GscFact
	GA       []model.GaFact
	GSCDaily []model.GscDaily
	GADaily  []model.GaDaily
}

func (r *Webstats) ReplaceSyncBatch(ctx context.Context, report model.WebSyncReport, from, through time.Time, batch SyncBatch) error {
	if (len(batch.GSC) > 0 && (report.Source != "gsc" || report.Report == "daily")) ||
		(len(batch.GA) > 0 && (report.Source != "ga4" || report.Report == "daily")) ||
		(len(batch.GSCDaily) > 0 && (report.Source != "gsc" || report.Report != "daily")) ||
		(len(batch.GADaily) > 0 && (report.Source != "ga4" || report.Report != "daily")) {
		return errors.New("Google batch does not match report")
	}
	dayOK := func(d time.Time) bool {
		v := d.Format("2006-01-02")
		return !d.IsZero() && v >= from.Format("2006-01-02") && v <= through.Format("2006-01-02")
	}
	if through.Before(from) {
		return errors.New("invalid Google sync range")
	}
	for _, v := range batch.GSC {
		if !dayOK(v.Day) || v.ProjectID != report.ProjectID || v.Property != report.Property || v.Slice != report.Report || v.SearchType != report.SearchType {
			return errors.New("GSC fact outside sync scope")
		}
	}
	for _, v := range batch.GA {
		if !dayOK(v.Day) || v.ProjectID != report.ProjectID || v.Property != report.Property || v.Report != report.Report {
			return errors.New("GA fact outside sync scope")
		}
	}
	for _, v := range batch.GSCDaily {
		if !dayOK(v.Day) || v.ProjectID != report.ProjectID || v.Property != report.Property || v.SearchType != "web" {
			return errors.New("GSC daily outside sync scope")
		}
	}
	for _, v := range batch.GADaily {
		if !dayOK(v.Day) || v.ProjectID != report.ProjectID || v.Property != report.Property {
			return errors.New("GA daily outside sync scope")
		}
	}
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The conditional write locks the report until commit on both databases.
		// A late response from an older run cannot replace newer data.
		res := tx.Model(&model.WebSyncReport{}).Where("id = ? AND token = ?", report.ID, report.Token).
			Updates(map[string]any{"revision": gorm.Expr("revision + 1"), "updated_at": time.Now().Unix()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSyncSuperseded
		}
		q := tx.Where("project_id = ? AND property = ? AND date(day) >= ? AND date(day) <= ?", report.ProjectID, report.Property, from.Format("2006-01-02"), through.Format("2006-01-02"))
		rr := &Webstats{DB: tx}
		var err error
		switch {
		case report.Source == "gsc" && report.Report == "daily":
			err = q.Where("search_type = ?", "web").Delete(&model.GscDaily{}).Error
			if err == nil {
				err = rr.UpsertGscDaily(ctx, batch.GSCDaily)
			}
		case report.Source == "ga4" && report.Report == "daily":
			err = q.Delete(&model.GaDaily{}).Error
			if err == nil {
				err = rr.UpsertGaDaily(ctx, batch.GADaily)
			}
		case report.Source == "gsc":
			err = q.Where("slice = ? AND search_type = ?", report.Report, report.SearchType).Delete(&model.GscFact{}).Error
			if err == nil {
				err = rr.UpsertGscFacts(ctx, batch.GSC)
			}
		case report.Source == "ga4":
			err = q.Where("report = ?", report.Report).Delete(&model.GaFact{}).Error
			if err == nil {
				err = rr.UpsertGaFacts(ctx, batch.GA)
			}
		default:
			return errors.New("unknown Google sync source")
		}
		if err != nil {
			return err
		}
		quality, err := json.Marshal(batch.Quality)
		if err != nil {
			return err
		}
		var days []model.WebSyncDay
		for d := from; !d.After(through); d = d.AddDate(0, 0, 1) {
			days = append(days, model.WebSyncDay{ReportID: report.ID, Day: d, QualityJSON: string(quality), FetchedAt: time.Now().Unix()})
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "report_id"}, {Name: "day"}}, DoUpdates: clause.AssignmentColumns([]string{"fetched_at", "quality_json"})}).CreateInBatches(days, upsertBatchSize).Error
	})
}
