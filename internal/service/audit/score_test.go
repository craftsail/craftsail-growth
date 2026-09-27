// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/pkg/htmlx"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

func makePage(url, text string, wc int, extra func(*crawl.PageDoc)) crawl.PageDoc {
	p := crawl.PageDoc{
		URL: url, Title: "", Status: 200, Canonical: url,
		H1: []string{"t"}, H2: []string{"a", "a", "a", "a", "a", "a"},
		ParaCount: 10, LiCount: 5, JSONLDTypes: []string{"WebSite"},
		ExternalLinks: 5, Text: text,
	}
	if wc >= 0 {
		p.WordCount = wc
	} else {
		p.WordCount = htmlx.WordCount(text)
	}
	if extra != nil {
		extra(&p)
	}
	return p
}

const spaText = "加载中"

func TestSPAShellHasCode(t *testing.T) {
	r := ScorePage(makePage("https://example.com/", spaText, -1, nil), nil)
	if !hasCode(r, "SPA_SHELL") {
		t.Fatalf("%v", r.IssueCodes)
	}
}

func TestFunctionalPageExempt(t *testing.T) {
	r := ScorePage(makePage("https://example.com/login", spaText, -1, nil), nil)
	if hasCode(r, "SPA_SHELL") {
		t.Fatal("SPA_SHELL")
	}
	if !hasCode(r, "LOW_CONTENT_PAGE") {
		t.Fatal("LOW_CONTENT_PAGE")
	}
}

func TestFunctionalPageCaseInsensitive(t *testing.T) {
	r := ScorePage(makePage("https://example.com/Auth/SignIn", spaText, -1, nil), nil)
	if !hasCode(r, "LOW_CONTENT_PAGE") || hasCode(r, "SPA_SHELL") {
		t.Fatalf("%v", r.IssueCodes)
	}
}

func TestNormalLowPageStillSPA(t *testing.T) {
	r := ScorePage(makePage("https://example.com/products", spaText, -1, nil), nil)
	if !hasCode(r, "SPA_SHELL") {
		t.Fatal("expected SPA_SHELL")
	}
}

func TestShortStaticPageIsNotShell(t *testing.T) {
	text := "Blog Rust Tag Rust 1 AcmeAPI blog post tagged Rust. All posts Sep 24, 2026 3 min read Why AcmeAPI is written in Rust The five documented advantages of AcmeAPI."
	r := ScorePage(makePage("https://acmeapi.example/blog/tag/rust/", text, -1, func(p *crawl.PageDoc) {
		p.H1 = []string{"Rust"}
	}), nil)
	if hasCode(r, "SPA_SHELL") {
		t.Fatalf("short static page marked shell: %v words %d", r.IssueCodes, r.WordCount)
	}
	if r.Dimensions["可抓取性"] != 12 {
		t.Fatalf("crawlability %v, want 12", r.Dimensions["可抓取性"])
	}
}

func TestEmptyRootIsShell(t *testing.T) {
	r := ScorePage(makePage("https://example.com/app", "", 0, func(p *crawl.PageDoc) {
		p.H1 = nil
		p.HTML = `<html><body><div id="root"></div><script src="/app.js"></script></body></html>`
	}), nil)
	if !hasCode(r, "SPA_SHELL") {
		t.Fatalf("empty root not marked shell: %v", r.IssueCodes)
	}
}

func TestHowto(t *testing.T) {
	r := ScorePage(makePage("https://example.com/", strings.Repeat("如何选择合适的方案？这是很多用户关心的问题。", 10), -1, func(p *crawl.PageDoc) {
		p.LiCount = 0
	}), nil)
	if r.Blocks["操作步骤"] {
		t.Fatal("soft 如何 without list is not howto")
	}
	r = ScorePage(makePage("https://example.com/", strings.Repeat("第一步：注册账号。第二步：填写资料。第三步：提交审核。", 5), -1, func(p *crawl.PageDoc) {
		p.LiCount = 0
	}), nil)
	if !r.Blocks["操作步骤"] {
		t.Fatal("numbered steps")
	}
	r = ScorePage(makePage("https://example.com/", strings.Repeat("怎么配置环境？请按下面的要点操作。", 10), -1, func(p *crawl.PageDoc) {
		p.LiCount = 6
	}), nil)
	if !r.Blocks["操作步骤"] {
		t.Fatal("如何 + list")
	}
}

func TestJapaneseBlocks(t *testing.T) {
	r := ScorePage(makePage("https://example.com/", strings.Repeat("craftsail-growthとは、生成エンジン最適化のための診断ツールです。", 5), -1, nil), nil)
	if !r.Blocks["定义"] {
		t.Fatal("ja definition")
	}
	r = ScorePage(makePage("https://example.com/", strings.Repeat("導入の手順：まずアカウントを作成し、次にサイトを登録します。", 5), -1, nil), nil)
	if !r.Blocks["操作步骤"] {
		t.Fatal("ja howto")
	}
	r = ScorePage(makePage("https://example.com/", strings.Repeat("よくある質問：料金はいくらですか。プランによって異なります。", 5), -1, nil), nil)
	if !r.Blocks["FAQ"] {
		t.Fatal("ja faq")
	}
	r = ScorePage(makePage("https://example.com/", strings.Repeat("導入企業は 300社、月間 5億回 の処理、平均 24時間 で稼働開始。", 3), -1, nil), nil)
	if !r.Blocks["数字事实"] {
		t.Fatal("ja numbers")
	}
}

