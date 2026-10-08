// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "testing"

func TestAlertsRequireCoverageVolumeAndSustainedDrop(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   AlertInput
		want int
	}{
		{"small baseline", AlertInput{CurrentClicks: 7, PreviousClicks: 10, Covered: true, Sustained: true}, 0},
		{"incomplete", AlertInput{CurrentClicks: 0, PreviousClicks: 300, Sustained: true}, 0},
		{"one week spike", AlertInput{CurrentClicks: 0, PreviousClicks: 300, Covered: true}, 0},
		{"small absolute", AlertInput{CurrentClicks: 75, PreviousClicks: 100, Covered: true, Sustained: true}, 0},
		{"large baseline zero", AlertInput{CurrentClicks: 0, PreviousClicks: 300, Covered: true, Sustained: true}, 1},
		{"sustained", AlertInput{CurrentClicks: 100, PreviousClicks: 300, Covered: true, Sustained: true}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateAlerts(tc.in)
			if len(got) != tc.want {
				t.Fatalf("%#v", got)
			}
			for _, hit := range got {
				if hit.Type != "traffic_drop" || hit.Message == "" {
					t.Fatalf("unsupported alert %#v", hit)
				}
			}
		})
	}
}
