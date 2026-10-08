// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"math"
	"time"

	"gorm.io/gorm/clause"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type Period struct {
	Comparable       bool     `json:"comparable"`
	PreviousClicks   float64  `json:"previous_clicks"`
	Clicks           float64  `json:"clicks"`
	Impressions      float64  `json:"impressions"`
	CTR              float64  `json:"ctr"`
	Position         float64  `json:"position"`
	ClicksDelta      *float64 `json:"clicks_delta"`
	ImpressionsDelta *float64 `json:"impressions_delta"`
	CTRDelta         *float64 `json:"ctr_delta"`
	PositionDelta    *float64 `json:"position_delta"`
	HasPrevious      bool     `json:"has_previous"`
	// Measured is false when no official date-only totals exist; the numbers
	// are then zero and must be shown as "not measured".
	Measured bool `json:"measured"`
}

type GrainCoverage struct {
	Report      string `json:"report"`
	From        string `json:"from"`
	Through     string `json:"through"`
	CoveredDays int    `json:"covered_days"`
	TotalDays   int    `json:"total_days"`
	State       string `json:"state"`
}

type SearchBoard struct {
	Observation   *SearchObservation `json:"observation"`
	Observations  []SearchOp         `json:"observations"`
	QueryCoverage GrainCoverage      `json:"query_coverage"`
	PageCoverage  GrainCoverage      `json:"page_coverage"`
	Keywords      []KeywordRow       `json:"keywords"`
	Pages         []PageRow          `json:"gsc_pages"`
	Period        Period             `json:"period"`
	Ops           []SearchOp         `json:"search_ops"`
}

func (s *Service) SearchBoard(ctx context.Context, slug string) (*SearchBoard, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	observation, official, err := s.searchObservation(ctx, p)
	if err != nil {
		return nil, err
	}
	start, _ := time.Parse("2006-01-02", observation.Coverage.From)
	end, _ := time.Parse("2006-01-02", observation.Coverage.Through)
	prevStart, prevEnd := previousWindow(start, end)
	property := observation.Property

	curFacts, err := s.rows.ListQueryPage(ctx, p.ID, property, start, end)
	if err != nil {
		return nil, err
	}
	queryFacts, err := s.rows.ListGscSlice(ctx, p.ID, property, "query", start, end)
	if err != nil {
		return nil, err
	}
	pageFacts, err := s.rows.ListGscSlice(ctx, p.ID, property, "page", start, end)
	if err != nil {
		return nil, err
	}
	prevPageFacts, err := s.rows.ListGscSlice(ctx, p.ID, property, "page", prevStart, prevEnd)
	if err != nil {
		return nil, err
	}
	qc, err := s.grainCoverage(ctx, p.ID, property, "query", start, end)
	if err != nil {
		return nil, err
	}
	pc, err := s.grainCoverage(ctx, p.ID, property, "page", start, end)
	if err != nil {
		return nil, err
	}
	ppc, err := s.grainCoverage(ctx, p.ID, property, "page", prevStart, prevEnd)
	if err != nil {
		return nil, err
	}
	cur := QueryRowsFromFacts(curFacts)

	kws := AggregateKeywords(QueryRowsFromFacts(queryFacts))
	// Associations are evidence only; they must not contribute to query totals.
	topPages := map[string]string{}
	for _, row := range AggregateKeywords(cur) {
		topPages[row.Query] = row.Page
	}
	for i := range kws {
		kws[i].Page = topPages[kws[i].Query]
	}
	pages := AggregatePages(QueryRowsFromFacts(pageFacts))
	period := periodFromOfficial(official, start, end, prevStart, prevEnd)
	period.Comparable = observation.Coverage.State == "covered" && observation.PreviousCoverage.State == "covered" && sumOfficial(official, start, end).rows == 28 && sumOfficial(official, prevStart, prevEnd).rows == 28
	if !period.Comparable {
		period.HasPrevious = false
		period.ClicksDelta = nil
		period.ImpressionsDelta = nil
		period.CTRDelta = nil
		period.PositionDelta = nil
	}
	var drops []PageClicks
	if pc.State == "covered" && ppc.State == "covered" {
		drops = pageClicks(QueryRowsFromFacts(pageFacts), QueryRowsFromFacts(prevPageFacts))
	}
	markSustainedPages(pageFacts, prevPageFacts, drops, end)
	pairCoverage, err := s.grainCoverage(ctx, p.ID, property, "query_page", start, end)
	if err != nil {
		return nil, err
	}
	ops, observations := SearchOpportunities(kws, drops, queryPages(cur), SearchPolicy{Mode: observation.Mode, MinImpressions: observation.MinImpressions, QueriesCovered: qc.State == "covered", PagesComparable: pc.State == "covered" && ppc.State == "covered", PairsCovered: pairCoverage.State == "covered"})
	ops = append(alertOps(period, official, end), ops...)
	return &SearchBoard{Observation: observation, Observations: observations, Keywords: kws, Pages: pages, Period: period, Ops: ops, QueryCoverage: qc, PageCoverage: pc}, nil
}

