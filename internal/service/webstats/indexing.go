// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

func (s *Service) inspectDue(ctx context.Context, token string, projectID uint64, property string, now time.Time) (string, error) {
	candidates, err := s.rows.IndexCandidates(ctx, projectID, property)
	if err != nil {
		return "", err
	}
	inventory := make([]model.IndexURL, 0, len(candidates))
	for _, candidate := range candidates {
		u := strings.TrimSpace(candidate.URL)
		if !urlInProperty(property, u) {
			continue
		}
		inventory = append(inventory, model.IndexURL{ProjectID: projectID, Property: property, URL: u, FromCrawl: candidate.FromCrawl, FromSearch: candidate.FromSearch, FirstSeenAt: now.Unix(), LastSeenAt: now.Unix()})
	}
	if err := s.rows.DiscoverIndexURLs(ctx, inventory); err != nil {
		return "", err
	}
	targets, err := s.rows.DueIndexURLs(ctx, projectID, property, now.Unix(), indexInspectCap)
	if err != nil {
		return "", err
	}
	failed := 0
	var first error
	for _, target := range targets {
		row, requestErr := s.client().InspectURLContext(ctx, token, property, target.URL)
		if requestErr != nil && (errors.Is(syncBudgetError(ctx, requestErr), ErrSyncBudget) || ctx.Err() != nil) {
			return "", syncBudgetError(ctx, requestErr)
		}
		var result *model.GscIndex
		failure := ""
		if requestErr == nil {
			result = &row
		} else {
			failure = requestErr.Error()
			failed++
			if first == nil {
				first = requestErr
			}
		}
		if err := s.rows.RecordInspection(ctx, target, result, failure, now); err != nil {
			return "", syncBudgetError(ctx, err)
		}
		if budget := budgetFrom(ctx); budget != nil {
			budget.mu.Lock()
			budget.committed++
			budget.mu.Unlock()
		}
		if requestErr != nil && (strings.Contains(failure, "HTTP 429") || strings.Contains(failure, "HTTP 403") || strings.Contains(failure, "HTTP 401") || strings.Contains(strings.ToLower(failure), "quota")) {
			break
		}
	}
	if failed > 0 {
		return fmt.Sprintf("%d URL inspections failed; last successful results retained: %v", failed, first), nil
	}
	return "", nil
}

func (s *Service) IndexInventory(ctx context.Context, slug, query, state string, page, size int) (*repo.IndexInventory, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.rows.IndexInventory(ctx, p.ID, s.indexProperty(ctx, p), query, state, page, size, s.now().Unix())
}
func (s *Service) IndexHistory(ctx context.Context, slug, url string, page, size int) (*repo.InspectionHistory, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.rows.IndexHistory(ctx, p.ID, s.indexProperty(ctx, p), url, page, size)
}

// Explicit property edits take effect immediately, before the next sync updates
// the active ledger. Never show a previous property's inspection as current.
func (s *Service) indexProperty(ctx context.Context, p *model.Project) string {
	if strings.TrimSpace(p.GscSite) != "" {
		return configuredProperty(p, "gsc")
	}
	active := s.officialProperty(ctx, p, "gsc")
	if !p.NoSite && urlInProperty(active, p.Site) {
		return active
	}
	return configuredProperty(p, "gsc")
}
