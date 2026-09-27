// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "sort"

// SearchOp kinds copy crawlseo lib/seo-opportunities.ts.
type SearchOp struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Query  string `json:"query,omitempty"`
	URL    string `json:"url,omitempty"`
	// URLs lists every competing page for cannibalization, largest first.
	URLs     []string `json:"urls,omitempty"`
	Metric   float64  `json:"metric,omitempty"`
	Severity string   `json:"severity"`
}

type PageClicks struct {
	URL      string
	Current  float64
	Previous float64
}

type QueryPage struct {
	Query       string
	Page        string
	Impressions float64
	Clicks      float64
	Position    float64
}

func expectedCtr(position float64) float64 {
	switch {
	case position <= 1:
		return 0.28
	case position <= 2:
		return 0.15
	case position <= 3:
		return 0.11
	case position <= 5:
		return 0.07
	case position <= 10:
		return 0.03
	case position <= 20:
		return 0.01
	default:
		return 0.005
	}
}

func SearchOpportunities(kws []KeywordRow, pages []PageClicks, pairs []QueryPage) []SearchOp {
	var striking, low, decay, cannibal []SearchOp
	for _, k := range kws {
		if k.Position >= 4 && k.Position <= 20 && k.Impressions >= 20 {
			sev := "medium"
			if k.Impressions > 200 {
				sev = "high"
			}
			striking = append(striking, SearchOp{
				Type: "striking_distance", Title: k.Query, Query: k.Query, Severity: sev, Metric: k.Impressions,
				Detail: formatStrike(k),
			})
		}
		if k.Impressions >= 50 && k.Position <= 15 {
			exp := expectedCtr(k.Position)
			gap := exp - k.CTR
			if gap > 0.02 {
				sev := "medium"
				if gap > 0.05 {
					sev = "high"
				}
				low = append(low, SearchOp{
					Type: "low_ctr", Title: k.Query, Query: k.Query, Severity: sev, Metric: k.Impressions,
					Detail: formatLowCtr(k, exp),
				})
			}
		}
	}
	for _, p := range pages {
		if p.Previous < 10 {
			continue
		}
		change := (p.Current - p.Previous) / p.Previous * 100
		if change <= -25 {
			sev := "medium"
			if change < -50 {
				sev = "high"
			}
			decay = append(decay, SearchOp{
				Type: "content_decay", Title: p.URL, URL: p.URL, Severity: sev, Metric: change,
				Detail: formatDecay(p, change),
			})
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
		cur := byQ[p.Query][p.Page]
		if cur == nil {
			cp := p
			byQ[p.Query][p.Page] = &cp
			continue
		}
		cur.Impressions += p.Impressions
		cur.Clicks += p.Clicks
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
		sort.Slice(list, func(i, j int) bool { return list[i].Impressions > list[j].Impressions })
		if list[0].Impressions < 20 {
			continue
		}
		urls := make([]string, 0, len(list))
		for _, p := range list {
			urls = append(urls, p.Page)
		}
		cannibal = append(cannibal, SearchOp{
			Type: "cannibalization", Title: q, Query: q, URL: list[0].Page, URLs: urls, Severity: "medium", Metric: total,
			Detail: formatCannibal(q, list),
		})
	}
	sort.Slice(striking, func(i, j int) bool { return striking[i].Metric > striking[j].Metric })
	sort.Slice(low, func(i, j int) bool { return low[i].Metric > low[j].Metric })
	sort.Slice(decay, func(i, j int) bool { return decay[i].Metric < decay[j].Metric })
	sort.Slice(cannibal, func(i, j int) bool { return cannibal[i].Metric > cannibal[j].Metric })
	cap := func(s []SearchOp, n int) []SearchOp {
		if len(s) > n {
			return s[:n]
		}
		return s
	}
	var out []SearchOp
	out = append(out, cap(striking, 8)...)
	out = append(out, cap(low, 8)...)
	out = append(out, cap(decay, 6)...)
	out = append(out, cap(cannibal, 6)...)
	return out
}

func formatStrike(k KeywordRow) string {
	return "Position " + format1(k.Position) + " with " + format0(k.Impressions) + " impressions; close to the top three."
}

func formatLowCtr(k KeywordRow, exp float64) string {
	return "CTR " + formatPct(k.CTR) + " where about " + formatPct(exp) + " is typical for this position; improve the title and description."
}

func formatDecay(p PageClicks, change float64) string {
	return "Clicks went from " + format0(p.Previous) + " to " + format0(p.Current) + " (" + format0(change) + "%) versus the previous 28 days."
}

func formatCannibal(q string, pages []QueryPage) string {
	return itoa(len(pages)) + " pages compete for \"" + q + "\". The one shown most is " + pages[0].Page + "."
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
