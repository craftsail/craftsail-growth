// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"net/http"
	"testing"
)

func TestFetchGAFactsSession(t *testing.T) {
	var calls int
	rt := roundTrip(func(req *http.Request) (int, string) {
		calls++
		if calls == 1 {
			return 200, `{"rows":[{"dimensionValues":[{"value":"20260901"},{"value":"chatgpt.com"},{"value":"referral"},{"value":"(not set)"},{"value":"Organic Search"},{"value":"/blog"},{"value":"United States"},{"value":"desktop"}],"metricValues":[{"value":"4"},{"value":"1"},{"value":"2"},{"value":"1"},{"value":"3"},{"value":"0"},{"value":"0"},{"value":"0.5"},{"value":"0.2"},{"value":"12"}]}],"rowCount":1}`
		}
		return 200, `{"rows":[]}`
	})
	c := &Client{HTTP: &http.Client{Transport: rt}, GALimit: 1}
	rows, err := c.FetchGAFacts(context.Background(), "tok", "properties/123", "session", "2026-09-01", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 || len(rows) != 1 || rows[0].Report != "session" || rows[0].Source != "chatgpt.com" || rows[0].Sessions != 4 {
		t.Fatalf("calls=%d %#v", calls, rows)
	}
	if rows[0].Day.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("day %s", rows[0].Day)
	}
}
