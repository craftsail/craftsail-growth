// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "github.com/craftsail/craftsail-growth/internal/model"

// QueryRow is one query x page pair over a window. It is a drill-down row:
// anonymized queries are missing, so sums never equal the site total.
type QueryRow struct {
	Query       string
	Page        string
	Clicks      float64
	Impressions float64
	Position    float64 // impression-weighted
}

func QueryRowsFromFacts(facts []model.GscFact) []QueryRow {
	type acc struct {
		row      QueryRow
		weighted float64
	}
	by := map[string]*acc{}
	var order []string
	for _, f := range facts {
		k := f.Query + "\x00" + f.Page
		a := by[k]
		if a == nil {
			a = &acc{row: QueryRow{Query: f.Query, Page: f.Page}}
			by[k] = a
			order = append(order, k)
		}
		a.row.Clicks += f.Clicks
		a.row.Impressions += f.Impressions
		a.weighted += f.Position * f.Impressions
	}
	out := make([]QueryRow, 0, len(order))
	for _, k := range order {
		a := by[k]
		if a.row.Impressions > 0 {
			a.row.Position = a.weighted / a.row.Impressions
		}
		out = append(out, a.row)
	}
	return out
}
