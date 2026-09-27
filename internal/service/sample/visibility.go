// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"net/url"
	"sort"
	"strings"
)

const WebQueriesUnavailable = "unavailable"

func ReportedWebQueries(searched bool, queries []string) []string {
	if !searched {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, q := range queries {
		q = strings.TrimSpace(q)
		if q == "" || seen[q] {
			continue
		}
		seen[q] = true
		out = append(out, q)
	}
	if len(out) == 0 {
		return []string{WebQueriesUnavailable}
	}
	return out
}

func QueriesFromPayload(data map[string]any) []string {
	if data == nil {
		return nil
	}
	var out []string
	var add func(v any)
	add = func(v any) {
		switch t := v.(type) {
		case string:
			if s := strings.TrimSpace(t); s != "" {
				out = append(out, s)
			}
		case []any:
			for _, item := range t {
				add(item)
			}
		case []string:
			for _, s := range t {
				add(s)
			}
		}
	}
	for _, key := range []string{"search_queries", "fan_out_queries", "web_search_query", "search_model_queries"} {
		add(data[key])
	}
	if si, ok := data["search_info"].(map[string]any); ok {
		for _, key := range []string{"search_queries", "fan_out_queries", "web_search_query", "search_model_queries"} {
			add(si[key])
		}
	}
	return out
}

func ClassifyDomain(domain, own string, competitors []string) string {
	host := bareHost(domain)
	if host == "" {
		return "other"
	}
	if ownHost := bareHost(own); ownHost != "" && hostMatch(host, ownHost) {
		return "brand"
	}
	for _, c := range competitors {
		if h := bareHost(c); h != "" && hostMatch(host, h) {
			return "competitor"
		}
	}
	switch {
	case listed(host, socialHosts):
		return "social"
	case listed(host, reviewHosts):
		return "reviews"
	case listed(host, referenceHosts):
		return "reference"
	case listed(host, ecommerceHosts):
		return "ecommerce"
	case listed(host, prHosts):
		return "pr"
	case institutional(host):
		return "institutional"
	default:
		return "other"
	}
}

func PageType(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "other"
	}
	host := bareHost(u.Host)
	path := strings.ToLower(u.Path)
	if host == "bilibili.com" || host == "youtube.com" || host == "youtu.be" || strings.Contains(path, "/video") {
		return "video"
	}
	if path == "" || path == "/" {
		return "homepage"
	}
	switch {
	case strings.Contains(path, "vs") || strings.Contains(path, "compar"):
		return "comparison"
	case strings.Contains(path, "review"):
		return "review"
	case strings.Contains(path, "howto") || strings.Contains(path, "how-to") || strings.Contains(path, "faq"):
		return "howto"
	case strings.Contains(path, "forum") || strings.Contains(path, "thread"):
		return "forum"
	case strings.Contains(path, "product") || strings.Contains(path, "/dp/"):
		return "product"
	case strings.Contains(path, "/doc"):
		return "doc"
	default:
		return "article"
	}
}

type DayDomains struct {
	Date   string
	Counts map[string]int
}

func Stability(days []DayDomains) (*int, string) {
	var kept []DayDomains
	for _, d := range days {
		if dayTotal(d.Counts) > 0 {
			kept = append(kept, d)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].Date < kept[j].Date })
	if len(kept) < 2 {
		return nil, "n/a"
	}
	var sum float64
	n := 0
	for i := 1; i < len(kept); i++ {
		sum += brayCurtis(kept[i-1].Counts, kept[i].Counts)
		n++
	}
	vol := sum / float64(n)
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	score := int((1 - vol) * 100)
	switch {
	case score < 40:
		return &score, "wide-open"
	case score < 70:
		return &score, "contested"
	default:
		return &score, "locked-in"
	}
}

func NextRounds(have, target int) []int {
	if target < 1 || have >= target {
		return nil
	}
	out := make([]int, 0, target-have)
	for r := have + 1; r <= target; r++ {
		out = append(out, r)
	}
	return out
}

func brayCurtis(a, b map[string]int) float64 {
	ta, tb := dayTotal(a), dayTotal(b)
	if ta == 0 || tb == 0 {
		return 1
	}
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	var overlap float64
	for k := range keys {
		sa := float64(a[k]) / float64(ta)
		sb := float64(b[k]) / float64(tb)
		if sa < sb {
			overlap += sa
		} else {
			overlap += sb
		}
	}
	return 1 - overlap
}

func dayTotal(m map[string]int) int {
	n := 0
	for _, c := range m {
		if c > 0 {
			n += c
		}
	}
	return n
}

func bareHost(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			raw = u.Hostname()
		}
	}
	raw = strings.TrimPrefix(raw, "www.")
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

func hostMatch(host, root string) bool {
	return host == root || strings.HasSuffix(host, "."+root)
}

func listed(host string, roots []string) bool {
	for _, root := range roots {
		if hostMatch(host, root) {
			return true
		}
	}
	return false
}

func institutional(host string) bool {
	for _, sfx := range []string{".edu", ".gov", ".mil", ".ac.cn", ".edu.cn", ".gov.cn"} {
		if strings.HasSuffix(host, sfx) {
			return true
		}
	}
	return false
}

var socialHosts = []string{
	"zhihu.com", "xiaohongshu.com", "xhslink.com", "bilibili.com", "weibo.com",
	"mp.weixin.qq.com", "weixin.qq.com", "douyin.com", "baijiahao.baidu.com",
	"reddit.com", "youtube.com", "youtu.be", "x.com", "twitter.com", "facebook.com",
	"linkedin.com", "medium.com", "quora.com", "tieba.baidu.com", "v.qq.com",
}

var reviewHosts = []string{
	"smzdm.com", "g2.com", "capterra.com", "trustpilot.com", "trustradius.com",
}

var referenceHosts = []string{
	"wikipedia.org", "wikidata.org", "baike.baidu.com", "baike.sogou.com",
}

var ecommerceHosts = []string{
	"taobao.com", "tmall.com", "jd.com", "amazon.com", "amazon.cn",
}

var prHosts = []string{
	"prnewswire.com", "businesswire.com", "globenewswire.com",
}
