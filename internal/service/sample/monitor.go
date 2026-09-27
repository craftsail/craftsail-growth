// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"

	"github.com/craftsail/craftsail-growth/internal/service/chartmath"
)

type MonitorCharts struct {
	Mention    []chartmath.MentionPoint  `json:"mention"`
	Takeaway   *chartmath.TakeawayResult `json:"takeaway"`
	Engines    []chartmath.EngineBar     `json:"engines"`
	Series     []chartmath.EngineSeries  `json:"series"`
	TargetRate float64                   `json:"target_rate"`
}

func (s *Service) MonitorCharts(ctx context.Context, slug string) (*MonitorCharts, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	rows, err := s.samples.AggregateDays(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	byDay := map[string]*chartmath.DayStat{}
	var engines []chartmath.EngineDay
	for _, row := range rows {
		day := byDay[row.Day]
		if day == nil {
			day = &chartmath.DayStat{Day: row.Day}
			byDay[row.Day] = day
		}
		day.OK += row.OK
		day.Mentioned += row.Mentioned
		day.Failed += row.Failed
		missed := row.OK - row.Mentioned
		if missed < 0 {
			missed = 0
		}
		engines = append(engines, chartmath.EngineDay{
			Day: row.Day, Platform: row.Platform, Mentioned: row.Mentioned, Missed: missed, Failed: row.Failed,
		})
	}
	days := make([]chartmath.DayStat, 0, len(byDay))
	for _, day := range byDay {
		days = append(days, *day)
	}
	points := chartmath.MentionSeries(days)
	target := p.Targets.MentionRate
	if target == 0 {
		target = 0.5
	}
	return &MonitorCharts{
		Mention:    points,
		Takeaway:   chartmath.MentionTakeaway(points),
		Engines:    chartmath.LatestEngineBars(engines),
		Series:     chartmath.EngineSeriesFrom(engines),
		TargetRate: target,
	}, nil
}
