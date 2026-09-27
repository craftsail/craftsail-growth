// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"math"
	"sort"
)

// KeywordRow is one query after collapsing query×page×day rows.
// Position is impression-weighted, matching crawlseo lib/google/aggregate.ts.
type KeywordRow struct {
	Query       string  `json:"query"`
	Page        string  `json:"page"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
	Band        string  `json:"band"`
}

type PageRow struct {
	URL         string  `json:"url"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
	Band        string  `json:"band"`
}

// PositionBand copies crawlseo lib/seo-metrics.ts positionBand.
func PositionBand(position float64) string {
	if position > 0 && position <= 3 {
		return "top3"
	}
	if position <= 10 {
		return "top10"
	}
	if position <= 20 {
		return "top20"
	}
	return "deep"
}

func AggregateKeywords(rows []QueryRow) []KeywordRow {
	type acc struct {
		clicks, impr, weighted, pageImpr float64
		topPage                          string
	}
	by := map[string]*acc{}
	var order []string
	for _, r := range rows {
		a := by[r.Query]
		if a == nil {
			a = &acc{}
			by[r.Query] = a
			order = append(order, r.Query)
		}
		a.clicks += r.Clicks
		a.impr += r.Impressions
		w := r.Impressions
		if w < 1 {
			w = 1
		}
		a.weighted += r.Position * w
		a.pageImpr += w
	}
	topImpr := map[string]float64{}
	for _, r := range rows {
		if r.Impressions >= topImpr[r.Query] {
			topImpr[r.Query] = r.Impressions
			by[r.Query].topPage = r.Page
		}
	}
	var out []KeywordRow
	for _, q := range order {
		a := by[q]
		pos := 0.0
		if a.pageImpr > 0 {
			pos = a.weighted / a.pageImpr
		}
		ctr := 0.0
		if a.impr > 0 {
			ctr = a.clicks / a.impr
		}
		out = append(out, KeywordRow{
			Query: q, Page: a.topPage, Clicks: a.clicks, Impressions: a.impr,
			CTR: round4(ctr), Position: round2(pos), Band: PositionBand(pos),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Clicks == out[j].Clicks {
			return out[i].Impressions > out[j].Impressions
		}
		return out[i].Clicks > out[j].Clicks
	})
	return out
}

func AggregatePages(rows []QueryRow) []PageRow {
	type acc struct {
		clicks, impr, weighted, weight float64
	}
	by := map[string]*acc{}
	var order []string
	for _, r := range rows {
		if r.Page == "" {
			continue
		}
		a := by[r.Page]
		if a == nil {
			a = &acc{}
			by[r.Page] = a
			order = append(order, r.Page)
		}
		a.clicks += r.Clicks
		a.impr += r.Impressions
		w := r.Impressions
		if w < 1 {
			w = 1
		}
		a.weighted += r.Position * w
		a.weight += w
	}
	var out []PageRow
	for _, u := range order {
		a := by[u]
		pos := 0.0
		if a.weight > 0 {
			pos = a.weighted / a.weight
		}
		ctr := 0.0
		if a.impr > 0 {
			ctr = a.clicks / a.impr
		}
		out = append(out, PageRow{
			URL: u, Clicks: a.clicks, Impressions: a.impr,
			CTR: round4(ctr), Position: round2(pos), Band: PositionBand(pos),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Clicks > out[j].Clicks })
	return out
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
