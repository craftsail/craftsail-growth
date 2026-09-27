// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

const insightTopN = 20

type insightQuery struct {
	query       string
	clicks      float64
	impressions float64
	weightedPos float64
}

type insightLanding struct {
	landing  string
	sessions float64
}

// Totals are official date-only sums for the window. Nil means not measured.
type Totals struct {
	Clicks      *float64
	Impressions *float64
	Sessions    *float64
}

// Insight computes the search insight for the default window on read. Site
// totals come from gsc_dailies / ga_dailies; query rows and GA4 sources are
// drill-down only and never used as totals.
func (s *Service) Insight(ctx context.Context, p *model.Project) (map[string]any, error) {
	startDay, endDay := window(s.now())
	var t Totals
	var rows []QueryRow
	var ga []model.GaFact
	if prop := s.officialProperty(ctx, p, "gsc"); prop != "" {
		daily, err := s.rows.ListGscDaily(ctx, p.ID, prop, startDay, endDay)
		if err != nil {
			return nil, err
		}
		if len(daily) > 0 {
			var c, i float64
			for _, d := range daily {
				if d.SearchType != "" && d.SearchType != "web" {
					continue
				}
				c += d.Clicks
				i += d.Impressions
			}
			t.Clicks, t.Impressions = &c, &i
		}
		facts, err := s.rows.ListQueryPage(ctx, p.ID, prop, startDay, endDay)
		if err != nil {
			return nil, err
		}
		rows = QueryRowsFromFacts(facts)
	}
	if prop := s.officialProperty(ctx, p, "ga4"); prop != "" {
		daily, err := s.rows.ListGaDaily(ctx, p.ID, prop, startDay, endDay)
		if err != nil {
			return nil, err
		}
		if len(daily) > 0 {
			var n float64
			for _, d := range daily {
				n += d.Sessions
			}
			t.Sessions = &n
		}
		facts, err := s.rows.ListGaSessionFacts(ctx, p.ID, prop, startDay, endDay)
		if err != nil {
			return nil, err
		}
		ga = facts
	}
	questions, err := (&repo.Questions{DB: s.rows.DB}).List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	texts := make([]string, len(questions))
	for i := range questions {
		texts[i] = questions[i].Text
	}
	return buildInsight(startDay, endDay, t, rows, ga, texts), nil
}

func buildInsight(from, to time.Time, t Totals, rows []QueryRow, ga []model.GaFact, questions []string) map[string]any {
	byQuery := map[string]*insightQuery{}
	for _, row := range rows {
		st := byQuery[row.Query]
		if st == nil {
			st = &insightQuery{query: row.Query}
			byQuery[row.Query] = st
		}
		st.clicks += row.Clicks
		st.impressions += row.Impressions
		st.weightedPos += row.Position * row.Impressions
	}

	var keyEvents, aiSessions float64
	byLanding := map[string]*insightLanding{}
	for _, row := range ga {
		keyEvents += row.KeyEvents
		if AISource(row.Source, row.Medium) {
			aiSessions += row.Sessions
		}
		st := byLanding[row.Landing]
		if st == nil {
			st = &insightLanding{landing: row.Landing}
			byLanding[row.Landing] = st
		}
		st.sessions += row.Sessions
	}

	queries := make([]*insightQuery, 0, len(byQuery))
	for _, st := range byQuery {
		queries = append(queries, st)
	}
	sort.Slice(queries, func(i, j int) bool {
		if queries[i].clicks != queries[j].clicks {
			return queries[i].clicks > queries[j].clicks
		}
		if queries[i].impressions != queries[j].impressions {
			return queries[i].impressions > queries[j].impressions
		}
		return queries[i].query < queries[j].query
	})

	top := make([]map[string]any, 0)
	var gaps []*insightQuery
	for _, st := range queries {
		if len(top) < insightTopN {
			pos := 0.0
			if st.impressions > 0 {
				pos = st.weightedPos / st.impressions
			}
			top = append(top, map[string]any{
				"query":       st.query,
				"clicks":      st.clicks,
				"impressions": st.impressions,
				"position":    pos,
			})
		}
		if st.impressions >= 10 && !queryCovered(st.query, questions) {
			gaps = append(gaps, st)
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].impressions != gaps[j].impressions {
			return gaps[i].impressions > gaps[j].impressions
		}
		if gaps[i].clicks != gaps[j].clicks {
			return gaps[i].clicks > gaps[j].clicks
		}
		return gaps[i].query < gaps[j].query
	})
	gapOut := make([]map[string]any, 0)
	for i, st := range gaps {
		if i >= insightTopN {
			break
		}
		gapOut = append(gapOut, map[string]any{
			"query":       st.query,
			"impressions": st.impressions,
			"clicks":      st.clicks,
		})
	}

	landings := make([]*insightLanding, 0, len(byLanding))
	for _, st := range byLanding {
		landings = append(landings, st)
	}
	sort.Slice(landings, func(i, j int) bool {
		if landings[i].sessions != landings[j].sessions {
			return landings[i].sessions > landings[j].sessions
		}
		return landings[i].landing < landings[j].landing
	})
	topLandings := make([]map[string]any, 0)
	for i, st := range landings {
		if i >= insightTopN {
			break
		}
		topLandings = append(topLandings, map[string]any{
			"landing":  st.landing,
			"sessions": st.sessions,
		})
	}

	return map[string]any{
		"from":            from.Format("2006-01-02"),
		"to":              to.Format("2006-01-02"),
		"gsc_clicks":      t.Clicks,
		"gsc_impressions": t.Impressions,
		"ga_sessions":     t.Sessions,
		"ga_key_events":   keyEvents,
		"ai_sessions":     aiSessions,
		"top_queries":     top,
		"gap_queries":     gapOut,
		"top_landings":    topLandings,
	}
}

// queryCovered is true when the lowercased query contains a 4-rune slice of a
// question, or a latin token of length >= 4 appears in a question.
func queryCovered(query string, questions []string) bool {
	q := strings.ToLower(query)
	tokens := latinTokens(q)
	for _, text := range questions {
		qt := strings.ToLower(text)
		if containsRuneSlice(q, qt, 4) {
			return true
		}
		for _, tok := range tokens {
			if strings.Contains(qt, tok) {
				return true
			}
		}
	}
	return false
}

func containsRuneSlice(query, question string, n int) bool {
	runes := []rune(question)
	if len(runes) < n {
		return false
	}
	for i := 0; i+n <= len(runes); i++ {
		if strings.Contains(query, string(runes[i:i+n])) {
			return true
		}
	}
	return false
}

// latinTokens splits on non-latin letters and keeps tokens of rune length >= 4.
func latinTokens(q string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		tok := b.String()
		b.Reset()
		if len([]rune(tok)) >= 4 {
			out = append(out, tok)
		}
	}
	for _, r := range q {
		if unicode.IsLetter(r) && unicode.Is(unicode.Latin, r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}
