// SPDX-License-Identifier: AGPL-3.0-or-later

package htmlx

import "testing"

func TestNormalizeStripsTracking(t *testing.T) {
	u := Normalize("https://example.com/", "/p?utm_source=x&fbclid=y&id=42&gclid=z")
	if u != "https://example.com/p?id=42" {
		t.Fatalf("got %q", u)
	}
}

func TestNormalizeStripsRefSpm(t *testing.T) {
	u := Normalize("https://example.com/", "https://example.com/a?ref=nav&spm=1.2.3&lang=zh")
	if u != "https://example.com/a?lang=zh" {
		t.Fatalf("got %q", u)
	}
}

func TestNormalizeDropsHash(t *testing.T) {
	u := Normalize("https://example.com/", "/about#team")
	if u != "https://example.com/about" {
		t.Fatalf("got %q", u)
	}
}

func TestSameSiteWWW(t *testing.T) {
	if !SameSite("https://www.example.com/", "https://example.com/a") {
		t.Fatal("www vs apex")
	}
	if SameSite("https://example.com/", "https://other.com/") {
		t.Fatal("different hosts")
	}
}

func TestMainTextSingleArticle(t *testing.T) {
	text := MainText("<main><nav>menu</nav><article>the post body</article></main>")
	if text != "the post body" {
		t.Fatalf("got %q", text)
	}
}

func TestMainTextMultipleArticlesTakesMain(t *testing.T) {
	html := "<main><article>intro section</article>" +
		"<article>steps: 1 2 3</article>" +
		"<article>faq answers</article></main>"
	text := MainText(html)
	for _, piece := range []string{"intro section", "steps: 1 2 3", "faq answers"} {
		if !contains(text, piece) {
			t.Fatalf("missing %q in %q", piece, text)
		}
	}
}

func TestWordCountCJK(t *testing.T) {
	// 16 CJK runes → 16/1.6 = 10
	n := WordCount("这是一段中文测试文字啊啊啊啊啊啊")
	if n != 10 {
		t.Fatalf("word_count=%d", n)
	}
}

func TestPageLanguage(t *testing.T) {
	ja := "これはテストページです。私たちは最高のサービスを提供します。詳細についてはお問い合わせください。"
	ja = ja + ja + ja
	if PageLanguage(ja, "") != "ja" {
		t.Fatalf("ja got %s", PageLanguage(ja, ""))
	}
	zh := "这是一个中文页面，用来测试语言检测是否会把纯中文误判成日文。"
	zh = zh + zh + zh
	if PageLanguage(zh, "") != "zh" {
		t.Fatalf("zh got %s", PageLanguage(zh, ""))
	}
	en := "This is an English page about our product and services for testing."
	en = en + en + en
	if PageLanguage(en, "") != "en" {
		t.Fatalf("en got %s", PageLanguage(en, ""))
	}
}

func TestJSONLDTypesGraph(t *testing.T) {
	html := `<script type="application/ld+json">{"@graph":[{"@type":"Organization"},{"@type":"WebSite"}]}</script>`
	types := JSONLDTypes(JSONLD(html))
	if !contains(join(types), "Organization") || !contains(join(types), "WebSite") {
		t.Fatalf("%v", types)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}

func join(ss []string) string {
	out := ""
	for _, s := range ss {
		out += s + ","
	}
	return out
}
