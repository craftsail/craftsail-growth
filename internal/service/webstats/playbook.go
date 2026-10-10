// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

// PlaybookSearch is what the playbook needs from search data. Unknown
// values stay nil; the playbook shows them as "no data", never as zero.
type PlaybookSearch struct {
	Established            bool             `json:"established"`
	NewPages               []model.IndexURL `json:"-"`
	Indexed                int64            `json:"indexed"`
	IndexedWithImpressions int64            `json:"indexed_with_impressions"`
	LastIndexSuccess       *int64           `json:"last_index_success"`
	// VisibleShare is clicks on visible queries / all clicks for the last
	// five complete Monday–Sunday weeks, oldest first.
	VisibleShare  []*float64 `json:"visible_share"`
	GAThresholded *bool      `json:"ga_thresholded"`
}

func (s *Service) PlaybookSearch(ctx context.Context, p *model.Project) (*PlaybookSearch, error) {
	out := &PlaybookSearch{}
	now := s.now()
	idx, err := s.rows.PlaybookIndex(ctx, p.ID, s.indexProperty(ctx, p), now.AddDate(0, 0, -28).Unix())
	if err != nil {
		return nil, err
	}
	out.NewPages, out.Indexed, out.IndexedWithImpressions, out.LastIndexSuccess = idx.NewPages, idx.Indexed, idx.IndexedWithImpressions, idx.LastSuccess
	obs, _, err := s.searchObservation(ctx, p)
	if err != nil {
		return nil, err
	}
	out.Established = obs.Mode == "established"
	_, through, property, err := s.searchWindow(ctx, p)
	if err != nil {
		return nil, err
	}
	if out.VisibleShare, err = s.visibleShare(ctx, p.ID, property, through); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.GA4Property) != "" {
		if ga := s.officialProperty(ctx, p, "ga4"); ga != "" {
			cov, err := s.reportCoverage(ctx, p.ID, "ga4", ga, "daily", "", through.AddDate(0, 0, -27), through)
			if err != nil {
				return nil, err
			}
			if cov.State == "covered" {
				t := cov.Quality.Thresholded
				out.GAThresholded = &t
			}
		}
	}
	return out, nil
}

// visibleShare never fills gaps: a week without full daily and query
// coverage, or with no clicks, is nil.
func (s *Service) visibleShare(ctx context.Context, projectID uint64, property string, through time.Time) ([]*float64, error) {
	d := dateOnly(through)
	weekEnd := d.AddDate(0, 0, -int(d.Weekday()))
	from := weekEnd.AddDate(0, 0, -34)
	daily, err := s.rows.ListGscDaily(ctx, projectID, property, from, weekEnd)
	if err != nil {
		return nil, err
	}
	out := make([]*float64, 0, 5)
	for i := 4; i >= 0; i-- {
		end := weekEnd.AddDate(0, 0, -7*i)
		start := end.AddDate(0, 0, -6)
		dc, err := s.grainCoverage(ctx, projectID, property, "daily", start, end)
		if err != nil {
			return nil, err
		}
		qc, err := s.grainCoverage(ctx, projectID, property, "query", start, end)
		if err != nil {
			return nil, err
		}
		total := sumOfficial(daily, start, end).clicks
		if dc.State != "covered" || qc.State != "covered" || total <= 0 {
			out = append(out, nil)
			continue
		}
		visible, err := s.rows.VisibleClicks(ctx, projectID, property, start, end)
		if err != nil {
			return nil, err
		}
		share := visible / total
		out = append(out, &share)
	}
	return out, nil
}
