// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type CiteGap struct {
	QID               string   `json:"qid"`
	Question          string   `json:"question"`
	CompetitorDomains []string `json:"competitor_domains"`
}

type CiteSnap struct {
	Categories     map[string]int `json:"categories"`
	NewURLs        []string       `json:"new_urls"`
	DroppedURLs    []string       `json:"dropped_urls"`
	Gaps           []CiteGap      `json:"gaps"`
	Stability      *int           `json:"stability_score"`
	StabilityLabel string         `json:"stability"`
	Opportunities  []Opportunity  `json:"opportunities"`
}

func BackfillCitations(ctx context.Context, db *gorm.DB) error {
	var missing []model.Sample
	err := db.WithContext(ctx).
		Where("NOT EXISTS (SELECT 1 FROM sample_citations WHERE sample_citations.sample_id = samples.id)").
		Find(&missing).Error
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	svc := New(db, nil)
	byProject := map[uint64][]model.Sample{}
	for _, sm := range missing {
		byProject[sm.ProjectID] = append(byProject[sm.ProjectID], sm)
	}
	for projectID, rows := range byProject {
		var p model.Project
		if err := db.WithContext(ctx).First(&p, projectID).Error; err != nil {
			return err
		}
		cfg, _, err := svc.cfgOf(ctx, &p)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, sm := range rows {
			key := sm.SampledOn.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true
			if err := svc.persistCitations(ctx, projectID, sm.SampledOn, cfg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) persistCitations(ctx context.Context, projectID uint64, day time.Time, cfg Cfg) error {
	rows, err := s.samples.List(ctx, projectID, day, "", "", 2000)
	if err != nil {
		return err
	}
	var ids []uint64
	var cites []model.SampleCitation
	hosts := competitorHosts(cfg)
	for _, sm := range rows {
		ids = append(ids, sm.ID)
		for i, c := range sm.Cited {
			if c.URL == "" {
				continue
			}
			domain := hostOf(c.URL)
			cites = append(cites, model.SampleCitation{
				ProjectID: projectID, SampleID: sm.ID, SampledOn: sm.SampledOn,
				Platform: sm.Platform, QID: sm.QID, Round: sm.Round, SampleMode: sm.SampleMode,
				URL: c.URL, Domain: domain, Title: c.Title, CitationIndex: i,
				Category: ClassifyDomain(domain, cfg.Site, hosts), PageType: PageType(c.URL),
			})
		}
	}
	return s.samples.ReplaceCitations(ctx, projectID, ids, cites)
}

func (s *Service) CitationSnap(ctx context.Context, projectID uint64, day time.Time) (CiteSnap, error) {
	return s.citationSnap(ctx, projectID, day)
}

func (s *Service) citationSnap(ctx context.Context, projectID uint64, day time.Time) (CiteSnap, error) {
	since := day.AddDate(0, 0, -30)
	rows, err := s.samples.ListCitations(ctx, projectID, since)
	if err != nil {
		return CiteSnap{}, err
	}
	snap := snapCitations(rows, day)
	samples, err := s.samples.List(ctx, projectID, time.Time{}, "", "", 2000)
	if err != nil {
		return snap, nil
	}
	text := map[string]string{}
	for _, sm := range samples {
		if sm.QuestionText != "" {
			text[sm.QID] = sm.QuestionText
		}
	}
	for i := range snap.Gaps {
		snap.Gaps[i].Question = text[snap.Gaps[i].QID]
	}
	var site string
	_ = s.samples.DB.WithContext(ctx).Model(&model.Project{}).Where("id = ?", projectID).Pluck("site", &site).Error
	snap.Opportunities = BuildOpportunities(rows, text, site, day)
	return snap, nil
}

func snapCitations(rows []model.SampleCitation, day time.Time) CiteSnap {
	today := day.Format("2006-01-02")
	cats := map[string]int{}
	todayURLs := map[string]bool{}
	prevURLs := map[string]bool{}
	byDay := map[string]map[string]int{}
	type gapAcc struct {
		question string
		own      bool
		comps    map[string]bool
	}
	gaps := map[string]*gapAcc{}
	var prevDay string
	for _, c := range rows {
		d := c.SampledOn.Format("2006-01-02")
		if byDay[d] == nil {
			byDay[d] = map[string]int{}
		}
		if c.Domain != "" {
			byDay[d][c.Domain]++
		}
		if d == today {
			cats[c.Category]++
			todayURLs[c.URL] = true
			g := gaps[c.QID]
			if g == nil {
				g = &gapAcc{comps: map[string]bool{}}
				gaps[c.QID] = g
			}
			if c.Category == "brand" {
				g.own = true
			}
			if c.Category == "competitor" && c.Domain != "" {
				g.comps[c.Domain] = true
			}
			continue
		}
		if prevDay == "" || d > prevDay {
			prevDay = d
		}
	}
	if prevDay != "" {
		for _, c := range rows {
			if c.SampledOn.Format("2006-01-02") == prevDay {
				prevURLs[c.URL] = true
			}
		}
	}
	var days []DayDomains
	for d, counts := range byDay {
		days = append(days, DayDomains{Date: d, Counts: counts})
	}
	score, label := Stability(days)
	snap := CiteSnap{
		Categories: cats, NewURLs: diffURLs(todayURLs, prevURLs), DroppedURLs: diffURLs(prevURLs, todayURLs),
		Gaps: []CiteGap{}, Stability: score, StabilityLabel: label,
	}
	for qid, g := range gaps {
		if g.own || len(g.comps) == 0 {
			continue
		}
		var domains []string
		for d := range g.comps {
			domains = append(domains, d)
		}
		snap.Gaps = append(snap.Gaps, CiteGap{QID: qid, Question: g.question, CompetitorDomains: domains})
	}
	if snap.Categories == nil {
		snap.Categories = map[string]int{}
	}
	return snap
}

func diffURLs(have, other map[string]bool) []string {
	var out []string
	for u := range have {
		if u != "" && !other[u] {
			out = append(out, u)
		}
	}
	return out
}

func citationsFromAnswer(answer string) []Citation {
	var out []Citation
	seen := map[string]bool{}
	for _, u := range urlRE.FindAllString(answer, -1) {
		if seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, Citation{URL: u})
	}
	return out
}

func envOf(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return "manual"
	}
	return mode
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func competitorHosts(cfg Cfg) []string {
	var hosts []string
	for _, c := range cfg.Competitors {
		if c.Site != "" {
			hosts = append(hosts, c.Site)
		}
	}
	return hosts
}
