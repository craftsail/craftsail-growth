// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/pkg/htmlx"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

var (
	reDefinition = regexp.MustCompile(`(?i)(是一[款种个家类]|是指|指的是|定义为|全称[为是]|又称|简称为?|属于一[种类]|とは|を指す|と呼ばれ|の略|\bis an? \w+|\brefers to\b|\bis defined as\b|\bstands for\b)`)
	reNumber     = regexp.MustCompile(`\d[\d,\.]*\s*(%|％|万|亿|千|倍|元|美元|人|家|个|天|小时|分钟|秒|次|条|款|年|月|件|社|名|回|億|円|時間|percent|x\b|hours?|days?|users?|customers?)`)
	reCompare    = regexp.MustCompile(`(?i)(对比|相比|区别|差异|优于|不如|竞品|替代|选型|哪个好|比較|違い|\bvs\.?\b|\bversus\b|\balternatives?\b)`)
	reHowto      = regexp.MustCompile(`(?i)(第[一二三四五六七八九十\d]+步|步骤\s*[一二三四五六七八九十\d]|操作流程|手順|ステップ\s*\d|使い方|\bstep\s*\d|\bhow to\b)`)
	reHowtoSoft  = regexp.MustCompile(`(如何|怎么)`)
	funcPage     = regexp.MustCompile(`(?i)/(login|signin|signup|register|cart|checkout|account|auth|contact)(/|$)`)
	shellPhrase  = regexp.MustCompile(`(?i)(加载中|请稍候|请启用\s*javascript|enable javascript|you need to enable javascript|^loading[\s.!*]*$|^please wait[\s.!*]*$)`)
	emptyMount   = regexp.MustCompile(`(?is)<(div|main)[^>]*\bid=["'](root|app|__next)["'][^>]*>\s*</(div|main)>`)
	reFAQ        = regexp.MustCompile(`(?im)(常见问题|常见疑问|问答|よくある質問|\bFAQ\b|^\s*[问Q][:：]|答[:：])`)
	reDate       = regexp.MustCompile(`(?i)(20\d{2}[-/年]\s?\d{1,2}[-/月]\s?\d{1,2}|更新[于时间]*[:：]?\s*20\d{2}|最后更新|发布于|\bupdated\b|\bpublished\b)`)
	reAuthor     = regexp.MustCompile(`(?i)(作者|撰文|编辑[:：]|著者|執筆|\bauthor\b|\bby\s+[A-Z][a-z]+)`)
	kwToken      = regexp.MustCompile(`[一-鿿A-Za-z]{2,}`)
)

var authoritySchema = map[string]bool{
	"Organization": true, "Corporation": true, "Product": true, "SoftwareApplication": true, "Service": true,
	"FAQPage": true, "Article": true, "TechArticle": true, "NewsArticle": true, "BlogPosting": true,
	"HowTo": true, "BreadcrumbList": true, "WebSite": true, "Review": true, "AggregateRating": true, "Offer": true,
}

type PageScore struct {
	URL              string             `json:"url"`
	Title            string             `json:"title"`
	WordCount        int                `json:"word_count"`
	Score            float64            `json:"score"`
	Grade            string             `json:"grade"`
	Dimensions       map[string]float64 `json:"dimensions"`
	SectionsTotal    int                `json:"sections_total"`
	SectionsQuotable int                `json:"sections_quotable"`
	Blocks           map[string]bool    `json:"blocks"`
	JSONLDTypes      []string           `json:"jsonld_types"`
	IssueCodes       []string           `json:"issue_codes"`
}

func band(value float64, stops [][2]float64) float64 {
	for _, st := range stops {
		if value >= st[0] {
			return st[1]
		}
	}
	return 0
}

func canonKey(u string) string {
	p, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return u
	}
	host := strings.TrimPrefix(strings.ToLower(p.Host), "www.")
	path := strings.TrimRight(p.Path, "/")
	if path == "" {
		path = "/"
	}
	return host + path
}

func canonMismatch(canonical, actual string) bool {
	if !strings.HasPrefix(canonical, "http") {
		return false
	}
	return canonKey(canonical) != canonKey(actual)
}

