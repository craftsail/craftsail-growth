// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"time"
)

// ObservePages reads a frozen resource/date scope without starting syncs.
func (s *Service) ObservePages(ctx context.Context, slug, property, from, through, metric string, urls []string) (model.Measurement, string, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return model.Measurement{}, "", err
	}
	active := s.officialProperty(ctx, p, "gsc")
	if p.GscSite != "" {
		active = configuredProperty(p, "gsc")
	}
	if property == "" {
		property = active
	}
	out := model.Measurement{Reason: "coverage", Signature: property}
	if active != property {
		return model.Measurement{Reason: "property_changed"}, property, nil
	}
	a, err := time.Parse("2006-01-02", from)
	if err != nil {
		return out, property, err
	}
	b, err := time.Parse("2006-01-02", through)
	if err != nil {
		return out, property, err
	}
	c, err := s.grainCoverage(ctx, p.ID, property, "page", a, b)
	if err != nil {
		return out, property, err
	}
	if !trustedSearch(c) {
		return out, property, nil
	}
	rows, err := s.rows.ListGscSlice(ctx, p.ID, property, "page", a, b)
	if err != nil {
		return out, property, err
	}
	wanted := map[string]bool{}
	for _, u := range urls {
		wanted[u] = true
	}
	out.Valid = true
	out.Reason = ""
	out.Signature = model.RowKey(property, c.Quality.Aggregations[0])
	out.Count = len(urls)
	for _, r := range rows {
		if !wanted[r.Page] {
			continue
		}
		out.Exposure += r.Impressions
		if metric == "impressions" {
			out.Value += r.Impressions
		} else {
			out.Value += r.Clicks
		}
	}
	return out, property, nil
}
