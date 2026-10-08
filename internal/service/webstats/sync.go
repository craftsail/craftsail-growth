// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

// v3 refetches legacy coverage after the daily/appearance request contract changed.
const currentSyncVersion = 3

var ErrMetricRestricted = errors.New("Google restricted a requested metric; previous data retained")

var ErrIncompleteReport = errors.New("Google report reached its row limit; previous data retained")

// coveragePlan refreshes the tail and fills missing dates, newest first. A
// successful empty day is covered. Legacy last_end cursors are not evidence.
func coveragePlan(days []model.WebSyncDay, from, through time.Time, chunkDays, maxChunks int) []dateChunk {
	return coveragePlanSince(days, from, through, chunkDays, maxChunks, time.Time{})
}

func coveragePlanSince(days []model.WebSyncDay, from, through time.Time, chunkDays, maxChunks int, refreshBefore time.Time) []dateChunk {
	covered := map[string]bool{}
	fresh := map[string]bool{}
	for _, d := range days {
		covered[d.Day.Format("2006-01-02")] = true
		fresh[d.Day.Format("2006-01-02")] = !refreshBefore.IsZero() && d.FetchedAt >= refreshBefore.Unix()
	}
	tail := through.AddDate(0, 0, -2)
	var parts []dateChunk
	for d := through; !d.Before(from) && len(parts) < maxChunks; {
		if covered[d.Format("2006-01-02")] && (d.Before(tail) || fresh[d.Format("2006-01-02")]) {
			d = d.AddDate(0, 0, -1)
			continue
		}
		end, start := d, d
		for n := 1; n < chunkDays; n++ {
			prev := start.AddDate(0, 0, -1)
			if prev.Before(from) || (covered[prev.Format("2006-01-02")] && (prev.Before(tail) || fresh[prev.Format("2006-01-02")])) {
				break
			}
			start = prev
		}
		parts = append(parts, dateChunk{start: start, end: end})
		d = start.AddDate(0, 0, -1)
	}
	return parts
}

type syncFetch func(context.Context, dateChunk) (repo.SyncBatch, error)

func (s *Service) syncReport(ctx context.Context, projectID uint64, source, property, report, searchType string, from, through time.Time, chunkDays, maxChunks int, fetch syncFetch) error {
	if s.projects != nil {
		p, err := (&repo.Projects{DB: s.rows.DB}).ByID(ctx, projectID)
		if err != nil {
			return err
		}
		if p != nil {
			if d, ok := parseDay(p.GoogleHistoryStart); ok && d.After(from) {
				from = d
				if from.After(through) {
					from = through
				}
			}
		}
	}
	if through.Before(from) {
		return nil
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return syncBudgetError(ctx, err)
	}
	state := model.WebSyncReport{ProjectID: projectID, Source: source, Property: property, Report: report, SearchType: searchType, Version: currentSyncVersion, Token: hex.EncodeToString(b), From: from, Through: through, UpdatedAt: time.Now().Unix()}
	if err := s.rows.BeginSyncReport(ctx, &state); err != nil {
		return syncBudgetError(ctx, err)
	}
	finish := func(status, class string) error {
		// Persist a cancelled request's state without extending network work.
		finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		return s.rows.FinishSyncReport(finishCtx, state, status, class)
	}
	days, err := s.rows.SyncDays(ctx, state.ID, from, through)
	if err != nil {
		_ = finish("failed", "storage")
		return syncBudgetError(ctx, err)
	}
	var refreshBefore time.Time
	if budget := budgetFrom(ctx); budget != nil {
		refreshBefore = budget.options.RefreshBefore
	}
	parts := coveragePlanSince(days, from, through, chunkDays, maxChunks, refreshBefore)
	for _, part := range parts {
		if err = takeSyncBudget(ctx, false); err != nil {
			if errors.Is(err, ErrSyncBudget) {
				_ = finish("backfilling", "budget")
				return ErrSyncBudget
			}
			_ = finish("paused", "cancelled")
			return err
		}
		requestCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		batch, fetchErr := fetch(requestCtx, part)
		fetchErr = syncBudgetError(requestCtx, fetchErr)
		cancel()
		if fetchErr != nil {
			status, class := "failed", "provider"
			switch {
			case errors.Is(fetchErr, ErrSyncBudget):
				status, class = "backfilling", "budget"
			case errors.Is(fetchErr, ErrSliceUnsupported):
				status, class = "unsupported", "unsupported"
			case errors.Is(fetchErr, ErrMetricRestricted):
				status, class = "partial", "restricted"
			case errors.Is(fetchErr, ErrIncompleteReport):
				status, class = "partial", "truncated"
			case errors.Is(fetchErr, context.Canceled), errors.Is(fetchErr, context.DeadlineExceeded):
				status, class = "paused", "cancelled"
			default:
				class, _ = classifyErr(fetchErr)
				if class == "rate_limited" {
					status = "paused"
				}
			}
			_ = finish(status, class)
			if errors.Is(fetchErr, ErrSliceUnsupported) {
				return nil
			}
			return fetchErr
		}
		if err = s.rows.ReplaceSyncBatch(ctx, state, part.start, part.end, batch); err != nil {
			if errors.Is(syncBudgetError(ctx, err), ErrSyncBudget) {
				_ = finish("backfilling", "budget")
				return ErrSyncBudget
			}
			_ = finish("failed", "storage")
			return err
		}
		if budget := budgetFrom(ctx); budget != nil {
			budget.mu.Lock()
			budget.committed++
			budget.mu.Unlock()
		}
	}
	days, err = s.rows.SyncDays(ctx, state.ID, from, through)
	if err != nil {
		_ = finish("failed", "storage")
		return syncBudgetError(ctx, err)
	}
	status := "completed"
	if len(days) < int(through.Sub(from).Hours()/24)+1 {
		status = "backfilling"
	}
	return finish(status, "")
}

