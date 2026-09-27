// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

func codesOf(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Code]++
	}
	return m
}

func TestPageSEOChecks(t *testing.T) {
	p := crawl.PageDoc{URL: "https://example.com/a", Status: 200, Title: "", MetaDescription: "",
		ImagesMissingAlt: 2, ResponseTimeMs: 4200, Redirects: []string{"https://example.com/x", "https://example.com/a"}}
	got := codesOf(PageSEO(p))
	for _, c := range []string{"MISSING_TITLE", "MISSING_META_DESCRIPTION", "IMAGES_MISSING_ALT", "SLOW_RESPONSE", "REDIRECT_CHAIN"} {
		if got[c] != 1 {
			t.Fatalf("missing %s in %v", c, got)
		}
	}
}

func TestPageSEOSkipsNon200(t *testing.T) {
	if got := PageSEO(crawl.PageDoc{URL: "https://example.com/a", Status: 404}); len(got) != 0 {
		t.Fatalf("status errors are reported by ScorePage, got %v", got)
	}
}

func TestTitleLengthUsesCJKWidth(t *testing.T) {
	cjk := crawl.PageDoc{URL: "https://example.com/a", Status: 200, MetaDescription: "d", Title: "这是一个非常非常长的中文标题用来测试截断宽度是否按中文计算的规则是否生效呢"}
	if codesOf(PageSEO(cjk))["TITLE_TOO_LONG"] != 1 {
		t.Fatal("long CJK title should be flagged")
	}
	latin := crawl.PageDoc{URL: "https://example.com/a", Status: 200, MetaDescription: "d", Title: "A short English title"}
	if codesOf(PageSEO(latin))["TITLE_TOO_LONG"] != 0 {
		t.Fatal("short title flagged")
	}
}

func TestSiteSEOChecks(t *testing.T) {
	pages := []crawl.PageDoc{
		{URL: "https://example.com/", Status: 200, Title: "Home", MetaDescription: "same", OutLinks: []string{"https://example.com/a", "https://example.com/gone"}},
		{URL: "https://example.com/a", Status: 200, Title: "Home", MetaDescription: "same"},
		{URL: "https://example.com/gone", Status: 404},
		{URL: "https://example.com/lonely", Status: 200, Title: "Lonely"},
	}
	got := SiteSEO(pages)
	c := codesOf(got)
	if c["DUPLICATE_TITLE"] != 1 || c["DUPLICATE_META_DESCRIPTION"] != 1 {
		t.Fatalf("duplicates: %v", c)
	}
	if c["BROKEN_INTERNAL_LINK"] != 1 {
		t.Fatalf("broken link: %v", c)
	}
	if c["ORPHAN_PAGE"] != 1 {
		t.Fatalf("orphan: %v", c)
	}
	for _, f := range got {
		if f.Code == "BROKEN_INTERNAL_LINK" && (f.URL != "https://example.com/" || f.Detail["target"] != "https://example.com/gone") {
			t.Fatalf("broken link finding = %+v", f)
		}
		if f.Code == "ORPHAN_PAGE" && f.URL != "https://example.com/lonely" {
			t.Fatalf("orphan finding = %+v", f)
		}
	}
}

func TestOrphanNeedsLinkData(t *testing.T) {
	pages := []crawl.PageDoc{
		{URL: "https://example.com/", Status: 200, Title: "Home"},
		{URL: "https://example.com/a", Status: 200, Title: "A"},
		{URL: "https://example.com/b", Status: 200, Title: "B"},
	}
	if c := codesOf(SiteSEO(pages)); c["ORPHAN_PAGE"] != 0 {
		t.Fatalf("pages crawled without link data must not be reported as orphans: %v", c)
	}
}
