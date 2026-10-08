// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"regexp"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
)

var Groups = []string{"推荐", "比较", "替代", "价格", "风险", "品牌验证", "场景"}

var fakeName = regexp.MustCompile(`(?i)(工具\s*[A-Z一二三四五六七八九十]|某某|XX|示例|competitor\s*[a-z]|foobar|acme)`)

func FilterCompetitors(rows []map[string]any, brandName string) []model.Competitor {
	f := false
	var out []model.Competitor
	seen := map[string]bool{}
	for _, c := range rows {
		n := strings.TrimSpace(asString(c["name"]))
		if n == "" || fakeName.MatchString(n) || n == brandName || seen[n] {
			continue
		}
		var aliases []string
		switch a := c["aliases"].(type) {
		case []any:
			for _, x := range a {
				if s := strings.TrimSpace(asString(x)); s != "" {
					aliases = append(aliases, s)
				}
			}
		case []string:
			for _, s := range a {
				if strings.TrimSpace(s) != "" {
					aliases = append(aliases, strings.TrimSpace(s))
				}
			}
		}
		seen[n] = true
		cf := f
		out = append(out, model.Competitor{Name: n, Aliases: aliases, Market: model.MarketAll, Confirmed: &cf})
		if len(out) >= 14 {
			break
		}
	}
	return out
}

func NormalizeQuestions(rows []map[string]any) []model.Question {
	var out []model.Question
	seen := map[string]bool{}
	inGroup := map[string]bool{}
	for _, g := range Groups {
		inGroup[g] = true
	}
	for _, q := range rows {
		t := strings.TrimSpace(asString(q["text"]))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		g := asString(q["group"])
		if !inGroup[g] {
			g = "推荐"
		}
		id := asString(q["id"])
		if id == "" {
			id = "q" + pad3(len(out)+1)
		}
		out = append(out, model.Question{
			Enabled: true, QID: id, GroupName: g, Market: model.MarketAll, Text: t, Intent: model.IntentOf(g),
		})
	}
	return out
}

// TemplateQuestions returns the fallback prompt library in one language,
// "zh", "en" or "pt".
func TemplateQuestions(name, lang string) []model.Question {
	type pair struct{ g, cn, en string }
	pairs := []pair{
		{"推荐", "有哪些好用的工具？", "What are the best tools in this category?"},
		{"推荐", "中小团队适合用什么方案？", "What should a small team use?"},
		{"比较", "这类产品和通用大模型比有什么差别？", "How does this compare to a general LLM?"},
		{"替代", "如果不用现在的方案还能选什么？", "What are alternatives if I do not use the current stack?"},
		{"价格", "大概要多少钱？有没有免费档？", "How much does it cost and is there a free tier?"},
		{"风险", "用之前要小心什么？", "What are the risks before adopting it?"},
		{"品牌验证", name + "是做什么的？", "What is " + name + "?"},
		{"场景", "第一次接入应该从哪一步开始？", "Where should I start on day one?"},
	}
	pt := []string{"Quais são as melhores ferramentas desta categoria?", "Qual solução uma equipe pequena deve usar?", "Como isso se compara a um modelo de linguagem de uso geral?", "Quais são as alternativas à solução atual?", "Quanto custa e existe um plano gratuito?", "Quais riscos devo considerar antes de adotar?", "O que é " + name + "?", "Por onde devo começar no primeiro dia?"}
	var rows []map[string]any
	for i, p := range pairs {
		text := p.en
		if lang == "zh" {
			text = p.cn
		}
		if lang == "pt" {
			text = pt[i]
		}
		rows = append(rows, map[string]any{"id": "q" + pad3(i+1), "group": p.g, "text": text})
	}
	out := NormalizeQuestions(rows)
	for i := range out {
		out[i].Language = lang
	}
	return out
}

func pad3(n int) string {
	if n < 10 {
		return "00" + itoa(n)
	}
	if n < 100 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	}
	return ""
}
