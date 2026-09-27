// SPDX-License-Identifier: AGPL-3.0-or-later

package crawl

import (
	"net/url"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/craftsail/craftsail-growth/internal/pkg/htmlx"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
)

var priority = []string{
	"product", "pricing", "price", "solution", "case", "customer", "doc", "docs",
	"help", "faq", "about", "news", "blog", "guide", "compare", "vs", "feature",
	"产品", "价格", "方案", "案例", "客户", "文档", "帮助", "关于", "新闻", "博客",
}

func DiscoverLinks(root, htmlSrc string, limit int) []string {
	doc := htmlx.Parse(htmlSrc)
	var out []string
	seen := map[string]bool{}
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if len(out) >= limit {
			return
		}
		href, _ := s.Attr("href")
		u := htmlx.Normalize(root, href)
		if u != "" && htmlx.SameSite(root, u) && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	})
	return out
}

func Rank(urls []string, root string) []string {
	rootU, _ := url.Parse(root)
	rootHost := strings.TrimPrefix(strings.ToLower(rootU.Host), "www.")
	seen := map[string]string{}
	var order []string
	for _, u := range append([]string{root}, urls...) {
		if !httputil.Fetchable(u) {
			continue
		}
		key := strings.TrimRight(u, "/")
		if key == "" {
			key = u
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = u
		order = append(order, u)
	}
	sort.SliceStable(order, func(i, j int) bool {
		ai, aj := rankTuple(order[i], root, rootHost), rankTuple(order[j], root, rootHost)
		for k := 0; k < 5; k++ {
			if ai[k] != aj[k] {
				return ai[k] < aj[k]
			}
		}
		return false
	})
	return order
}

func rankTuple(u, root, rootHost string) [5]int {
	pu, _ := url.Parse(u)
	p := pu.Path
	if p == "" {
		p = "/"
	}
	depth := 0
	for _, x := range strings.Split(p, "/") {
		if x != "" {
			depth++
		}
	}
	hit := 1
	low := strings.ToLower(u)
	for _, k := range priority {
		if strings.Contains(low, strings.ToLower(k)) {
			hit = 0
			break
		}
	}
	sub := 1
	host := strings.TrimPrefix(strings.ToLower(pu.Host), "www.")
	if host == rootHost {
		sub = 0
	}
	home := 1
	if strings.TrimRight(u, "/") == strings.TrimRight(root, "/") {
		home = 0
	}
	return [5]int{home, sub, hit, depth, len(u)}
}
