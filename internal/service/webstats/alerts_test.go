// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "testing"

func TestEvaluateAlerts(t *testing.T) {
	fires := EvaluateAlerts(AlertInput{ClicksDeltaPct: -25, CurrentClicks: 40, PositionDelta: -3})
	types := map[string]bool{}
	for _, f := range fires {
		types[f.Type] = true
		if f.Message == "" {
			t.Fatal("empty message")
		}
	}
	if !types["traffic_drop"] || !types["position_change"] || len(fires) != 2 {
		t.Fatalf("%+v", fires)
	}
	quiet := EvaluateAlerts(AlertInput{ClicksDeltaPct: -5, CurrentClicks: 40, PositionDelta: -0.2})
	if len(quiet) != 0 {
		t.Fatalf("should be quiet: %+v", quiet)
	}
}

func TestAlertOpsNeedMeasuredPreviousWindow(t *testing.T) {
	down, pos := -30.0, -2.5
	p := Period{Clicks: 20, ClicksDelta: &down, PositionDelta: &pos, HasPrevious: true, Measured: true}
	if ops := alertOps(p); len(ops) != 2 || ops[0].Type != "traffic_drop" {
		t.Fatalf("%+v", ops)
	}
	p.Measured = false
	if ops := alertOps(p); len(ops) != 0 {
		t.Fatal("unmeasured periods must not raise alerts")
	}
}
