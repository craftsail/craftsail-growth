// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// These thresholds are product heuristics, not statistical guarantees. Only
// visible nonbrand queries contribute; this is not a universal rank curve.
const ctrVersion = 1
const ctrMinQueries = 10
const ctrMinImpressions = 1000

type CTRReference struct {
	Version     int      `json:"version"`
	From        string   `json:"from"`
	Through     string   `json:"through"`
	Band        string   `json:"band"`
	Scope       string   `json:"scope"`
	Country     string   `json:"country,omitempty"`
	Device      string   `json:"device,omitempty"`
	Queries     int      `json:"queries"`
	Impressions float64  `json:"impressions"`
	CTR         *float64 `json:"ctr"`
	Reason      string   `json:"reason"`
}
type ctrPool struct {
	rows    []KeywordRow
	scope   string
	trusted bool
}
type ctrReferences struct {
	from, through, country, device string
	pools                          []ctrPool
	project                        *model.Project
}

func trustedSearch(c GrainCoverage) bool {
	q := c.Quality
	return c.State == "covered" && q.Known && !q.Sampled && !q.Thresholded && !q.OtherRow && !q.Restricted && !q.EmptyReason && len(q.Aggregations) == 1
}
func ctrBand(position float64) string {
	switch {
	case position >= 1 && position <= 3:
		return "1–3"
	case position > 3 && position <= 6:
		return "4–6"
	case position > 6 && position <= 10:
		return "7–10"
	case position > 10 && position <= 20:
		return "11–20"
	}
	return ""
}
func nonbrand(query string, p *model.Project) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "site:") || model.IsPromptBranded(query, p.Name, p.Brand.Aliases, p.Site) {
		return false
	}
	compact := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, s)
	}
	q := compact(query)
	for _, s := range append([]string{p.Name}, p.Brand.Aliases...) {
		s = compact(s)
		if len([]rune(s)) >= 3 && strings.Contains(q, s) {
			return false
		}
	}
	return true
}
func (s *Service) ctrReferences(ctx context.Context, p *model.Project, property string, end time.Time, country, device string) (ctrReferences, error) {
	start := end.AddDate(0, 0, -89)
	out := ctrReferences{from: start.Format("2006-01-02"), through: end.Format("2006-01-02"), country: country, device: device, project: p}
	load := func(grain, scope string) error {
		coverage, err := s.grainCoverage(ctx, p.ID, property, grain, start, end)
		if err != nil {
			return err
		}
		pool := ctrPool{scope: scope, trusted: trustedSearch(coverage)}
		if pool.trusted {
			rows, err := s.rows.ListGscSlice(ctx, p.ID, property, grain, start, end)
			if err != nil {
				return err
			}
			filtered := rows[:0]
			for _, r := range rows {
				if scope == "site" || (country == "" || r.Country == country) && (device == "" || strings.EqualFold(r.Device, device)) {
					filtered = append(filtered, r)
				}
			}
			pool.rows = AggregateKeywords(QueryRowsFromFacts(filtered))
		}
		out.pools = append(out.pools, pool)
		return nil
	}
	if country != "" || device != "" {
		if err := load("query_country_device", "segment"); err != nil {
			return out, err
		}
	}
	err := load("query", "site")
	return out, err
}
func (r ctrReferences) reference(query string, position float64) CTRReference {
	out := CTRReference{Version: ctrVersion, From: r.from, Through: r.through, Band: ctrBand(position), Country: r.country, Device: r.device, Reason: "insufficient"}
	if !nonbrand(query, r.project) {
		out.Reason = "brand"
		return out
	}
	if out.Band == "" {
		out.Reason = "rank"
		return out
	}
	anyTrusted := false
	for _, pool := range r.pools {
		if !pool.trusted {
			continue
		}
		anyTrusted = true
		clicks, impressions := 0.0, 0.0
		count := 0
		for _, k := range pool.rows {
			if strings.EqualFold(strings.TrimSpace(k.Query), strings.TrimSpace(query)) || !nonbrand(k.Query, r.project) || ctrBand(k.Position) != out.Band || k.Impressions <= 0 {
				continue
			}
			clicks += k.Clicks
			impressions += k.Impressions
			count++
		}
		out.Scope, out.Queries, out.Impressions = pool.scope, count, impressions
		if count >= ctrMinQueries && impressions >= ctrMinImpressions {
			v := clicks / impressions
			out.CTR = &v
			out.Reason = "ready"
			return out
		}
	}
	if !anyTrusted {
		out.Reason = "quality"
	}
	return out
}
func ctrOpportunities(kws []KeywordRow, refs ctrReferences, minimum int) []SearchOp {
	if minimum < 100 {
		minimum = 500
	}
	var out []SearchOp
	for _, k := range kws {
		if k.Impressions < float64(minimum) {
			continue
		}
		ref := refs.reference(k.Query, k.Position)
		if ref.CTR == nil {
			continue
		}
		scenario := k.Impressions * math.Max(*ref.CTR-k.CTR, 0)
		if scenario < 10 {
			continue
		}
		out = append(out, SearchOp{Type: "low_ctr", Title: k.Query, Query: k.Query, URL: k.Page, Severity: "low", Reason: "local_ctr", Metric: scenario, Reference: &ref, Facts: map[string]float64{"scenario_clicks": scenario, "reference_ctr": *ref.CTR, "ctr": k.CTR, "impressions": k.Impressions}, Detail: "Visible nonbrand queries in this site's rank band provide a reference. Additional clicks are a scenario, not a forecast or proof of a title problem."})
	}
	return out
}

func comparableSearch(a, b GrainCoverage) bool {
	return trustedSearch(a) && trustedSearch(b) && a.Quality.Aggregations[0] == b.Quality.Aggregations[0]
}
