// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

type GAMetric = repo.GAMetric

type GAExplore struct {
	Items            []repo.GAMetric     `json:"items"`
	Total            int64               `json:"total"`
	Page             int                 `json:"page"`
	PageSize         int                 `json:"page_size"`
	Filters          ExploreInput        `json:"filters"`
	Property         string              `json:"property"`
	Timezone         string              `json:"timezone"`
	ReportState      string              `json:"report_state"`
	ErrorClass       string              `json:"error_class"`
	Coverage         GrainCoverage       `json:"coverage"`
	PreviousCoverage GrainCoverage       `json:"previous_coverage"`
	Quality          model.GoogleQuality `json:"quality"`
	PreviousQuality  model.GoogleQuality `json:"previous_quality"`
	Comparable       bool                `json:"comparable"`
}

func (s *Service) analyticsFilter(ctx context.Context, slug, report string, in ExploreInput) (repo.GAFilter, *GAExplore, error) {
	var f repo.GAFilter
	if report != "channel" && report != "landing" {
		return f, nil, invalidSearch("invalid GA report")
	}
	if in.Country != "" || in.Device != "" || in.Value != "" {
		return f, nil, invalidSearch("filters are not available in this GA report")
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return f, nil, err
	}
	property := s.officialProperty(ctx, p, "ga4")
	timezone := s.propertyTimezone(ctx, p.ID, "ga4", property)
	now := s.now()
	through := gaFinalizedThrough(now, timezone)
	if imp, err := s.rows.GetImport(ctx, p.ID, "ga4"); err != nil {
		return f, nil, err
	} else if imp != nil && imp.Property == property && imp.FinalizedThrough != nil {
		through = dateOnly(*imp.FinalizedThrough)
	}
	from := through.AddDate(0, 0, -27)
	if (in.From == "") != (in.Through == "") {
		return f, nil, invalidSearch("from and through must be supplied together")
	}
	if in.From != "" {
		from, err = time.Parse("2006-01-02", in.From)
		if err != nil {
			return f, nil, invalidSearch("invalid from date")
		}
		through, err = time.Parse("2006-01-02", in.Through)
		if err != nil {
			return f, nil, invalidSearch("invalid through date")
		}
	}
	if through.Before(from) || through.Sub(from) > 365*24*time.Hour || through.After(dateOnly(now)) {
		return f, nil, invalidSearch("date range must be 1–366 days and cannot end in the future")
	}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 50
	}
	if in.Page < 1 || in.Page > 100000 || in.PageSize < 1 || in.PageSize > 200 {
		return f, nil, invalidSearch("invalid pagination")
	}
	if len(in.Text) > 1000 {
		return f, nil, invalidSearch("search value too long")
	}
	if in.Sort == "" {
		in.Sort = "sessions"
	}
	switch in.Sort {
	case "sessions", "engaged", "key_events", "engagement_rate", "duration", "duration_per_session", "sessions_change":
	default:
		return f, nil, invalidSearch("invalid GA sort")
	}
	if in.Direction == "" {
		in.Direction = "desc"
	}
	if in.Direction != "asc" && in.Direction != "desc" {
		return f, nil, invalidSearch("invalid direction")
	}
	in.From = from.Format("2006-01-02")
	in.Through = through.Format("2006-01-02")
	previousFrom, previousThrough := previousWindow(from, through)
	f = repo.GAFilter{ProjectID: p.ID, Property: property, Report: report, Text: in.Text, From: from, Through: through, PreviousFrom: previousFrom, Sort: in.Sort, Direction: in.Direction, Offset: (in.Page - 1) * in.PageSize, Limit: in.PageSize}
	out := &GAExplore{Page: in.Page, PageSize: in.PageSize, Filters: in, Property: property, Timezone: timezone, ReportState: "missing"}
	reports, err := s.rows.SyncReports(ctx, p.ID, "ga4", property)
	if err != nil {
		return f, nil, err
	}
	var active *model.WebSyncReport
	for i := range reports {
		r := &reports[i]
		if r.Report == report && r.Version == currentSyncVersion {
			active = r
			out.ReportState = r.State
			out.ErrorClass = r.ErrorClass
			break
		}
	}
	coverage := func(start, end time.Time) (GrainCoverage, model.GoogleQuality, error) {
		c := GrainCoverage{Report: report, From: start.Format("2006-01-02"), Through: end.Format("2006-01-02"), TotalDays: int(end.Sub(start).Hours()/24) + 1, State: "missing"}
		quality := model.GoogleQuality{}
		if active == nil {
			return c, quality, nil
		}
		days, err := s.rows.SyncDays(ctx, active.ID, start, end)
		if err != nil {
			return c, quality, err
		}
		c.CoveredDays = len(days)
		if len(days) > 0 {
			c.State = "partial"
		}
		if c.CoveredDays == c.TotalDays {
			c.State = "covered"
		}
		quality.Known = len(days) > 0
		for _, day := range days {
			var q model.GoogleQuality
			if err := json.Unmarshal([]byte(day.QualityJSON), &q); err != nil {
				q.Known = false
			}
			mergeQuality(&quality, q)
		}
		return c, quality, nil
	}
	out.Coverage, out.Quality, err = coverage(from, through)
	if err != nil {
		return f, nil, err
	}
	out.PreviousCoverage, out.PreviousQuality, err = coverage(previousFrom, previousThrough)
	if err != nil {
		return f, nil, err
	}
	trustworthy := func(q model.GoogleQuality) bool {
		return q.Known && !q.Sampled && !q.Thresholded && !q.OtherRow && !q.Restricted && !q.EmptyReason
	}
	out.Comparable = out.Coverage.State == "covered" && out.PreviousCoverage.State == "covered" && trustworthy(out.Quality) && trustworthy(out.PreviousQuality) && len(out.Quality.TimeZones) == 1 && len(out.PreviousQuality.TimeZones) == 1 && out.Quality.TimeZones[0] == out.PreviousQuality.TimeZones[0]
	return f, out, nil
}

func (s *Service) ExploreGA(ctx context.Context, slug, report string, in ExploreInput) (*GAExplore, error) {
	f, out, err := s.analyticsFilter(ctx, slug, report, in)
	if err != nil {
		return nil, err
	}
	out.Items, out.Total, err = s.rows.GAMetrics(ctx, f)
	return out, err
}
func (s *Service) ExportGA(ctx context.Context, slug, report string, in ExploreInput, header func(*GAExplore) error, visit func(GAMetric) error) error {
	f, out, err := s.analyticsFilter(ctx, slug, report, in)
	if err != nil {
		return err
	}
	if err := header(out); err != nil {
		return err
	}
	return s.rows.WalkGAMetrics(ctx, f, visit)
}
