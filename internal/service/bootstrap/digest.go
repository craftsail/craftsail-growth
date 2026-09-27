// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type PageText struct {
	URL, Title, Text string
}

func Digest(pages []PageText, scores map[string]float64, materials string, limit int) string {
	if limit <= 0 {
		limit = 14000
	}
	if len(pages) == 0 {
		return materialsDigest(materials, limit)
	}
	root := pages[0].URL
	ordered := append([]PageText(nil), pages...)
	sort.SliceStable(ordered, func(i, j int) bool {
		ai, aj := 0, 0
		if ordered[i].URL != root {
			ai = 1
		}
		if ordered[j].URL != root {
			aj = 1
		}
		if ai != aj {
			return ai < aj
		}
		return scores[ordered[i].URL] > scores[ordered[j].URL]
	})
	var b strings.Builder
	used := 0
	for _, p := range ordered {
		if strings.TrimSpace(p.Text) == "" {
			continue
		}
		title := p.Title
		if title == "" {
			title = p.URL
		}
		text := p.Text
		if utf8.RuneCountInString(text) > 2600 {
			text = string([]rune(text)[:2600])
		}
		block := fmt.Sprintf("\n## 页面：%s\nURL: %s\n%s\n", title, p.URL, text)
		if used+len(block) > limit {
			break
		}
		b.WriteString(block)
		used += len(block)
	}
	return b.String()
}

func materialsDigest(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	var body []string
	for _, ln := range strings.Split(text, "\n") {
		s := strings.TrimSpace(ln)
		if s == "" {
			continue
		}
		ls := strings.TrimLeft(ln, " \t")
		if strings.HasPrefix(ls, "#") || strings.HasPrefix(ls, ">") || strings.HasPrefix(ls, "（") {
			continue
		}
		body = append(body, s)
	}
	if len(strings.Join(body, "\n")) < 40 {
		return ""
	}
	if utf8.RuneCountInString(text) > limit {
		text = string([]rune(text)[:limit])
	}
	return "\n## 商品/品牌介绍材料（无自有网站，以下是唯一依据）\n" + text + "\n"
}
