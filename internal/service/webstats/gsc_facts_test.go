// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fnRT struct {
	fn func(*http.Request) (int, string)
}

func (f fnRT) RoundTrip(req *http.Request) (*http.Response, error) {
	code, body := f.fn(req)
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func roundTrip(fn func(*http.Request) (int, string)) http.RoundTripper {
	return fnRT{fn: fn}
}

func TestFetchGSCFactsPages(t *testing.T) {
	var calls int
	var body string
	rt := roundTrip(func(req *http.Request) (int, string) {
		calls++
		b, _ := io.ReadAll(req.Body)
		body = string(b)
		if calls == 1 {
			return 200, `{"rows":[{"keys":["2026-09-01","geo tool","https://ex.com/"],"clicks":3,"impressions":10,"ctr":0.3,"position":4.2}]}`
		}
		return 200, `{"rows":[]}`
	})
	c := &Client{HTTP: &http.Client{Transport: rt}, RowLimit: 1}
	rows, err := c.FetchGSCFacts(context.Background(), "tok", "sc-domain:ex.com", "query_page", "web", "2026-09-01", "2026-09-07")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(rows) != 1 || rows[0].Slice != "query_page" || rows[0].SearchType != "web" || rows[0].Clicks != 3 || rows[0].Country != "" {
		t.Fatalf("calls=%d rows=%#v", calls, rows)
	}
	if !strings.Contains(body, `"dataState":"all"`) || !strings.Contains(body, `"type":"web"`) {
		t.Fatalf("body %s", body)
	}
}
