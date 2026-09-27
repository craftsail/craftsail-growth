// SPDX-License-Identifier: AGPL-3.0-or-later

package chartmath

import "sort"

type DayStat struct {
	Day       string
	OK        int
	Mentioned int
	Failed    int
}

type MentionPoint struct {
	Day    string   `json:"day"`
	Value  *float64 `json:"value"`
	Failed int      `json:"failed"`
}

type EngineDay struct {
	Day       string
	Platform  string
	Mentioned int
	Missed    int
	Failed    int
}

type EngineBar struct {
	Platform  string `json:"platform"`
	Mentioned int    `json:"mentioned"`
	Missed    int    `json:"missed"`
	Failed    int    `json:"failed"`
}

func MentionSeries(days []DayStat) []MentionPoint {
	sorted := append([]DayStat(nil), days...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Day < sorted[j].Day })
	out := make([]MentionPoint, 0, len(sorted))
	for _, day := range sorted {
		point := MentionPoint{Day: day.Day, Failed: day.Failed}
		if day.OK > 0 {
			rate := float64(day.Mentioned) / float64(day.OK)
			point.Value = &rate
		}
		out = append(out, point)
	}
	return out
}

func MentionTakeaway(points []MentionPoint) *TakeawayResult {
	var rates []float64
	for _, point := range points {
		if point.Value != nil {
			rates = append(rates, *point.Value)
		}
	}
	got := Takeaway(rates, true)
	if got.Kind == "" {
		return nil
	}
	return &got
}

type EngineSeries struct {
	Platform string         `json:"platform"`
	Points   []MentionPoint `json:"points"`
}

func EngineSeriesFrom(rows []EngineDay) []EngineSeries {
	byPlatform := map[string][]DayStat{}
	for _, row := range rows {
		ok := row.Mentioned + row.Missed
		byPlatform[row.Platform] = append(byPlatform[row.Platform], DayStat{
			Day: row.Day, OK: ok, Mentioned: row.Mentioned, Failed: row.Failed,
		})
	}
	names := make([]string, 0, len(byPlatform))
	for name := range byPlatform {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]EngineSeries, 0, len(names))
	for _, name := range names {
		out = append(out, EngineSeries{Platform: name, Points: MentionSeries(byPlatform[name])})
	}
	return out
}

func LatestEngineBars(rows []EngineDay) []EngineBar {
	latest := map[string]EngineDay{}
	for _, row := range rows {
		cur, ok := latest[row.Platform]
		if !ok || row.Day >= cur.Day {
			latest[row.Platform] = row
		}
	}
	out := make([]EngineBar, 0, len(latest))
	for _, row := range latest {
		out = append(out, EngineBar{
			Platform: row.Platform, Mentioned: row.Mentioned, Missed: row.Missed, Failed: row.Failed,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Platform < out[j].Platform })
	return out
}
