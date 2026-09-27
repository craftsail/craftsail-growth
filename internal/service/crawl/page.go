// SPDX-License-Identifier: AGPL-3.0-or-later

package crawl

import (
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/craftsail/craftsail-growth/internal/pkg/htmlx"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
)

type PageDoc struct {
	URL              string   `json:"url"`
	FinalURL         string   `json:"final_url"`
	Status           int      `json:"status"`
	UAFallback       bool     `json:"ua_fallback"`
	Error            string   `json:"error"`
	Title            string   `json:"title"`
	MetaDescription  string   `json:"meta_description"`
	MetaRobots       string   `json:"meta_robots"`
	XRobotsTag       string   `json:"x_robots_tag"`
	HreflangCount    int      `json:"hreflang_count"`
	Canonical        string   `json:"canonical"`
	Lang             string   `json:"lang"`
	H1               []string `json:"h1"`
	H2               []string `json:"h2"`
	H3Count          int      `json:"h3_count"`
	ParaCount        int      `json:"para_count"`
	LiCount          int      `json:"li_count"`
	TableCount       int      `json:"table_count"`
	ImgCount         int      `json:"img_count"`
	ImagesMissingAlt int      `json:"images_missing_alt"`
	InternalLinks    int      `json:"internal_links"`
	ExternalLinks    int      `json:"external_links"`
	ResponseTimeMs   int      `json:"response_time_ms"`
	OutLinks         []string `json:"out_links"`
	Redirects        []string `json:"redirects"`
	JSONLDTypes      []string `json:"jsonld_types"`
	JSONLDRaw        []any    `json:"jsonld_raw"`
	WordCount        int      `json:"word_count"`
	Language         string   `json:"language"`
	CJKRatio         float64  `json:"cjk_ratio"`
	Text             string   `json:"text"`
	FetchedAt        string   `json:"fetched_at"`
	HTML             string   `json:"html,omitempty"`
}

func Analyze(pageURL string, res httputil.Result) PageDoc {
	doc := htmlx.Parse(res.HTML)
	text := htmlx.MainText(res.HTML)
	runes := []rune(text)
	if len(runes) > 20000 {
		text = string(runes[:20000])
	}
	blocks := htmlx.JSONLD(res.HTML)
	title := strings.TrimSpace(doc.Find("title").First().Text())
	lang, _ := doc.Find("html").First().Attr("lang")
	desc := meta(doc, "description")
	robots := meta(doc, "robots")
	canonical := ""
	doc.Find(`link`).Each(func(_ int, s *goquery.Selection) {
		rel, _ := s.Attr("rel")
		if strings.Contains(strings.ToLower(rel), "canonical") && canonical == "" {
			canonical, _ = s.Attr("href")
		}
	})
	hreflang := 0
	doc.Find(`link`).Each(func(_ int, s *goquery.Selection) {
		rel, _ := s.Attr("rel")
		if _, ok := s.Attr("hreflang"); ok && strings.Contains(strings.ToLower(rel), "alternate") {
			hreflang++
		}
	})
	paras := 0
	doc.Find("p").Each(func(_ int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) != "" {
			paras++
		}
	})
	ext, internal, missingAlt := 0, 0, 0
	seenOut := map[string]bool{}
	var outLinks []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		u := htmlx.Normalize(pageURL, href)
		if u == "" || !strings.HasPrefix(u, "http") {
			return
		}
		if htmlx.SameSite(pageURL, u) {
			internal++
			if !seenOut[u] && len(outLinks) < 300 {
				seenOut[u] = true
				outLinks = append(outLinks, u)
			}
		} else {
			ext++
		}
	})
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		alt, ok := s.Attr("alt")
		if !ok || strings.TrimSpace(alt) == "" {
			missingAlt++
		}
	})
	if res.XRobotsTag == "" {
		res.XRobotsTag = ""
	}
	h1 := htmlx.Texts(doc, "h1")
	h2 := htmlx.Texts(doc, "h2")
	h3 := htmlx.Texts(doc, "h3")
	if h1 == nil {
		h1 = []string{}
	}
	if h2 == nil {
		h2 = []string{}
	}
	return PageDoc{
		URL:              pageURL,
		FinalURL:         res.FinalURL,
		Status:           res.Status,
		UAFallback:       res.UAFallback,
		Error:            res.Error,
		Title:            title,
		MetaDescription:  desc,
		MetaRobots:       robots,
		XRobotsTag:       res.XRobotsTag,
		HreflangCount:    hreflang,
		Canonical:        canonical,
		Lang:             lang,
		H1:               h1,
		H2:               h2,
		H3Count:          len(h3),
		ParaCount:        paras,
		LiCount:          doc.Find("li").Length(),
		TableCount:       doc.Find("table").Length(),
		ImgCount:         doc.Find("img").Length(),
		ImagesMissingAlt: missingAlt,
		InternalLinks:    internal,
		ExternalLinks:    ext,
		ResponseTimeMs:   int(res.Elapsed * 1000),
		OutLinks:         outLinks,
		Redirects:        res.Redirects,
		JSONLDTypes:      htmlx.JSONLDTypes(blocks),
		JSONLDRaw:        blocks,
		WordCount:        htmlx.WordCount(text),
		Language:         htmlx.PageLanguage(text, lang),
		CJKRatio:         htmlx.CJKRatio(text),
		Text:             text,
		FetchedAt:        time.Now().Format(time.RFC3339),
		HTML:             res.HTML,
	}
}

func CountSignals(pageURL, html string) (internal, missingAlt int) {
	if html == "" {
		return 0, 0
	}
	doc := htmlx.Parse(html)
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		u := htmlx.Normalize(pageURL, href)
		if u != "" && strings.HasPrefix(u, "http") && htmlx.SameSite(pageURL, u) {
			internal++
		}
	})
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		alt, ok := s.Attr("alt")
		if !ok || strings.TrimSpace(alt) == "" {
			missingAlt++
		}
	})
	return internal, missingAlt
}

func meta(doc *goquery.Document, name string) string {
	var out string
	doc.Find("meta").Each(func(_ int, s *goquery.Selection) {
		n, _ := s.Attr("name")
		if strings.EqualFold(n, name) && out == "" {
			out, _ = s.Attr("content")
		}
	})
	return out
}
