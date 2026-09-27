// SPDX-License-Identifier: AGPL-3.0-or-later

package crawl

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
)

func TestRankPrefersHomeAndPriority(t *testing.T) {
	root := "https://example.com"
	got := Rank([]string{
		"https://example.com/deep/nested/page",
		"https://example.com/pricing",
		"https://cdn.example.com/about",
		"https://example.com/file.pdf",
	}, root)
	if got[0] != root && got[0] != "https://example.com" {
		t.Fatalf("home first: %v", got)
	}
	var idxPrice, idxDeep int
	for i, u := range got {
		if u == "https://example.com/pricing" {
			idxPrice = i
		}
		if u == "https://example.com/deep/nested/page" {
			idxDeep = i
		}
	}
	if idxPrice > idxDeep {
		t.Fatalf("pricing should outrank deep page: %v", got)
	}
	for _, u := range got {
		if u == "https://example.com/file.pdf" {
			t.Fatal("pdf should be skipped")
		}
	}
}

func TestFetchable(t *testing.T) {
	if httputil.Fetchable("https://x.com/a.zip") {
		t.Fatal("zip")
	}
	if !httputil.Fetchable("https://x.com/about") {
		t.Fatal("about")
	}
}