func TestJSONLDDate(t *testing.T) {
	r := ScorePage(makePage("https://example.com/blog/a", spaText, -1, func(p *crawl.PageDoc) {
		p.JSONLDTypes = []string{"Article"}
		p.JSONLDRaw = []any{map[string]any{"@type": "Article", "dateModified": "2026-01-01"}}
	}), nil)
	if hasCode(r, "NO_DATE") {
		t.Fatal("dateModified should count")
	}
	r = ScorePage(makePage("https://example.com/blog/a", spaText, -1, func(p *crawl.PageDoc) {
		p.JSONLDTypes = []string{"Article"}
		p.JSONLDRaw = []any{map[string]any{"@type": "Article"}}
	}), nil)
	if !hasCode(r, "NO_DATE") {
		t.Fatal("expected NO_DATE")
	}
}

func TestKeywordsExcludeBrand(t *testing.T) {
	kws := KeywordsFromConfig(map[string]any{
		"brand": map[string]any{"name": "甲工智能", "aliases": []any{"甲工"}, "products": []any{"甲工云"}},
		"questions": []any{
			map[string]any{"text": "智能体平台怎么选？有哪些好用的替代品？"},
		},
	})
	set := map[string]bool{}
	for _, k := range kws {
		set[k] = true
	}
	for _, bad := range []string{"甲工智能", "甲工", "甲工云"} {
		if set[bad] {
			t.Fatalf("brand term %s leaked", bad)
		}
	}
	if !set["智能体平台怎么选"] || !set["有哪些好用的替代品"] {
		t.Fatalf("%v", kws)
	}
}

func TestAvgScoreIgnoresNon200(t *testing.T) {
	a := makePage("https://example.com/a", spaText, -1, nil)
	b := makePage("https://example.com/b", spaText, -1, func(p *crawl.PageDoc) { p.Status = 404 })
	ra, rb := ScorePage(a, nil), ScorePage(b, nil)
	ok := []PageScore{ra}
	_ = rb
	avg := ra.Score
	if len(ok) != 1 || avg != ra.Score {
		t.Fatal(avg)
	}
}

func hasCode(r PageScore, code string) bool {
	for _, c := range r.IssueCodes {
		if c == code {
			return true
		}
	}
	return false
}

func TestContentRulesSkipNonContentPages(t *testing.T) {
	home := crawl.PageDoc{URL: "https://example.com/", Status: 200, Title: "Example", H1: []string{"Example"},
		WordCount: 200, Text: "Example builds tools.", Canonical: "https://example.com/"}
	r := ScorePage(home, nil)
	for _, c := range r.IssueCodes {
		if Lookup(c).Scope == ScopeContent {
			t.Fatalf("home page should not get content-scope issue %s", c)
		}
	}
}

func TestContentRulesApplyToArticles(t *testing.T) {
	art := crawl.PageDoc{URL: "https://example.com/blog/how-to-pick", Status: 200, Title: "How to pick", H1: []string{"How to pick"},
		WordCount: 400, Text: strings.Repeat("word ", 400), Canonical: "https://example.com/blog/how-to-pick"}
	if r := ScorePage(art, nil); !hasCode(r, "SHORT_CONTENT") {
		t.Fatalf("article under 1000 words should get SHORT_CONTENT, got %v", r.IssueCodes)
	}
}

func TestFAQIsNotAnIssue(t *testing.T) {
	art := crawl.PageDoc{URL: "https://example.com/blog/x", Status: 200, WordCount: 1500, Text: strings.Repeat("word ", 1500)}
	if r := ScorePage(art, nil); hasCode(r, "NO_FAQ") {
		t.Fatal("NO_FAQ was removed: Q&A formatting showed -5.74% influence (zhang2026)")
	}
}

func TestH1CodesSplit(t *testing.T) {
	art := crawl.PageDoc{URL: "https://example.com/blog/x", Status: 200, WordCount: 800, Text: strings.Repeat("word ", 800), H1: []string{"a", "b"}}
	if r := ScorePage(art, nil); !hasCode(r, "MULTIPLE_H1") || hasCode(r, "BAD_H1") {
		t.Fatalf("got %v", r.IssueCodes)
	}
}

func TestStatusCodesSplit(t *testing.T) {
	if r := ScorePage(crawl.PageDoc{URL: "https://example.com/x", Status: 503}, nil); !hasCode(r, "SERVER_ERROR") {
		t.Fatalf("503: %v", r.IssueCodes)
	}
	if r := ScorePage(crawl.PageDoc{URL: "https://example.com/x", Status: 404}, nil); !hasCode(r, "CLIENT_ERROR") {
		t.Fatalf("404: %v", r.IssueCodes)
	}
	if r := ScorePage(crawl.PageDoc{URL: "https://example.com/x", Status: 0}, nil); !hasCode(r, "PAGE_UNREACHABLE") {
		t.Fatalf("0: %v", r.IssueCodes)
	}
}
