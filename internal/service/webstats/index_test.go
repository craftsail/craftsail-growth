// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type indexRT struct {
	body string
	url  string
}

func (t *indexRT) RoundTrip(req *http.Request) (*http.Response, error) {
	t.url = req.URL.String()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestFetchSitemapsSubmitted(t *testing.T) {
	rt := &indexRT{body: `{"sitemap":[{"path":"https://ex.com/sitemap.xml","isSitemapsIndex":true,"errors":"1","warnings":2,"contents":[{"submitted":"12"},{"submitted":3}]}]}`}
	c := &Client{HTTP: &http.Client{Transport: rt}}
	rows, err := c.FetchSitemaps("tok", "sc-domain:ex.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Submitted != 15 || rows[0].Errors != 1 || rows[0].Warnings != 2 || !rows[0].IsIndex || rows[0].Property != "sc-domain:ex.com" {
		t.Fatalf("%#v", rows)
	}
	if !strings.Contains(rt.url, "sc-domain%3Aex.com") {
		t.Fatalf("url %s", rt.url)
	}
}

func TestResolveGSCSiteUsesProjectHost(t *testing.T) {
	got := ResolveGSCSite([]string{"sc-domain:acmecli.example", "https://acmecli.example/"}, "https://acmecli.example", "sc-domain:other.example")
	if got != "sc-domain:acmecli.example" {
		t.Fatalf("got %q", got)
	}
	got = ResolveGSCSite(nil, "https://acmecli.example/docs", "")
	if got != "sc-domain:acmecli.example" {
		t.Fatalf("fallback %q", got)
	}
	got = ResolveGSCSite([]string{"https://acmecli.example/"}, "https://acmecli.example", "")
	if got != "https://acmecli.example/" {
		t.Fatalf("prefix %q", got)
	}
}

func TestURLInProperty(t *testing.T) {
	if !urlInProperty("sc-domain:acmecli.example", "https://acmecli.example/pricing") {
		t.Fatal("domain page should match")
	}
	if urlInProperty("sc-domain:acmecli.example", "https://acmeapi.example") {
		t.Fatal("other site must not match")
	}
	if !urlInProperty("https://www.example.com/blog/", "https://www.example.com/blog/post") {
		t.Fatal("url prefix should match")
	}
	if urlInProperty("https://www.example.com/blog/", "https://www.example.com/other") {
		t.Fatal("other path must not match")
	}
}

func TestInspectURLVerdict(t *testing.T) {
	rt := &indexRT{body: `{"inspectionResult":{"indexStatusResult":{"verdict":"PASS","coverageState":"已提交并编入索引","lastCrawlTime":"2026-09-01T00:00:00Z"}}}`}
	c := &Client{HTTP: &http.Client{Transport: rt}}
	row, err := c.InspectURL("tok", "sc-domain:ex.com", "https://ex.com/a")
	if err != nil {
		t.Fatal(err)
	}
	if row.Verdict != "PASS" || row.CoverageState != "已提交并编入索引" || row.URL != "https://ex.com/a" {
		t.Fatalf("%#v", row)
	}
}
