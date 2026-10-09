// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type QuotaWait struct{ Until int64 }

func (e *QuotaWait) Error() string {
	return fmt.Sprintf("Google property quota: retry after %s", time.Unix(e.Until, 0).UTC().Format(time.RFC3339))
}
func envBudget(key string, fallback, maximum int) int {
	n, err := strconv.Atoi(os.Getenv(key))
	if err != nil || n < 1 {
		return fallback
	}
	return min(n, maximum)
}
func (s *Service) reserveInspection(ctx context.Context, property string, now time.Time) error {
	// Do not reserve provider quota if this local request budget is already spent.
	if b := budgetFrom(ctx); b != nil {
		b.mu.Lock()
		spent := b.exhausted || b.requests >= b.options.MaxRequests
		if spent {
			b.exhausted = true
		}
		b.mu.Unlock()
		if spent {
			return ErrSyncBudget
		}
	}
	zone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return err
	}
	local := now.In(zone)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	retry, err := s.rows.ReserveGoogleRequest(ctx, property, "inspection", now.Unix(), day.Unix(), day.AddDate(0, 0, 1).Unix(), envBudget("GOOGLE_INDEX_DAILY_BUDGET", 1800, 2000), envBudget("GOOGLE_INDEX_MINUTE_BUDGET", 120, 600))
	if err != nil {
		return err
	}
	if retry > 0 {
		return &QuotaWait{Until: retry}
	}
	return nil
}

type IndexRunResult struct {
	Pending   bool   `json:"pending"`
	RetryAt   int64  `json:"retry_at"`
	Connected bool   `json:"connected"`
	Note      string `json:"note"`
}

func (s *Service) RunIndexBatch(ctx context.Context, slug string, startedAt time.Time) (*IndexRunResult, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	property := s.indexProperty(ctx, p)
	if property == "" {
		return nil, fmt.Errorf("set a website or Search Console property first")
	}
	if startedAt.IsZero() {
		startedAt = s.now()
	}
	batch, cancel, _ := withSyncBudget(ctx, BatchOptions{RefreshBefore: startedAt, MaxRequests: 40, MaxDuration: 2 * time.Minute})
	defer cancel()
	out := &IndexRunResult{Connected: UserConnected() || strings.TrimSpace(os.Getenv("GOOGLE_SA_JSON")) != ""}
	if err := s.discoverIndexSitemaps(batch, p, property, s.now()); err != nil {
		if errors.Is(context.Cause(batch), ErrSyncBudget) {
			out.Pending = true
			out.RetryAt = s.now().Add(15 * time.Second).Unix()
			return out, nil
		}
		return out, err
	}
	// Discovery works without Google access; inspection is a separate optional step.
	if err := s.discoverStoredURLs(batch, p.ID, property, s.now()); err != nil {
		return out, err
	}
	if out.Connected {
		token, err := s.accessTokenContext(batch, true)
		if err != nil {
			return out, err
		}
		maps, mapErr := s.client().FetchSitemapsContext(batch, token, property)
		if mapErr == nil {
			seeds := []string{}
			for i := range maps {
				maps[i].ProjectID = p.ID
				maps[i].FetchedAt = s.now().Unix()
				if sitemapAllowed(property, maps[i].Path) {
					seeds = append(seeds, maps[i].Path)
				}
			}
			if err := s.rows.UpsertSitemaps(batch, maps); err != nil {
				return out, err
			}
			if err := s.rows.QueueSitemaps(batch, p.ID, property, seeds); err != nil {
				return out, err
			}
		} else {
			out.Note = mapErr.Error()
		}
		note, err := s.inspectDue(batch, token, p.ID, property, s.now())
		if note != "" {
			out.Note = note
		}
		var wait *QuotaWait
		if errors.As(err, &wait) {
			out.Pending = true
			out.RetryAt = wait.Until
			return out, nil
		}
		if err != nil && !errors.Is(err, ErrSyncBudget) {
			return out, err
		}
	}
	scans, err := s.rows.DueSitemaps(ctx, p.ID, property, startedAt.Unix(), 1)
	if err != nil {
		return out, err
	}
	out.Pending = len(scans) > 0
	if out.Connected {
		due, err := s.rows.DueIndexURLs(ctx, p.ID, property, startedAt.Unix(), 1)
		if err != nil {
			return out, err
		}
		out.Pending = out.Pending || len(due) > 0
	}
	if out.Pending {
		out.RetryAt = s.now().Add(15 * time.Second).Unix()
	}
	if days, _ := strconv.Atoi(os.Getenv("GOOGLE_INDEX_HISTORY_DAYS")); days > 0 {
		if err := s.rows.PruneIndexHistory(ctx, days); err != nil {
			return out, err
		}
	}
	return out, nil
}
