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

type SearchBoard struct {
	Keywords []KeywordRow `json:"keywords"`
	Pages    []PageRow    `json:"gsc_pages"`
	Period   Period       `json:"period"`
	Ops      []SearchOp   `json:"search_ops"`
}

func (s *Service) SearchBoard(ctx context.Context, slug string) (*SearchBoard, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	start, end := window(now)
	prevStart, prevEnd := previousWindow(start, end)
	property := s.officialProperty(ctx, p, "gsc")
	curFacts, err := s.rows.ListQueryPage(ctx, p.ID, property, start, end)
	if err != nil {
		return nil, err
	}
	prevFacts, err := s.rows.ListQueryPage(ctx, p.ID, property, prevStart, prevEnd)
	if err != nil {
		return nil, err
	}
	cur, prev := QueryRowsFromFacts(curFacts), QueryRowsFromFacts(prevFacts)
	official, err := s.officialTotals(ctx, p, prevStart, end)
	if err != nil {
		return nil, err
	}
	kws := AggregateKeywords(cur)
	if len(kws) > 200 {
		kws = kws[:200]
	}
	pages := AggregatePages(cur)
	if len(pages) > 200 {
		pages = pages[:200]
	}
	period := periodFromOfficial(official, start, end, prevStart, prevEnd)
	ops := SearchOpportunities(AggregateKeywords(cur), pageClicks(cur, prev), queryPages(cur))
	ops = append(alertOps(period), ops...)
	return &SearchBoard{Keywords: kws, Pages: pages, Period: period, Ops: ops}, nil
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
	out := Period{Clicks: cur.clicks, Impressions: cur.impr, CTR: cur.ctr, Position: round2(cur.pos)}
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
		v := 0.0
		if cur > 0 {
			v = 100
		}
		return &v
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
