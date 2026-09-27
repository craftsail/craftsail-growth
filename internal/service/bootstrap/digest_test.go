// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestDigestHomepageFirstNotHighestScore(t *testing.T) {
	home := "https://t.example.com/"
	deep := "https://t.example.com/blog/hot-article"
	d := Digest([]PageText{
		{URL: home, Title: "首页", Text: strings.Repeat("首页正文 ", 50)},
		{URL: deep, Title: "高分页", Text: strings.Repeat("高分页正文 ", 50)},
	}, map[string]float64{home: 1, deep: 99}, "", 14000)
	blocks := strings.Split(d, "## 页面：")
	var nonempty []string
	for _, b := range blocks {
		if strings.TrimSpace(b) != "" {
			nonempty = append(nonempty, b)
		}
	}
	if len(nonempty) == 0 {
		t.Fatal("digest empty")
	}
	if !strings.Contains(nonempty[0], home) {
		t.Fatalf("first block should be homepage, got %q", nonempty[0][:min(80, len(nonempty[0]))])
	}
	if strings.Contains(nonempty[0], deep) {
		t.Fatal("deep page leaked into first block")
	}
}

func TestDigestMaterialsWhenNoPages(t *testing.T) {
	mat := "# 商品\n\n> 提示\n\n这是一段足够长的介绍材料，用来推导品牌事实和问题库，不能只剩模板。"
	d := Digest(nil, nil, mat, 14000)
	if !strings.Contains(d, "唯一依据") || !strings.Contains(d, "介绍材料") {
		t.Fatalf("got %q", d)
	}
}

func TestDigestIgnoresTemplateMaterials(t *testing.T) {
	mat := "# 标题\n\n> 提示\n\n（占位）\n"
	if Digest(nil, nil, mat, 14000) != "" {
		t.Fatal("template skeleton should be empty digest")
	}
}

func TestFilterFakeCompetitors(t *testing.T) {
	got := FilterCompetitors([]map[string]any{
		{"name": "竞品A", "market": "cn", "aliases": []any{"A1"}},
		{"name": "工具A", "market": "cn"},
		{"name": "某某Pro", "market": "cn"},
		{"name": "测试品牌", "market": "cn"},
		{"name": "ChatGPT", "market": "global"},
	}, "测试品牌")
	if len(got) != 2 || got[0].Name != "竞品A" || got[1].Name != "ChatGPT" {
		t.Fatalf("%+v", got)
	}
	if model.CompetitorConfirmed(got[0]) {
		t.Fatal("LLM candidates start unconfirmed")
	}
}

func TestRenderFactsMarksUnconfirmed(t *testing.T) {
	f := false
	md := RenderFacts(BrandFacts{Name: "测试品牌"}, "https://t.example.com", []model.Competitor{
		{Name: "竞品A", Market: "cn", Confirmed: &f},
		{Name: "老牌竞品", Market: "cn"},
	})
	if !strings.Contains(md, "not yet seen in answers") {
		t.Fatal("missing unconfirmed mark")
	}
	var unconf, conf string
	for _, ln := range strings.Split(md, "\n") {
		if strings.Contains(ln, "竞品A") {
			unconf = ln
		}
		if strings.Contains(ln, "老牌竞品") {
			conf = ln
		}
	}
	if !strings.Contains(unconf, "not yet seen in answers") {
		t.Fatalf("unconfirmed line %q", unconf)
	}
	if strings.Contains(conf, "not yet seen in answers") {
		t.Fatalf("legacy competitor should be confirmed: %q", conf)
	}
}

func TestNormalizeQuestions(t *testing.T) {
	qs := NormalizeQuestions([]map[string]any{
		{"id": "q001", "group": "推荐", "market": "cn", "text": "有什么好用的工具？"},
		{"id": "q002", "group": "推荐", "market": "cn", "text": "有什么好用的工具？"},
		{"id": "q101", "group": "推荐", "market": "global", "text": "best tools?"},
		{"group": "nope", "market": "cn", "text": "怎么选方案？"},
	})
	if len(qs) != 3 {
		t.Fatalf("every unique prompt is kept whatever its market: len=%d %+v", len(qs), qs)
	}
	if qs[2].GroupName != "推荐" {
		t.Fatalf("unknown group fallback %q", qs[1].GroupName)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestTemplateQuestionsOneLanguage(t *testing.T) {
	for lang, want := range map[string]string{"zh": "有哪些好用的工具？", "en": "What are the best tools in this category?"} {
		qs := TemplateQuestions("Acme", lang)
		if len(qs) != 8 || qs[0].Text != want {
			t.Fatalf("%s: %d prompts, first %q", lang, len(qs), qs[0].Text)
		}
		for _, q := range qs {
			if q.Market != model.MarketAll {
				t.Fatalf("prompt %s has market %q", q.QID, q.Market)
			}
		}
	}
}
