// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"time"
)

const stageMinImpressions = 1000
const stageMinClicks = 100

type SearchWeek struct {
	From        string   `json:"from"`
	Through     string   `json:"through"`
	CoveredDays int      `json:"covered_days"`
	Clicks      *float64 `json:"clicks"`
	Impressions *float64 `json:"impressions"`
}
type SearchObservation struct {
	Configured           string        `json:"configured"`
	Mode                 string        `json:"mode"`
	Reason               string        `json:"reason"`
	Property             string        `json:"property"`
	EstablishedThrough   string        `json:"established_through,omitempty"`
	MinImpressions       int           `json:"min_impressions"`
	Coverage             GrainCoverage `json:"coverage"`
	PreviousCoverage     GrainCoverage `json:"previous_coverage"`
	PageCoverage         GrainCoverage `json:"page_coverage"`
	PagesWithImpressions *int64        `json:"pages_with_impressions"`
	Weeks                []SearchWeek  `json:"weeks"`
}

func (s *Service) searchWindow(ctx context.Context, p *model.Project) (time.Time, time.Time, string, error) {
	from, through := window(s.now())
	property := s.officialProperty(ctx, p, "gsc")
	imp, err := s.rows.GetImport(ctx, p.ID, "gsc")
	if err != nil {
		return from, through, property, err
	}
	if imp != nil && imp.Property == property && imp.FinalizedThrough != nil {
		through = dateOnly(*imp.FinalizedThrough)
		from = through.AddDate(0, 0, -27)
	}
	return from, through, property, nil
}
func (s *Service) searchObservation(ctx context.Context, p *model.Project) (*SearchObservation, []model.GscDaily, error) {
	from, through, property, err := s.searchWindow(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	previousFrom, previousThrough := previousWindow(from, through)
	out := &SearchObservation{Configured: p.SearchMode, Mode: "new_site", Reason: "insufficient_history", Property: property, MinImpressions: p.SearchMinImpressions}
	if out.Configured == "" {
		out.Configured = "auto"
	}
	if out.MinImpressions < 100 {
		out.MinImpressions = 500
	}
	if out.Configured != "auto" {
		out.Mode = out.Configured
		out.Reason = "manual"
	} else {
		record, err := s.rows.SearchProperty(ctx, p.ID, property)
		if err != nil {
			return nil, nil, err
		}
		if record != nil && record.SearchEstablishedThrough != nil && record.SearchStageVersion == currentSyncVersion {
			out.Mode = "established"
			out.Reason = "qualified_history"
			out.EstablishedThrough = record.SearchEstablishedThrough.Format("2006-01-02")
		}
	}
	out.Coverage, err = s.grainCoverage(ctx, p.ID, property, "daily", from, through)
	if err != nil {
		return nil, nil, err
	}
	out.PreviousCoverage, err = s.grainCoverage(ctx, p.ID, property, "daily", previousFrom, previousThrough)
	if err != nil {
		return nil, nil, err
	}
	out.PageCoverage, err = s.grainCoverage(ctx, p.ID, property, "page", from, through)
	if err != nil {
		return nil, nil, err
	}
	rows, err := s.rows.ListGscDaily(ctx, p.ID, property, previousFrom, through)
	if err != nil {
		return nil, nil, err
	}
	if out.PageCoverage.State == "covered" {
		n, err := s.rows.PagesWithImpressions(ctx, p.ID, property, from, through)
		if err != nil {
			return nil, nil, err
		}
		out.PagesWithImpressions = &n
	}
	// Only complete Monday–Sunday weeks, never a partial current week.
	weekEnd := through.AddDate(0, 0, -int(through.Weekday()))
	for i := 3; i >= 0; i-- {
		end := weekEnd.AddDate(0, 0, -7*i)
		start := end.AddDate(0, 0, -6)
		coverage, err := s.grainCoverage(ctx, p.ID, property, "daily", start, end)
		if err != nil {
			return nil, nil, err
		}
		sum := sumOfficial(rows, start, end)
		week := SearchWeek{From: start.Format("2006-01-02"), Through: end.Format("2006-01-02"), CoveredDays: coverage.CoveredDays}
		if coverage.State == "covered" && sum.rows == 7 {
			week.Clicks = &sum.clicks
			week.Impressions = &sum.impr
		}
		out.Weeks = append(out.Weeks, week)
	}
	return out, rows, nil
}

// Promotion runs after successful sync, not during GET. Once qualified, a
// property keeps its stage through later low traffic; switching properties
// cannot reuse that evidence. Overrides never bypass diagnostic safeguards.
func (s *Service) promoteSearchStage(ctx context.Context, projectID uint64, property string, through time.Time) error {
	record, err := s.rows.SearchProperty(ctx, projectID, property)
	if err != nil || record == nil {
		return err
	}
	if record.SearchEstablishedThrough != nil && record.SearchStageVersion == currentSyncVersion {
		return nil
	}
	end := through.AddDate(0, 0, -int(through.Weekday()))
	start := end.AddDate(0, 0, -55)
	coverage, err := s.grainCoverage(ctx, projectID, property, "daily", start, end)
	if err != nil || coverage.State != "covered" {
		return err
	}
	rows, err := s.rows.ListGscDaily(ctx, projectID, property, start, end)
	if err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		b := end.AddDate(0, 0, -28*i)
		a := b.AddDate(0, 0, -27)
		sum := sumOfficial(rows, a, b)
		if sum.rows != 28 || sum.clicks < stageMinClicks || sum.impr < stageMinImpressions {
			return nil
		}
	}
	return s.rows.PromoteSearchStage(ctx, projectID, property, end, currentSyncVersion)
}
