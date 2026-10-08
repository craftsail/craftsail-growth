// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

type dateChunk struct {
	start time.Time
	end   time.Time
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

type factTask struct {
	source, property, report, searchType string
	through                              time.Time
	stage, covered                       int
}

func factTasks(gscKey, gaKey string, endGSC, endGA time.Time) []factTask {
	var tasks []factTask
	if gaKey != "" {
		for _, report := range []string{"channel", "landing", "session", "page", "event", "hour"} {
			tasks = append(tasks, factTask{source: "ga4", property: gaKey, report: report, through: endGA})
		}
	}
	if gscKey != "" {
		for _, typ := range GSCSearchTypes {
			for _, slice := range gscSlices() {
				for _, supported := range slice.Types {
					if supported == typ {
						tasks = append(tasks, factTask{source: "gsc", property: gscKey, report: slice.Name, searchType: typ, through: endGSC})
						break
					}
				}
			}
		}
	}
	return tasks
}
func (t factTask) from() time.Time {
	if t.report == "hour" {
		return t.through.AddDate(0, 0, -9)
	}
	return t.through.AddDate(0, -16, 0)
}
func (t factTask) key() string { return t.source + "/" + t.report + "/" + t.searchType }

func (s *Service) SyncFacts(ctx context.Context, projectID uint64, token, gscSite, gaProp string, now time.Time) (string, error) {
	ctx = s.trafficQuotaContext(ctx, gscSite, gaProp)
	gscKey, _ := GSCPropertyKey(gscSite)
	gaKey, _ := GAPropertyKey(gaProp)
	endGSC, _ := FinalizedThrough(now, "")
	endGA := gaFinalizedThrough(now, s.propertyTimezone(ctx, projectID, "ga4", gaKey))
	blocked := map[string]bool{}
	var failures []error
	for source, property := range map[string]string{"gsc": gscKey, "ga4": gaKey} {
		imp, err := s.rows.GetImport(ctx, projectID, source)
		if err != nil {
			return "", syncBudgetError(ctx, err)
		}
		if imp != nil && imp.Property == property {
			if imp.FinalizedThrough != nil {
				if source == "gsc" {
					endGSC = dateOnly(*imp.FinalizedThrough)
				} else {
					endGA = dateOnly(*imp.FinalizedThrough)
				}
			}
			if imp.LastErrorClass == "rate_limited" || imp.LastErrorClass == "needs_reauth" {
				blocked[source] = true
				failures = append(failures, fmt.Errorf("%s: %s", source, imp.LastErrorClass))
			}
		}
	}
	tasks := factTasks(gscKey, gaKey, endGSC, endGA)
	if budget := budgetFrom(ctx); budget != nil {
		// Serve recent core reports first, then recent secondary reports, before
		// consuming the remaining budget on history. Lower coverage wins ties.
		reports := map[string]model.WebSyncReport{}
		for source, property := range map[string]string{"gsc": gscKey, "ga4": gaKey} {
			rows, err := s.rows.SyncReports(ctx, projectID, source, property)
			if err != nil {
				return "", syncBudgetError(ctx, err)
			}
			for _, r := range rows {
				if r.Version == currentSyncVersion {
					reports[r.Source+"/"+r.Report+"/"+r.SearchType] = r
				}
			}
		}
		for i := range tasks {
			t := &tasks[i]
			secondary := t.source == "gsc" && t.searchType != "web"
			if secondary {
				t.stage = 1
			}
			if r, ok := reports[t.key()]; ok {
				if r.State == "unsupported" && r.UpdatedAt >= budget.options.RefreshBefore.Unix() {
					t.stage = 9
					continue
				}
				days, err := s.rows.SyncDays(ctx, r.ID, t.from(), t.through)
				if err != nil {
					return "", syncBudgetError(ctx, err)
				}
				t.covered = len(days)
				recent := t.through.AddDate(0, 0, -55)
				if t.from().After(recent) {
					recent = t.from()
				}
				count := 0
				for _, d := range days {
					if !d.Day.Before(recent) {
						count++
					}
				}
				if count >= int(t.through.Sub(recent).Hours()/24)+1 {
					t.stage += 2
				}
			}
		}
		sort.SliceStable(tasks, func(i, j int) bool {
			if tasks[i].stage != tasks[j].stage {
				return tasks[i].stage < tasks[j].stage
			}
			return tasks[i].covered < tasks[j].covered
		})
	}
	for _, task := range tasks {
		if errors.Is(context.Cause(ctx), ErrSyncBudget) {
			return "", errors.Join(append(failures, ErrSyncBudget)...)
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if blocked[task.source] || task.stage == 9 {
			continue
		}
		maxChunks := 8
		if budgetFrom(ctx) != nil {
			maxChunks = 2
		}
		checked := false
		chunkDays := 7
		if task.source == "gsc" {
			chunkDays = 1
		}
		err := s.syncReport(ctx, projectID, task.source, task.property, task.report, task.searchType, task.from(), task.through, chunkDays, maxChunks, func(partCtx context.Context, part dateChunk) (repo.SyncBatch, error) {
			start, end := part.start.Format("2006-01-02"), part.end.Format("2006-01-02")
			if task.source == "gsc" {
				rows, quality, err := s.client().FetchGSCReport(partCtx, token, task.property, task.report, task.searchType, start, end)
				if err != nil {
					return repo.SyncBatch{}, err
				}
				for i := range rows {
					rows[i].ProjectID = projectID
					rows[i].Property = task.property
					rows[i].FetchedAt = now.Unix()
				}
				return repo.SyncBatch{GSC: rows, Quality: quality}, nil
			}
			if !checked && (task.report == "channel" || task.report == "landing") {
				if err := s.client().CheckGACompatibility(partCtx, token, task.property, task.report); err != nil {
					return repo.SyncBatch{}, err
				}
				checked = true
			}
			rows, quality, err := s.client().FetchGAReport(partCtx, token, task.property, task.report, start, end)
			if err != nil {
				return repo.SyncBatch{}, err
			}
			for i := range rows {
				rows[i].ProjectID = projectID
				rows[i].Property = task.property
				rows[i].FetchedAt = now.Unix()
			}
			return repo.SyncBatch{GA: rows, Quality: quality}, nil
		})
		if errors.Is(err, ErrSyncBudget) {
			return "", errors.Join(append(failures, ErrSyncBudget)...)
		}
		if err != nil {
			class, _ := classifyErr(err)
			if class == "rate_limited" || class == "needs_reauth" {
				blocked[task.source] = true
			}
			failures = append(failures, fmt.Errorf("%s: %w", task.key(), err))
		}
	}
	return "", errors.Join(failures...)
}

// Missing templates also count as pending: an exhausted batch may not have
// reached BeginSyncReport for them yet. Unsupported reports are terminal.
func (s *Service) pendingSync(ctx context.Context, projectID uint64, gscSite, gaProp string) (bool, error) {
	gscKey, _ := GSCPropertyKey(gscSite)
	gaKey, _ := GAPropertyKey(gaProp)
	tasks := factTasks(gscKey, gaKey, time.Time{}, time.Time{})
	if gscKey != "" {
		tasks = append(tasks, factTask{source: "gsc", property: gscKey, report: "daily", searchType: "web"})
	}
	if gaKey != "" {
		tasks = append(tasks, factTask{source: "ga4", property: gaKey, report: "daily"})
	}
	reports := map[string]model.WebSyncReport{}
	for source, property := range map[string]string{"gsc": gscKey, "ga4": gaKey} {
		rows, err := s.rows.SyncReports(ctx, projectID, source, property)
		if err != nil {
			return false, err
		}
		for _, r := range rows {
			if r.Version == currentSyncVersion {
				reports[r.Source+"/"+r.Report+"/"+r.SearchType] = r
			}
		}
	}
	for _, t := range tasks {
		r, ok := reports[t.key()]
		if !ok || r.State == "backfilling" || r.State == "running" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) archivePage(ctx context.Context, projectID uint64, source, report, request, body string, now time.Time) {
	if strings.TrimSpace(body) == "" {
		return
	}
	_ = s.rows.AppendRaw(ctx, &model.GoogleRaw{
		ProjectID: projectID,
		Source:    source,
		Report:    report,
		Request:   request,
		Body:      body,
		FetchedAt: now.Unix(),
	})
}
