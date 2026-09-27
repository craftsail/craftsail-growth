// SPDX-License-Identifier: AGPL-3.0-or-later

package htmlx

import (
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var (
	cjkRe   = regexp.MustCompile(`[一-鿿]`)
	kanaRe  = regexp.MustCompile(`[぀-ヿ]`)
	latinRe = regexp.MustCompile(`[A-Za-z][A-Za-z'\-]*`)
	boiler  = regexp.MustCompile(`(?i)(nav|header|footer|sidebar|menu|breadcrumb|cookie|banner|advert)`)
)

var tracking = map[string]bool{
	"fbclid": true, "gclid": true, "dclid": true, "msclkid": true, "igshid": true,
	"mc_cid": true, "mc_eid": true, "ref": true, "spm": true, "scm": true,
}

func Normalize(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	low := strings.ToLower(href)
	if strings.HasPrefix(low, "mailto:") || strings.HasPrefix(low, "tel:") ||
		strings.HasPrefix(low, "javascript:") || strings.HasPrefix(href, "#") {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	out := u.ResolveReference(ref)
	out.Fragment = ""
	if out.RawQuery != "" {
		q := out.Query()
		nq := url.Values{}
		for k, vs := range q {
			lk := strings.ToLower(k)
			if strings.HasPrefix(lk, "utm_") || tracking[lk] {
				continue
			}
			for _, v := range vs {
				nq.Add(k, v)
			}
		}
		out.RawQuery = nq.Encode()
	}
	return out.String()
}

func SameSite(a, b string) bool {
	ha, hb := host(a), host(b)
	return ha == hb || strings.HasSuffix(ha, "."+hb) || strings.HasSuffix(hb, "."+ha)
}

func host(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Host), "www.")
}

func Parse(htmlSrc string) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlSrc))
	if err != nil {
		doc, _ = goquery.NewDocumentFromReader(strings.NewReader(""))
	}
	return doc
}

func MainText(htmlSrc string) string {
	doc := Parse(htmlSrc)
	articles := doc.Find("article")
	var node *goquery.Selection
	if articles.Length() == 1 {
		node = articles.First()
	} else if main := doc.Find("main"); main.Length() > 0 {
		node = main.First()
	} else if body := doc.Find("body"); body.Length() > 0 {
		node = body.First()
	} else {
		node = doc.Selection
	}
	frag, _ := goquery.OuterHtml(node)
	clone := Parse(frag)
	clone.Find("script,style,noscript,svg,iframe,form,template").Remove()
	clone.Find("[class],[id]").Each(func(_ int, s *goquery.Selection) {
		cls, _ := s.Attr("class")
		id, _ := s.Attr("id")
		if boiler.MatchString(cls) || boiler.MatchString(id) {
			s.Remove()
		}
	})
	raw := walkText(clone.Selection)
	var lines []string
	for _, ln := range strings.Split(raw, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	return strings.Join(lines, "\n")
}

func walkText(sel *goquery.Selection) string {
	var b strings.Builder
	if sel == nil || sel.Length() == 0 {
		return ""
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "section", "article", "li", "h1", "h2", "h3", "h4", "br", "tr":
				b.WriteString("\n")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "section", "article", "li", "h1", "h2", "h3", "h4", "tr":
				b.WriteString("\n")
			}
		}
	}
	for _, n := range sel.Nodes {
		walk(n)
	}
	return b.String()
}

func CJKRatio(text string) float64 {
	cjk := len(cjkRe.FindAllString(text, -1))
	latin := len(latinRe.FindAllString(text, -1))
	total := cjk + latin
	if total == 0 {
		return 0
	}
	return float64(cjk) / float64(total)
}

func PageLanguage(text, langAttr string) string {
	if len([]rune(text)) < 80 {
		la := strings.ToLower(langAttr)
		switch {
		case strings.HasPrefix(la, "zh"):
			return "zh"
		case strings.HasPrefix(la, "en"):
			return "en"
		case strings.HasPrefix(la, "ja"):
			return "ja"
		default:
			return "unknown"
		}
	}
	kana := len(kanaRe.FindAllString(text, -1))
	cjk := len(cjkRe.FindAllString(text, -1))
	if kana >= 5 && float64(kana)/float64(kana+cjk) > 0.2 {
		return "ja"
	}
	r := CJKRatio(text)
	if r >= 0.5 {
		return "zh"
	}
	if r <= 0.1 {
		return "en"
	}
	return "mixed"
}

func WordCount(text string) int {
	cjk := len(cjkRe.FindAllString(text, -1)) + len(kanaRe.FindAllString(text, -1))
	latin := len(latinRe.FindAllString(text, -1))
	return int(float64(cjk)/1.6 + float64(latin))
}

func JSONLD(htmlSrc string) []any {
	doc := Parse(htmlSrc)
	var out []any
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		typ, _ := s.Attr("type")
		if !strings.Contains(strings.ToLower(typ), "ld+json") {
			return
		}
		var data any
		if err := json.Unmarshal([]byte(strings.TrimSpace(s.Text())), &data); err != nil {
			return
		}
		if arr, ok := data.([]any); ok {
			out = append(out, arr...)
		} else {
			out = append(out, data)
		}
	})
	return out
}

func JSONLDTypes(blocks []any) []string {
	seen := map[string]bool{}
	var add func(any)
	addType := func(t any) {
		switch v := t.(type) {
		case string:
			seen[v] = true
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok {
					seen[s] = true
				}
			}
		}
	}
	add = func(b any) {
		m, ok := b.(map[string]any)
		if !ok {
			return
		}
		if t, ok := m["@type"]; ok {
			addType(t)
		}
		if g, ok := m["@graph"].([]any); ok {
			for _, sub := range g {
				add(sub)
			}
		}
	}
	for _, b := range blocks {
		add(b)
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func JSONLDHasKey(obj any, keys map[string]bool) bool {
	switch v := obj.(type) {
	case map[string]any:
		for k, val := range v {
			if keys[k] || JSONLDHasKey(val, keys) {
				return true
			}
		}
	case []any:
		for _, x := range v {
			if JSONLDHasKey(x, keys) {
				return true
			}
		}
	}
	return false
}

func Texts(doc *goquery.Document, sel string) []string {
	var out []string
	doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
		t := strings.TrimSpace(s.Text())
		if t != "" {
			out = append(out, t)
		}
	})
	return out
}
