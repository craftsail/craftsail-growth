// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// Week holds one complete Monday–Sunday week. A nil value means the week
// was not fully covered by imported data; it is shown as "—", never 0.
type Week struct {
	From                 string   `json:"from"`
	Through              string   `json:"through"`
	Clicks               *float64 `json:"clicks"`
	Impressions          *float64 `json:"impressions"`
	NonBrandClicks       *float64 `json:"non_brand_clicks"`      // visible queries only
	NonBrandImpressions  *float64 `json:"non_brand_impressions"` // visible queries only
	VisibleClicks        *float64 `json:"visible_clicks"`        // all visible queries, for the non-brand share
	PagesWithImpressions *float64 `json:"pages_with_impressions"`
	NewPages             *float64 `json:"new_pages"` // publish dates set on the Indexing tab
	OrganicSessions      *float64 `json:"organic_sessions"`
	OrganicEngaged       *float64 `json:"organic_engaged"`
	OrganicKeyEvents     *float64 `json:"organic_key_events"`
	AISessions           *float64 `json:"ai_sessions"`
}

// WeeklySearch is the weekly table's data: this week, last week and four
// weeks ago, plus index facts that exist only as a current snapshot.
type WeeklySearch struct {
	Weeks             [3]Week `json:"weeks"`
	Indexed           int64   `json:"indexed"`
	Inspected         int64   `json:"inspected"`
	PublishedMature   int64   `json:"published_mature"`
	IndexedWithinWeek int64   `json:"indexed_within_week"`
}

// weekOffsets are the number of weeks back from "this week" for Weeks[0..2]:
// this week, last week, four weeks ago.
var weekOffsets = [3]int{0, 1, 4}

// WeeklySearch computes the weekly report's table for three complete
// Monday–Sunday weeks ending on or before through. A zero through is
// resolved from searchWindow's finalized-through date; the weekly report
// passes the zero value so it always gets the latest finalized weeks.
func (s *Service) WeeklySearch(ctx context.Context, p *model.Project, through time.Time) (*WeeklySearch, error) {
	property := s.officialProperty(ctx, p, "gsc")
	if through.IsZero() {
		_, t, _, err := s.searchWindow(ctx, p)
		if err != nil {
			return nil, err
		}
		through = t
	}
	idxProperty := s.indexProperty(ctx, p)
	hasPublished, err := s.rows.HasPublished(ctx, p.ID, idxProperty)
	if err != nil {
		return nil, err
	}
	gaProperty := ""
	if strings.TrimSpace(p.GA4Property) != "" {
		gaProperty = s.officialProperty(ctx, p, "ga4")
	}

	out := &WeeklySearch{}
	// Only complete Monday–Sunday weeks, never a partial current week. Anchor
	// on the UTC date so Weekday() matches the same day dateOnly already
	// converted to, regardless of through's own location.
	d := dateOnly(through)
	weekEnd := d.AddDate(0, 0, -int(d.Weekday()))
	for idx, off := range weekOffsets {
		end := weekEnd.AddDate(0, 0, -7*off)
		start := end.AddDate(0, 0, -6)
		w := Week{From: start.Format("2006-01-02"), Through: end.Format("2006-01-02")}

		dc, err := s.grainCoverage(ctx, p.ID, property, "daily", start, end)
		if err != nil {
			return nil, err
		}
		daily, err := s.rows.ListGscDaily(ctx, p.ID, property, start, end)
		if err != nil {
			return nil, err
		}
		sum := sumOfficial(daily, start, end)
		if dc.State == "covered" && sum.rows == 7 {
			clicks, impr := sum.clicks, sum.impr
			w.Clicks, w.Impressions = &clicks, &impr
		}

		qc, err := s.grainCoverage(ctx, p.ID, property, "query", start, end)
		if err != nil {
			return nil, err
		}
		if qc.State == "covered" {
			rows, err := s.rows.ListGscSlice(ctx, p.ID, property, "query", start, end)
			if err != nil {
				return nil, err
			}
			var nbClicks, nbImpr, visible float64
			for _, r := range rows {
				visible += r.Clicks
				if nonbrand(r.Query, p) {
					nbClicks += r.Clicks
					nbImpr += r.Impressions
				}
			}
			w.NonBrandClicks, w.NonBrandImpressions, w.VisibleClicks = &nbClicks, &nbImpr, &visible
		}

		pc, err := s.grainCoverage(ctx, p.ID, property, "page", start, end)
		if err != nil {
			return nil, err
		}
		if pc.State == "covered" {
			n, err := s.rows.PagesWithImpressions(ctx, p.ID, property, start, end)
			if err != nil {
				return nil, err
			}
			v := float64(n)
			w.PagesWithImpressions = &v
		}

		if hasPublished {
			fromUnix := start.Unix()
			toUnix := end.AddDate(0, 0, 1).Add(-time.Second).Unix()
			n, err := s.rows.CountPublished(ctx, p.ID, idxProperty, fromUnix, toUnix)
			if err != nil {
				return nil, err
			}
			v := float64(n)
			w.NewPages = &v
		}

		if gaProperty != "" {
			gc, err := s.reportCoverage(ctx, p.ID, "ga4", gaProperty, "channel", "", start, end)
			if err != nil {
				return nil, err
			}
			if gc.State == "covered" && trustworthyGA(gc.Quality) {
				facts, err := s.rows.ListGaChannelFacts(ctx, p.ID, gaProperty, start, end)
				if err != nil {
					return nil, err
				}
				var sessions, engaged, keyEvents, aiSessions float64
				for _, f := range facts {
					if f.Channel == "Organic Search" {
						sessions += f.Sessions
						engaged += f.Engaged
						keyEvents += f.KeyEvents
					}
					if AISource(f.Source, f.Medium) {
						aiSessions += f.Sessions
					}
				}
				w.OrganicSessions, w.OrganicEngaged, w.OrganicKeyEvents, w.AISessions = &sessions, &engaged, &keyEvents, &aiSessions
			}
		}

		out.Weeks[idx] = w
	}

	inv, err := s.IndexInventory(ctx, p.Slug, "", "", 1, 1)
	if err != nil {
		return nil, err
	}
	out.Indexed, out.Inspected, out.PublishedMature, out.IndexedWithinWeek = inv.Indexed, inv.Inspected, inv.PublishedMature, inv.IndexedWithinWeek
	return out, nil
}
