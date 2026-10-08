// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"github.com/craftsail/craftsail-growth/internal/model"
	"sort"
	"time"
)

// SearchOp describes a diagnostic lead, never proof of causality.
type SearchOp struct {
	Reference *CTRReference      `json:"reference,omitempty"`
	Reason    string             `json:"reason,omitempty"`
	Facts     map[string]float64 `json:"facts,omitempty"`
	Type      string             `json:"type"`
	Title     string             `json:"title"`
	Detail    string             `json:"detail"`
	Query     string             `json:"query,omitempty"`
	URL       string             `json:"url,omitempty"`
	// URLs lists associated pages, largest first.
	URLs     []string `json:"urls,omitempty"`
	Metric   float64  `json:"metric,omitempty"`
	Severity string   `json:"severity"`
}

type PageClicks struct {
	URL       string
	Current   float64
	Previous  float64
	Sustained bool
}

type QueryPage struct {
	Query       string
	Page        string
	Impressions float64
	Clicks      float64
	Position    float64
}

type SearchPolicy struct {
	Mode                                          string
	MinImpressions                                int
	QueriesCovered, PagesComparable, PairsCovered bool
}

// Fixed CTR expectations cannot establish a problem for this site. Until a
// local reference is available, low CTR and multi-page queries stay facts.
func SearchOpportunities(kws []KeywordRow, pages []PageClicks, pairs []QueryPage, policy SearchPolicy) ([]SearchOp, []SearchOp) {
	minimum := policy.MinImpressions
	if minimum < 100 {
		minimum = 500
	}
	var actions, observations []SearchOp
	for _, k := range kws {
		if k.Position < 4 || k.Position > 20 || k.Impressions < 20 {
			continue
		}
		op := SearchOp{Type: "striking_distance", Title: k.Query, Query: k.Query, URL: k.Page, Metric: k.Impressions, Severity: "low", Detail: formatStrike(k), Reason: "candidate", Facts: map[string]float64{"position": k.Position, "impressions": k.Impressions}}
		switch {
		case !policy.QueriesCovered:
			op.Reason = "coverage"
		case k.Impressions < float64(minimum):
			op.Reason = "small_sample"
		case policy.Mode != "established":
			op.Reason = "new_site"
		default:
			op.Severity = "medium"
			actions = append(actions, op)
			continue
		}
		observations = append(observations, op)
	}
	if policy.PagesComparable {
		for _, p := range pages {
			if !meaningfulClickDrop(p.Current, p.Previous) || !p.Sustained {
				continue
			}
			change := (p.Current - p.Previous) / p.Previous * 100
			actions = append(actions, SearchOp{Type: "content_decay", Title: p.URL, URL: p.URL, Severity: "medium", Metric: change, Detail: formatDecay(p, change), Reason: "sustained_decline", Facts: map[string]float64{"current": p.Current, "previous": p.Previous}})
		}
	}
	byQ := map[string]map[string]*QueryPage{}
	for _, p := range pairs {
		if p.Page == "" {
			continue
		}
		if byQ[p.Query] == nil {
			byQ[p.Query] = map[string]*QueryPage{}
		}
		if cur := byQ[p.Query][p.Page]; cur != nil {
			cur.Impressions += p.Impressions
			cur.Clicks += p.Clicks
		} else {
			cp := p
			byQ[p.Query][p.Page] = &cp
		}
	}
	for q, pages := range byQ {
		if len(pages) < 2 {
			continue
		}
		var list []QueryPage
		var total float64
		for _, p := range pages {
			list = append(list, *p)
			total += p.Impressions
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].Impressions == list[j].Impressions {
				return list[i].Page < list[j].Page
			}
			return list[i].Impressions > list[j].Impressions
		})
		if list[0].Impressions < 20 {
			continue
		}
		var urls []string
		for _, p := range list {
			urls = append(urls, p.Page)
		}
		reason := "association_only"
		if !policy.PairsCovered {
			reason = "coverage"
		}
		observations = append(observations, SearchOp{Type: "multiple_pages", Title: q, Query: q, URL: list[0].Page, URLs: urls, Severity: "low", Metric: total, Reason: reason, Detail: "Several pages appeared for this query. Different search intents may explain this; the association alone does not justify merging pages."})
	}
	stable := func(rows []SearchOp) {
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Type != rows[j].Type {
				return rows[i].Type < rows[j].Type
			}
			if rows[i].Metric != rows[j].Metric {
				return rows[i].Metric > rows[j].Metric
			}
			if rows[i].Query != rows[j].Query {
				return rows[i].Query < rows[j].Query
			}
			return rows[i].URL < rows[j].URL
		})
	}
	stable(actions)
	stable(observations)
	return actions, observations
}

// Aggregate once per period, rather than scanning every fact for each page.
func markSustainedPages(current, previous []model.GscFact, drops []PageClicks, through time.Time) {
	sums := func(rows []model.GscFact, end time.Time) map[string][2]float64 {
		out := map[string][2]float64{}
		for _, row := range rows {
			days := int(end.Sub(dateOnly(row.Day)).Hours() / 24)
			if days < 0 || days >= 14 {
				continue
			}
			v := out[row.Page]
			v[days/7] += row.Clicks
			out[row.Page] = v
		}
		return out
	}
	cur, prev := sums(current, through), sums(previous, through.AddDate(0, 0, -28))
	for i := range drops {
		a, b := cur[drops[i].URL], prev[drops[i].URL]
		drops[i].Sustained = weeklyDrop(a[0], b[0]) && weeklyDrop(a[1], b[1])
	}
}

func formatStrike(k KeywordRow) string {
	return "Position " + format1(k.Position) + " with " + format0(k.Impressions) + " impressions. Review search intent and the associated page before choosing an action."
}

func formatDecay(p PageClicks, change float64) string {
	return "Clicks went from " + format0(p.Previous) + " to " + format0(p.Current) + " (" + format0(change) + "%) versus the previous 28 days."
}

func format1(v float64) string {
	return trimFloat(round2(v), 1)
}

func format0(v float64) string {
	return trimFloat(v, 0)
}

func formatPct(v float64) string {
	return trimFloat(round2(v*100), 1) + "%"
}

func trimFloat(v float64, places int) string {
	s := ""
	neg := v < 0
	if neg {
		v = -v
	}
	p := 1.0
	for i := 0; i < places; i++ {
		p *= 10
	}
	n := int64(v*p + 0.5)
	ip := n / int64(p)
	s = itoa64(ip)
	if places > 0 {
		frac := n % int64(p)
		fs := itoa64(frac)
		for len(fs) < places {
			fs = "0" + fs
		}
		s += "." + fs
	}
	if neg {
		return "-" + s
	}
	return s
}

func itoa(n int) string { return itoa64(int64(n)) }

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [24]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
