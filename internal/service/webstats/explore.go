// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/repo"
)

type ExploreInput struct {
	From      string `form:"from" json:"from"`
	Through   string `form:"through" json:"through"`
	Text      string `form:"q" json:"q"`
	Country   string `form:"country" json:"country"`
	Device    string `form:"device" json:"device"`
	Sort      string `form:"sort" json:"sort"`
	Direction string `form:"direction" json:"direction"`
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"page_size" json:"page_size"`
	Value     string `form:"value" json:"value"`
}

type SearchMetric = repo.SearchMetric

type SearchExplore struct {
	Items            []repo.SearchMetric `json:"items"`
	Total            int64               `json:"total"`
	Page             int                 `json:"page"`
	PageSize         int                 `json:"page_size"`
	Coverage         GrainCoverage       `json:"coverage"`
	PreviousCoverage GrainCoverage       `json:"previous_coverage"`
	Comparable       bool                `json:"comparable"`
	Filters          ExploreInput        `json:"filters"`
}

type InvalidSearchInput struct{ Reason string }

func (e *InvalidSearchInput) Error() string { return e.Reason }
func invalidSearch(reason string) error     { return &InvalidSearchInput{Reason: reason} }

func (s *Service) searchFilter(ctx context.Context, slug, kind string, in ExploreInput) (repo.SearchFilter, ExploreInput, error) {
	var f repo.SearchFilter
	if kind != "page" && kind != "query" {
		return f, in, invalidSearch("kind must be page or query")
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return f, in, err
	}
	now := s.now()
	_, end := window(now)
	property := s.officialProperty(ctx, p, "gsc")
	if imp, err := s.rows.GetImport(ctx, p.ID, "gsc"); err != nil {
		return f, in, err
	} else if imp != nil && imp.Property == property && imp.FinalizedThrough != nil {
		end = dateOnly(*imp.FinalizedThrough)
	}
	start := end.AddDate(0, 0, -27)
	if (in.From == "") != (in.Through == "") {
		return f, in, invalidSearch("from and through must be supplied together")
	}
	if in.From != "" {
		start, err = time.Parse("2006-01-02", in.From)
		if err != nil {
			return f, in, invalidSearch("invalid from date")
		}
		end, err = time.Parse("2006-01-02", in.Through)
		if err != nil {
			return f, in, invalidSearch("invalid through date")
		}
	}
	if end.Before(start) || end.Sub(start) > 365*24*time.Hour || end.After(dateOnly(now)) {
		return f, in, invalidSearch("date range must be 1–366 days and cannot end in the future")
	}
	if in.Page == 0 {
		in.Page = 1
	}
	if in.PageSize == 0 {
		in.PageSize = 50
	}
	if in.Page < 1 || in.Page > 100000 || in.PageSize < 1 || in.PageSize > 200 {
		return f, in, invalidSearch("invalid pagination")
	}
	if len(in.Text) > 1000 || len(in.Value) > 8192 {
		return f, in, invalidSearch("search value too long")
	}
	in.Country = strings.ToLower(strings.TrimSpace(in.Country))
	if in.Country != "" {
		if len(in.Country) != 3 {
			return f, in, invalidSearch("country must be a three-letter code")
		}
		for _, c := range in.Country {
			if c < 'a' || c > 'z' {
				return f, in, invalidSearch("invalid country")
			}
		}
	}
	if in.Device != "" && in.Device != "desktop" && in.Device != "mobile" && in.Device != "tablet" {
		return f, in, invalidSearch("invalid device")
	}
	if in.Sort == "" {
		in.Sort = "clicks"
	}
	switch in.Sort {
	case "clicks", "impressions", "ctr", "position", "clicks_change", "name":
	default:
		return f, in, invalidSearch("invalid sort")
	}
	if in.Direction == "" {
		in.Direction = "desc"
		if in.Sort == "position" || in.Sort == "name" {
			in.Direction = "asc"
		}
	}
	if in.Direction != "asc" && in.Direction != "desc" {
		return f, in, invalidSearch("invalid direction")
	}
	in.From = start.Format("2006-01-02")
	in.Through = end.Format("2006-01-02")
	previous, _ := previousWindow(start, end)
	grain := kind
	if in.Country != "" || in.Device != "" {
		grain += "_country_device"
	}
	f = repo.SearchFilter{ProjectID: p.ID, Property: property, Slice: grain, Group: kind, From: start, Through: end, PreviousFrom: previous, Country: in.Country, Device: in.Device, Text: in.Text, Sort: in.Sort, Direction: in.Direction, Limit: in.PageSize, Offset: (in.Page - 1) * in.PageSize}
	return f, in, nil
}

func (s *Service) exploreCoverage(ctx context.Context, f repo.SearchFilter, in ExploreInput) (*SearchExplore, error) {
	cur, err := s.grainCoverage(ctx, f.ProjectID, f.Property, f.Slice, f.From, f.Through)
	if err != nil {
		return nil, err
	}
	prev, err := s.grainCoverage(ctx, f.ProjectID, f.Property, f.Slice, f.PreviousFrom, f.From.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	return &SearchExplore{Coverage: cur, PreviousCoverage: prev, Comparable: cur.State == "covered" && prev.State == "covered", Page: in.Page, PageSize: in.PageSize, Filters: in}, nil
}

func (s *Service) ExploreSearch(ctx context.Context, slug, kind string, in ExploreInput) (*SearchExplore, error) {
	f, in, err := s.searchFilter(ctx, slug, kind, in)
	if err != nil {
		return nil, err
	}
	out, err := s.exploreCoverage(ctx, f, in)
	if err != nil {
		return nil, err
	}
	out.Items, out.Total, err = s.rows.SearchMetrics(ctx, f)
	return out, err
}

// ExportSearch uses the same filters and grain as the list. Pagination is ignored.
func (s *Service) ExportSearch(ctx context.Context, slug, kind string, in ExploreInput, header func(*SearchExplore) error, visit func(repo.SearchMetric) error) error {
	f, in, err := s.searchFilter(ctx, slug, kind, in)
	if err != nil {
		return err
	}
	out, err := s.exploreCoverage(ctx, f, in)
	if err != nil {
		return err
	}
	if err := header(out); err != nil {
		return err
	}
	return s.rows.WalkSearchMetrics(ctx, f, visit)
}

type DetailDay struct {
	Day         string   `json:"day"`
	Clicks      *float64 `json:"clicks"`
	Impressions *float64 `json:"impressions"`
}
type SearchDetail struct {
	Kind             string            `json:"kind"`
	Value            string            `json:"value"`
	Summary          repo.SearchMetric `json:"summary"`
	Coverage         GrainCoverage     `json:"coverage"`
	PreviousCoverage GrainCoverage     `json:"previous_coverage"`
	Comparable       bool              `json:"comparable"`
	Daily            []DetailDay       `json:"daily"`
	Related          *SearchExplore    `json:"related"`
}

func (s *Service) SearchDetail(ctx context.Context, slug, kind string, in ExploreInput) (*SearchDetail, error) {
	if in.Value == "" {
		return nil, invalidSearch("value is required")
	}
	f, in, err := s.searchFilter(ctx, slug, kind, in)
	if err != nil {
		return nil, err
	}
	coverage, err := s.exploreCoverage(ctx, f, in)
	if err != nil {
		return nil, err
	}
	out := &SearchDetail{Kind: kind, Value: in.Value, Coverage: coverage.Coverage, PreviousCoverage: coverage.PreviousCoverage, Comparable: coverage.Comparable}
	f.Text = ""
	if kind == "page" {
		f.ExactPage = in.Value
	} else {
		f.ExactQuery = in.Value
	}
	totals := f
	totals.Offset = 0
	totals.Limit = 1
	rows, _, err := s.rows.SearchMetrics(ctx, totals)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		out.Summary = rows[0]
	} else {
		out.Summary.Name = in.Value
	}
	timeline := totals
	timeline.Group = "day"
	timeline.PreviousFrom = f.From
	timeline.Sort = "name"
	timeline.Direction = "asc"
	byDay := map[string]repo.SearchMetric{}
	if err := s.rows.WalkSearchMetrics(ctx, timeline, func(row repo.SearchMetric) error { byDay[row.Name] = row; return nil }); err != nil {
		return nil, err
	}
	reports, err := s.rows.SyncReports(ctx, f.ProjectID, "gsc", f.Property)
	if err != nil {
		return nil, err
	}
	covered := map[string]bool{}
	for _, r := range reports {
		if r.Report != f.Slice || r.SearchType != "web" || r.Version != currentSyncVersion {
			continue
		}
		days, err := s.rows.SyncDays(ctx, r.ID, f.From, f.Through)
		if err != nil {
			return nil, err
		}
		for _, d := range days {
			covered[d.Day.Format("2006-01-02")] = true
		}
	}
	for d := f.From; !d.After(f.Through); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		v := DetailDay{Day: key}
		if row, ok := byDay[key]; ok || covered[key] {
			clicks, impressions := row.Clicks, row.Impressions
			v.Clicks = &clicks
			v.Impressions = &impressions
		}
		out.Daily = append(out.Daily, v)
	}
	related := f
	related.Text = in.Text
	related.Slice = "query_page"
	if f.Country != "" || f.Device != "" {
		related.Slice = "query_page_country_device"
	}
	related.Group = "page"
	if kind == "page" {
		related.Group = "query"
	}
	out.Related, err = s.exploreCoverage(ctx, related, in)
	if err != nil {
		return nil, err
	}
	out.Related.Items, out.Related.Total, err = s.rows.SearchMetrics(ctx, related)
	if err != nil {
		return nil, fmt.Errorf("related search: %w", err)
	}
	return out, nil
}