// grainCoverage never infers missing dates from facts: successful empty days
// count as requested, while legacy rows have unknown coverage.
func (s *Service) grainCoverage(ctx context.Context, projectID uint64, property, report string, from, through time.Time) (GrainCoverage, error) {
	out := GrainCoverage{Report: report, From: from.Format("2006-01-02"), Through: through.Format("2006-01-02"), TotalDays: int(through.Sub(from).Hours()/24) + 1, State: "missing"}
	reports, err := s.rows.SyncReports(ctx, projectID, "gsc", property)
	if err != nil {
		return out, err
	}
	for _, r := range reports {
		if r.Report != report || r.SearchType != "web" || r.Version != currentSyncVersion {
			continue
		}
		days, err := s.rows.SyncDays(ctx, r.ID, from, through)
		if err != nil {
			return out, err
		}
		out.CoveredDays = len(days)
		if len(days) > 0 {
			out.State = "partial"
		}
		if out.CoveredDays == out.TotalDays {
			out.State = "covered"
		}
	}
	return out, nil
}

func (s *Service) officialTotals(ctx context.Context, p *model.Project, from, to time.Time) ([]model.GscDaily, error) {
	property := s.officialProperty(ctx, p, "gsc")
	if property == "" {
		return nil, nil
	}
	return s.rows.ListGscDaily(ctx, p.ID, property, from, to)
}

func (s *Service) officialProperty(ctx context.Context, p *model.Project, source string) string {
	active, _ := s.rows.ActiveProperty(ctx, p.ID, source)
	property := active
	if property == "" {
		property = configuredProperty(p, source)
	}
	if imp, err := s.rows.GetImport(ctx, p.ID, source); err == nil && imp != nil && imp.Property != "" {
		property = imp.Property
	}
	return property
}

func previousWindow(start, end time.Time) (time.Time, time.Time) {
	days := int(end.Sub(start).Hours()/24) + 1
	prevEnd := start.AddDate(0, 0, -1)
	return prevEnd.AddDate(0, 0, -(days - 1)), prevEnd
}

func periodFromOfficial(rows []model.GscDaily, start, end, prevStart, prevEnd time.Time) Period {
	cur := sumOfficial(rows, start, end)
	prev := sumOfficial(rows, prevStart, prevEnd)
	out := finishPeriod(cur, prev)
	out.Measured = cur.rows > 0
	return out
}

func finishPeriod(cur, prev dailySum) Period {
	out := Period{PreviousClicks: prev.clicks, Clicks: cur.clicks, Impressions: cur.impr, CTR: cur.ctr, Position: round2(cur.pos)}
	if prev.impr == 0 && prev.clicks == 0 {
		return out
	}
	out.HasPrevious = true
	out.ClicksDelta = pctDelta(cur.clicks, prev.clicks)
	out.ImpressionsDelta = pctDelta(cur.impr, prev.impr)
	out.CTRDelta = pctDelta(cur.ctr, prev.ctr)
	d := prev.pos - cur.pos
	out.PositionDelta = &d
	return out
}

type dailySum struct {
	clicks, impr, weighted, weight, pos, ctr float64
	rows                                     int
}

func sumOfficial(rows []model.GscDaily, start, end time.Time) dailySum {
	var s dailySum
	for _, r := range rows {
		if r.SearchType != "" && r.SearchType != "web" {
			continue
		}
		day := time.Date(r.Day.Year(), r.Day.Month(), r.Day.Day(), 0, 0, 0, 0, time.UTC)
		a := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		b := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
		if day.Before(a) || day.After(b) {
			continue
		}
		s.rows++
		s.clicks += r.Clicks
		s.impr += r.Impressions
		w := r.Impressions
		if w < 1 {
			w = 1
		}
		s.weighted += r.Position * w
		s.weight += w
	}
	if s.impr > 0 {
		s.ctr = s.clicks / s.impr
	}
	if s.weight > 0 {
		s.pos = s.weighted / s.weight
	}
	return s
}

func pctDelta(cur, prev float64) *float64 {
	if prev == 0 {
		return nil
	}
	v := math.Round(((cur-prev)/prev)*100*100) / 100
	return &v
}

func pageClicks(cur, prev []QueryRow) []PageClicks {
	now := map[string]float64{}
	was := map[string]float64{}
	for _, r := range cur {
		now[r.Page] += r.Clicks
	}
	for _, r := range prev {
		was[r.Page] += r.Clicks
	}
	seen := map[string]bool{}
	var out []PageClicks
	for u, n := range was {
		seen[u] = true
		out = append(out, PageClicks{URL: u, Current: now[u], Previous: n})
	}
	return out
}

func queryPages(rows []QueryRow) []QueryPage {
	var out []QueryPage
	for _, r := range rows {
		out = append(out, QueryPage{Query: r.Query, Page: r.Page, Impressions: r.Impressions, Clicks: r.Clicks, Position: r.Position})
	}
	return out
}

func (s *Service) SaveKeyword(ctx context.Context, slug, query, notes string) error {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	row := model.SavedKeyword{ProjectID: p.ID, Query: query, Notes: notes}
	return s.rows.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "project_id"}, {Name: "query"}},
		DoUpdates: clause.AssignmentColumns([]string{"notes"}),
	}).Create(&row).Error
}

func (s *Service) ListSaved(ctx context.Context, slug string) ([]model.SavedKeyword, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	var out []model.SavedKeyword
	err = s.rows.DB.WithContext(ctx).Where("project_id = ?", p.ID).Order("created_at desc").Find(&out).Error
	return out, err
}

func (s *Service) DeleteSaved(ctx context.Context, slug, query string) error {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	return s.rows.DB.WithContext(ctx).Where("project_id = ? AND query = ?", p.ID, query).Delete(&model.SavedKeyword{}).Error
}