func splitSections(text string, h2s []string) []string {
	heads := map[string]bool{}
	for _, h := range h2s {
		h = strings.TrimSpace(h)
		if h != "" {
			heads[h] = true
		}
	}
	if len(heads) == 0 {
		return nil
	}
	lines := strings.Split(text, "\n")
	var idx []int
	for i, ln := range lines {
		if heads[strings.TrimSpace(ln)] {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return nil
	}
	idx = append(idx, len(lines))
	var out []string
	for i := 0; i < len(idx)-1; i++ {
		seg := strings.TrimSpace(strings.Join(lines[idx[i]+1:idx[i+1]], "\n"))
		out = append(out, seg)
	}
	return out
}

func quotable(seg string) bool {
	if htmlx.WordCount(seg) < 60 {
		return false
	}
	return reNumber.MatchString(seg) || reDefinition.MatchString(seg) || reHowto.MatchString(seg)
}

// emptyShell is a rendering check, not a length check. A short index page
// that already has sentences in the static HTML is not an empty shell.
func emptyShell(page crawl.PageDoc) bool {
	text := strings.TrimSpace(page.Text)
	if page.WordCount >= 20 && !shellPhrase.MatchString(text) {
		return false
	}
	if text == "" || shellPhrase.MatchString(text) {
		return true
	}
	if page.HTML != "" && emptyMount.MatchString(page.HTML) && page.WordCount < 20 {
		return true
	}
	return page.WordCount < 12 && len(page.H1) == 0
}

func ScorePage(page crawl.PageDoc, keywords []string) PageScore {
	text := page.Text
	wc := page.WordCount
	h1, h2 := page.H1, page.H2
	if h1 == nil {
		h1 = []string{}
	}
	if h2 == nil {
		h2 = []string{}
	}
	paras := page.ParaCount
	lis := page.LiCount
	types := map[string]bool{}
	for _, t := range page.JSONLDTypes {
		types[t] = true
	}
	var codes []string
	content := IsContentPage(page)
	issue := func(code string) {
		if Lookup(code).Scope == ScopeContent && !content {
			return
		}
		codes = append(codes, code)
	}
	d := map[string]float64{}

	s := 0.0
	status := page.Status
	switch {
	case status == 200:
		s += 7
	case status > 200 && status < 400:
		s += 3
		issue("NON_200_STATUS")
	case status >= 500:
		issue("SERVER_ERROR")
	case status >= 400:
		issue("CLIENT_ERROR")
	default:
		issue("PAGE_UNREACHABLE")
	}
	metaNo := strings.Contains(strings.ToLower(page.MetaRobots), "noindex")
	headNo := strings.Contains(strings.ToLower(page.XRobotsTag), "noindex")
	if !metaNo && !headNo {
		s += 3
	} else if headNo {
		issue("XROBOTS_NOINDEX")
	} else {
		issue("NOINDEX")
	}
	canon := page.Canonical
	if canon != "" {
		s += 2
		actual := page.FinalURL
		if actual == "" {
			actual = page.URL
		}
		if canonMismatch(canon, actual) {
			issue("CANONICAL_MISMATCH")
		}
	} else {
		issue("NO_CANONICAL")
	}
	if wc >= 120 {
		s += 3
	} else {
		path := ""
		if u, err := url.Parse(page.URL); err == nil {
			path = u.Path
		}
		if funcPage.MatchString(path) {
			issue("LOW_CONTENT_PAGE")
		} else if emptyShell(page) {
			issue("SPA_SHELL")
		}
	}
	d["可抓取性"] = s

	r := band(float64(wc), [][2]float64{{1500, 1}, {1000, 0.85}, {600, 0.6}, {300, 0.35}, {120, 0.15}})
	d["内容长度"] = 15 * r
	if wc < 1000 {
		issue("SHORT_CONTENT")
	}

	s = 0
	switch len(h1) {
	case 1:
		s += 4
	case 0:
		issue("MISSING_H1")
	default:
		s += 2
		issue("MULTIPLE_H1")
	}
	s += 6 * band(float64(len(h2)), [][2]float64{{8, 1}, {6, 0.85}, {4, 0.6}, {2, 0.3}})
	if len(h2) < 6 {
		issue("FEW_H2")
	}
	s += 5 * band(float64(paras), [][2]float64{{40, 1}, {25, 0.8}, {15, 0.55}, {8, 0.3}})
	density := float64(lis) / float64(max(paras+lis, 1))
	s += 5 * band(density, [][2]float64{{0.35, 1}, {0.2, 0.75}, {0.1, 0.45}, {0.03, 0.2}})
	if density < 0.1 {
		issue("LOW_LIST_DENSITY")
	}
	d["结构规范"] = s

	has := map[string]bool{
		"定义":   reDefinition.MatchString(text),
		"数字事实": len(reNumber.FindAllString(text, -1)) >= 3,
		"对比":   reCompare.MatchString(text) || page.TableCount >= 1,
		"操作步骤": reHowto.MatchString(text) || (reHowtoSoft.MatchString(text) && lis >= 3),
		"FAQ":  reFAQ.MatchString(text) || types["FAQPage"],
	}
	// FAQ is detected for display but not scored: Q&A formatting showed
	// -5.74% mean influence in the absorption dataset (zhang2026).
	blockCodes := map[string]string{"定义": "NO_DEFINITION", "数字事实": "NO_NUMBERS", "对比": "NO_COMPARISON", "操作步骤": "NO_HOWTO"}
	weights := map[string]float64{"定义": 7, "数字事实": 7, "对比": 6, "操作步骤": 5}
	sumB := 0.0
	for k, w := range weights {
		if has[k] {
			sumB += w
		} else {
			issue(blockCodes[k])
		}
	}
	d["可抽取块"] = sumB

	segs := splitSections(text, h2)
	qN := 0
	for _, sg := range segs {
		if quotable(sg) {
			qN++
		}
	}
	if len(segs) >= 3 && wc >= 300 && qN == 0 {
		issue("NO_QUOTABLE_PASSAGE")
	}

	s = 0
	if reDate.MatchString(text) || htmlx.JSONLDHasKey(page.JSONLDRaw, map[string]bool{"dateModified": true, "datePublished": true}) {
		s += 4
	} else {
		issue("NO_DATE")
	}
	if reAuthor.MatchString(text) {
		s += 2
	}
	ext := page.ExternalLinks
	s += 4 * band(float64(ext), [][2]float64{{6, 1}, {3, 0.7}, {1, 0.4}})
	if ext < 3 {
		issue("FEW_EXTERNAL_LINKS")
	}
	hit := 0
	var typeList []string
	for t := range types {
		typeList = append(typeList, t)
		if authoritySchema[t] {
			hit++
		}
	}
	s += 5 * band(float64(hit), [][2]float64{{3, 1}, {2, 0.75}, {1, 0.45}})
	if hit == 0 {
		issue("NO_JSONLD")
	}
	if types["FAQPage"] && !reFAQ.MatchString(text) {
		issue("SCHEMA_CONTENT_MISMATCH")
	}
	if (types["Article"] || types["TechArticle"] || types["NewsArticle"] || types["BlogPosting"]) &&
		!htmlx.JSONLDHasKey(page.JSONLDRaw, map[string]bool{"author": true}) {
		issue("NO_AUTHOR_ENTITY")
	}
	d["权威信号"] = s

	surface := strings.ToLower(strings.Join(append(append([]string{page.Title}, h1...), h2...), " "))
	hits := 0
	for _, k := range keywords {
		if k != "" && strings.Contains(surface, strings.ToLower(k)) {
			hits++
		}
	}
	cover := 0.0
	if len(keywords) > 0 {
		cover = float64(hits) / float64(len(keywords))
	}
	d["对题性"] = 10 * band(cover, [][2]float64{{0.4, 1}, {0.25, 0.8}, {0.12, 0.55}, {0.04, 0.3}})
	if cover < 0.12 {
		issue("LOW_RELEVANCE")
	}

	total := 0.0
	for _, v := range d {
		total += v
	}
	total = float64(int(total*10+0.5)) / 10
	grade := "D"
	if total >= 80 {
		grade = "A"
	} else if total >= 65 {
		grade = "B"
	} else if total >= 45 {
		grade = "C"
	}
	dims := map[string]float64{}
	for k, v := range d {
		dims[k] = float64(int(v*10+0.5)) / 10
	}
	title := page.Title
	if len([]rune(title)) > 120 {
		title = string([]rune(title)[:120])
	}
	tl := make([]string, 0, len(types))
	for t := range types {
		tl = append(tl, t)
	}
	return PageScore{
		URL: page.URL, Title: title, WordCount: wc, Score: total, Grade: grade,
		Dimensions: dims, SectionsTotal: len(segs), SectionsQuotable: qN,
		Blocks: has, JSONLDTypes: tl, IssueCodes: codes,
	}
}

func KeywordsFromConfig(cfg map[string]any) []string {
	brand := map[string]bool{}
	if b, ok := cfg["brand"].(map[string]any); ok {
		add := func(v any) {
			if s, ok := v.(string); ok && s != "" {
				brand[strings.ToLower(s)] = true
			}
		}
		add(b["name"])
		if a, ok := b["aliases"].([]any); ok {
			for _, x := range a {
				add(x)
			}
		}
		if a, ok := b["products"].([]any); ok {
			for _, x := range a {
				add(x)
			}
		}
	}
	kws := map[string]bool{}
	if qs, ok := cfg["questions"].([]any); ok {
		for _, q := range qs {
			m, _ := q.(map[string]any)
			text, _ := m["text"].(string)
			for _, tok := range kwToken.FindAllString(text, -1) {
				if len([]rune(tok)) >= 2 {
					kws[tok] = true
				}
			}
		}
	}
	stop := map[string]bool{"什么": true, "怎么": true, "哪个": true, "如何": true, "可以": true, "适合": true, "推荐": true, "有没有": true,
		"the": true, "and": true, "for": true, "how": true, "what": true, "which": true}
	var out []string
	for k := range kws {
		lk := strings.ToLower(k)
		if stop[lk] || brand[lk] || len([]rune(k)) < 2 {
			continue
		}
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
