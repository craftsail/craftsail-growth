// SPDX-License-Identifier: AGPL-3.0-or-later

package generate

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestParseFactsDefinitionAndCJKSpaces(t *testing.T) {
	md := "## 一句话定义\n\n> 甲工智能是 面向团队 的 GEO 工具。\n\n## 关键数字\n\n| 事实 | 数值 | 来源 | 证据 |\n|---|---|---|---|\n| 客户数 | 300家 | 官网 | A |\n\n## 适用与不适用\n\n**适合**：\n\n- 中小团队\n\n**不适合**：\n\n- 个人玩具\n"
	f := ParseFacts(md)
	if f.Definition != "甲工智能是面向团队的 GEO 工具。" {
		t.Fatalf("def %q", f.Definition)
	}
	if len(f.Numbers) != 1 || f.Numbers[0].Value != "300家" {
		t.Fatalf("%+v", f.Numbers)
	}
	if len(f.Suitable) != 1 || f.Suitable[0] != "中小团队" {
		t.Fatalf("%v", f.Suitable)
	}
}

func TestLLMSContainsDefinition(t *testing.T) {
	p := &model.Project{Name: "甲工", Site: "https://a.example", Market: "cn", Brand: model.Brand{Industry: "GEO", Aliases: []string{"甲工智能"}}}
	txt := LLMS(p, Facts{Definition: "甲工是面向团队的 GEO 工具。", Numbers: []Num{{Fact: "客户", Value: "300家"}}}, nil, "zh")
	if !strings.Contains(txt, "# 甲工") || !strings.Contains(txt, "甲工是面向团队的 GEO 工具。") || !strings.Contains(txt, "300家") {
		t.Fatalf("%s", txt)
	}
}

func TestJSONLDOrganization(t *testing.T) {
	p := &model.Project{Name: "甲工", Site: "https://a.example", Brand: model.Brand{Aliases: []string{"甲工智能"}}}
	docs := JSONLD(p, Facts{Definition: "定义句"}, nil)
	org := docs["organization"]
	raw, _ := json.Marshal(org)
	if !strings.Contains(string(raw), `"Organization"`) || !strings.Contains(string(raw), "定义句") {
		t.Fatalf("%s", raw)
	}
}

func TestDefinitionSnippetEscapes(t *testing.T) {
	p := &model.Project{Name: "A&B", Brand: model.Brand{}}
	html := DefinitionBlock(p, Facts{Definition: "<script>"}, "zh")
	if strings.Contains(html, "<script>") && !strings.Contains(html, "&lt;script&gt;") {
		t.Fatal("must escape")
	}
	if !strings.Contains(html, "A&amp;B") {
		t.Fatalf("%s", html)
	}
}

func TestParseFactsReadsLegacyChineseHeadings(t *testing.T) {
	md := "## 一句话定义\n\n> 甲工是面向团队的工具。\n\n## 关键数字\n\n| 事实 | 数值 | 来源 | 证据 |\n|---|---|---|---|\n| 客户 | 300 家 | 官网 | A |\n\n## 适用与不适用\n\n**适合**：\n\n- 小团队\n\n**不适合**：\n\n- 大企业\n"
	f := ParseFacts(md)
	if f.Definition != "甲工是面向团队的工具。" || len(f.Numbers) != 1 || len(f.Suitable) != 1 || len(f.Unsuitable) != 1 {
		t.Fatalf("%+v", f)
	}
}

func TestParseFactsReadsEnglishHeadings(t *testing.T) {
	md := "## One-line definition\n\n> Acme is a CRM for small teams.\n\n## Key numbers\n\n| Fact | Value | Source | Evidence |\n|---|---|---|---|\n| Customers | 300 | site | A |\n\n## Fit\n\n**Good fit**:\n\n- small teams\n\n**Not a fit**:\n\n- enterprises\n"
	f := ParseFacts(md)
	if f.Definition != "Acme is a CRM for small teams." || len(f.Numbers) != 1 || f.Suitable[0] != "small teams" || f.Unsuitable[0] != "enterprises" {
		t.Fatalf("%+v", f)
	}
}
