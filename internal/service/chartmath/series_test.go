// SPDX-License-Identifier: AGPL-3.0-or-later

package chartmath

import "testing"

func TestMentionSeriesSkipsFailedAsRate(t *testing.T) {
	points := MentionSeries([]DayStat{
		{Day: "2026-09-02", Failed: 2},
		{Day: "2026-09-01", OK: 4, Mentioned: 1},
		{Day: "2026-09-03", OK: 2, Mentioned: 2, Failed: 1},
	})
	if len(points) != 3 || points[0].Day != "2026-09-01" || points[0].Value == nil || *points[0].Value != 0.25 {
		t.Fatalf("%#v", points)
	}
	if points[1].Value != nil || points[1].Failed != 2 {
		t.Fatalf("failed-only day must not become zero: %#v", points[1])
	}
	if points[2].Value == nil || *points[2].Value != 1 || points[2].Failed != 1 {
		t.Fatalf("%#v", points[2])
	}
}

func TestMentionTakeawayNeedsSevenRates(t *testing.T) {
	var days []DayStat
	for i := 1; i <= 6; i++ {
		days = append(days, DayStat{Day: "2026-09-0" + string(rune('0'+i)), OK: 2, Mentioned: 1})
	}
	if MentionTakeaway(MentionSeries(days)) != nil {
		t.Fatal("6 days must not write a takeaway")
	}
	days = append(days, DayStat{Day: "2026-09-07", OK: 2, Mentioned: 2})
	got := MentionTakeaway(MentionSeries(days))
	if got == nil || got.Kind == "" || got.Days != 7 {
		t.Fatalf("%+v", got)
	}
}

func TestLatestEngineBars(t *testing.T) {
	bars := LatestEngineBars([]EngineDay{
		{Day: "2026-09-01", Platform: "glm", Mentioned: 1, Missed: 3},
		{Day: "2026-09-02", Platform: "glm", Mentioned: 2, Missed: 1, Failed: 1},
		{Day: "2026-09-02", Platform: "kimi", Mentioned: 0, Missed: 2},
	})
	if len(bars) != 2 || bars[0].Platform != "glm" || bars[0].Mentioned != 2 || bars[0].Missed != 1 {
		t.Fatalf("%#v", bars)
	}
	if bars[1].Platform != "kimi" || bars[1].Missed != 2 {
		t.Fatalf("%#v", bars)
	}
}
