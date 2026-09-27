// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func testCfg() Cfg {
	return Cfg{
		BrandName:   "AIGCLINK定制家",
		Aliases:     []string{"定制家"},
		Site:        "https://aigclink.example.com",
		Competitors: []Comp{{Name: "竞品X"}},
	}
}

func TestCJKAliasSubstringStillMatches(t *testing.T) {
	r := AnalyzeAnswer("这个定制家很好用，推荐试试", testCfg(), nil)
	if !r.BrandMentioned || r.NeedsReview {
		t.Fatalf("%+v", r)
	}
}

func TestNormalHitNotOverExcluded(t *testing.T) {
	r := AnalyzeAnswer("AIGCLINK定制家很好用", testCfg(), nil)
	if !r.BrandMentioned || r.NeedsReview {
		t.Fatalf("%+v", r)
	}
}

func TestNegatedHitNotCountedAndFlagged(t *testing.T) {
	r := AnalyzeAnswer("这不是全屋定制家居类工具", testCfg(), nil)
	if r.BrandMentioned {
		t.Fatal("negated should not count")
	}
	if !r.NeedsReview {
		t.Fatal("needs_review")
	}
}

func TestLatinAliasRequiresBoundary(t *testing.T) {
	cfg := Cfg{BrandName: "灵眸", Aliases: []string{"AIGC"}, Site: "https://x.example.com"}
	if AnalyzeAnswer("AIGCLINK很好用", cfg, nil).BrandMentioned {
		t.Fatal("AIGC should not match AIGCLINK")
	}
	if !AnalyzeAnswer("AIGC 很好用", cfg, nil).BrandMentioned {
		t.Fatal("AIGC with space")
	}
	if !AnalyzeAnswer("推荐AIGC，挺好", cfg, nil).BrandMentioned {
		t.Fatal("AIGC after CJK")
	}
}

func TestBrandInQuestion(t *testing.T) {
	cfg := testCfg()
	if !BrandInQuestion("AIGCLINK定制家是什么", cfg) {
		t.Fatal("name in question")
	}
	if BrandInQuestion("有什么好用的工具？", cfg) {
		t.Fatal("generic question")
	}
}

func TestMarketOf(t *testing.T) {
	if MarketOf("deepssek") != "unknown" {
		t.Fatal("typo")
	}
	if MarketOf("deepseek") != "cn" || MarketOf("perplexity") != "global" || MarketOf("chatgpt") != "global" {
		t.Fatal("known")
	}
}

func TestBaseForOfficialDefault(t *testing.T) {
	p, ok := Lookup("deepseek")
	if !ok {
		t.Fatal("missing deepseek")
	}
	t.Setenv("DEEPSEEK_BASE", "")
	if BaseForEnv(p, func(string) string { return "" }) != "https://api.deepseek.com/v1" {
		t.Fatalf("%s", BaseForEnv(p, func(string) string { return "" }))
	}
}

func TestBaseForCustomRelay(t *testing.T) {
	p, _ := Lookup("deepseek")
	got := BaseForEnv(p, func(k string) string {
		if k == "DEEPSEEK_BASE" {
			return "relay.example.com/v1"
		}
		return ""
	})
	if got != "https://relay.example.com/v1" {
		t.Fatalf("%s", got)
	}
}

func TestBaseForHostOnlyInheritsOfficialPath(t *testing.T) {
	p, _ := Lookup("openai")
	got := NormalizeBase("http://203.0.113.10", p.Base)
	if got != "http://203.0.113.10/v1" {
		t.Fatalf("%s", got)
	}
	got = NormalizeBase("http://203.0.113.10/", p.Base)
	if got != "http://203.0.113.10/v1" {
		t.Fatalf("slash %s", got)
	}
}

func TestBaseForCustomPathKept(t *testing.T) {
	p, _ := Lookup("openai")
	got := NormalizeBase("http://relay.example.com/openai", p.Base)
	if got != "http://relay.example.com/openai" {
		t.Fatalf("%s", got)
	}
}

func TestBaseForOfficialURLClearsToOfficial(t *testing.T) {
	p, _ := Lookup("openai")
	got := NormalizeBase("https://api.openai.com/v1/", p.Base)
	if got != strings.TrimRight(p.Base, "/") {
		t.Fatalf("%s", got)
	}
}

func TestBrandInQuestionMatchesModel(t *testing.T) {
	cfg := Cfg{BrandName: "acmecli", Aliases: []string{"示例命令行"}, Site: "https://acmecli.example"}
	for _, q := range []string{"示例命令行 靠谱吗", "best cli tools", "acmecli.example 文档", "acmecli vs x"} {
		want := model.IsPromptBranded(q, cfg.BrandName, cfg.Aliases, cfg.Site)
		if got := BrandInQuestion(q, cfg); got != want {
			t.Fatalf("%q: sample=%v model=%v", q, got, want)
		}
	}
}