type SyncGap struct {
	From    string `json:"from"`
	Through string `json:"through"`
}
type SyncProgress struct {
	Gaps              []SyncGap           `json:"gaps"`
	LastSuccessAt     int64               `json:"last_success_at"`
	RetryAt           int64               `json:"retry_at"`
	Quality           model.GoogleQuality `json:"quality"`
	Source            string              `json:"source"`
	Property          string              `json:"property"`
	Report            string              `json:"report"`
	SearchType        string              `json:"search_type"`
	State             string              `json:"state"`
	ErrorClass        string              `json:"error_class"`
	From              string              `json:"from"`
	Through           string              `json:"through"`
	CoveredDays       int                 `json:"covered_days"`
	TotalDays         int                 `json:"total_days"`
	RecentCoveredDays int                 `json:"recent_covered_days"`
	RecentTotalDays   int                 `json:"recent_total_days"`
	UpdatedAt         int64               `json:"updated_at"`
}

func (s *Service) syncProgress(ctx context.Context, projectID uint64, source, property string) ([]SyncProgress, error) {
	rows, err := s.rows.SyncReports(ctx, projectID, source, property)
	if err != nil {
		return nil, err
	}
	out := make([]SyncProgress, 0, len(rows))
	for _, row := range rows {
		if row.Version != currentSyncVersion {
			continue
		}
		days, err := s.rows.SyncDays(ctx, row.ID, row.From, row.Through)
		if err != nil {
			return nil, err
		}
		recent := row.Through.AddDate(0, 0, -55)
		if row.From.After(recent) {
			recent = row.From
		}
		quality := model.GoogleQuality{Known: len(days) > 0}
		n := 0
		lastSuccess := int64(0)
		covered := map[string]bool{}
		for _, day := range days {
			covered[day.Day.Format("2006-01-02")] = true
			if day.FetchedAt > lastSuccess {
				lastSuccess = day.FetchedAt
			}
			var q model.GoogleQuality
			if err := json.Unmarshal([]byte(day.QualityJSON), &q); err != nil {
				q.Known = false
			}
			mergeQuality(&quality, q)
			if !day.Day.Before(recent) {
				n++
			}
		}
		gaps := []SyncGap{}
		for d := row.From; !d.After(row.Through); d = d.AddDate(0, 0, 1) {
			key := d.Format("2006-01-02")
			if covered[key] {
				continue
			}
			if len(gaps) > 0 && gaps[len(gaps)-1].Through == d.AddDate(0, 0, -1).Format("2006-01-02") {
				gaps[len(gaps)-1].Through = key
			} else {
				gaps = append(gaps, SyncGap{key, key})
			}
		}
		family := "traffic/gsc"
		if source == "ga4" {
			family = "traffic/ga"
		}
		quota, err := s.rows.GoogleQuota(ctx, property, family)
		if err != nil {
			return nil, err
		}
		var retry int64
		if quota != nil && quota.BlockedUntil > s.now().Unix() {
			retry = quota.BlockedUntil
		}
		out = append(out, SyncProgress{Gaps: gaps, LastSuccessAt: lastSuccess, RetryAt: retry, Quality: quality, Source: source, Property: property, Report: row.Report, SearchType: row.SearchType, State: row.State, ErrorClass: row.ErrorClass, From: row.From.Format("2006-01-02"), Through: row.Through.Format("2006-01-02"), CoveredDays: len(days), TotalDays: int(row.Through.Sub(row.From).Hours()/24) + 1, RecentCoveredDays: n, RecentTotalDays: int(row.Through.Sub(recent).Hours()/24) + 1, UpdatedAt: row.UpdatedAt})
	}
	return out, nil
}
