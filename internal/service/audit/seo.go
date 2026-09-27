// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"sort"
	"strings"
	"unicode"

	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

// Finding is one occurrence of a registered issue. URL is empty for
// site-level findings.
type Finding struct {
	Code   string         `json:"code"`
	URL    string         `json:"url,omitempty"`
	Detail map[string]any `json:"detail,omitempty"`
}

const slowMs = 3000 // heuristic, see SLOW_RESPONSE

func find(code, url string, detail map[string]any) Finding {
	return Finding{Code: code, URL: url, Detail: detail}
}

// titleTooLong approximates display width: CJK characters count as ~1.9
// Latin characters.
func titleTooLong(title string) bool {
	w := 0.0
	for _, r := range title {
		if unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			w += 1.875
		} else {
			w++
		}
	}
	return w > 60
}

// PageSEO runs the page-level checks that do not need other pages. Status
// errors are reported by ScorePage, so non-200 pages are skipped here.
func PageSEO(p crawl.PageDoc) []Finding {
	if p.Status != 200 {
		return nil
	}
	var out []Finding
	title := strings.TrimSpace(p.Title)
	if title == "" {
		out = append(out, find("MISSING_TITLE", p.URL, nil))
	} else if titleTooLong(title) {
		out = append(out, find("TITLE_TOO_LONG", p.URL, map[string]any{"title": title}))
	}
	if strings.TrimSpace(p.MetaDescription) == "" {
		out = append(out, find("MISSING_META_DESCRIPTION", p.URL, nil))
	}
	if p.ImagesMissingAlt > 0 {
		out = append(out, find("IMAGES_MISSING_ALT", p.URL, map[string]any{"count": p.ImagesMissingAlt}))
	}
	if p.ResponseTimeMs > slowMs {
		out = append(out, find("SLOW_RESPONSE", p.URL, map[string]any{"ms": p.ResponseTimeMs}))
	}
	if len(p.Redirects) >= 2 {
		out = append(out, find("REDIRECT_CHAIN", p.URL, map[string]any{"hops": p.Redirects}))
	}
	return out
}

func normURL(u string) string {
	u = strings.TrimSpace(u)
	if i := strings.Index(u, "#"); i >= 0 {
		u = u[:i]
	}
	return strings.TrimRight(strings.Replace(u, "://www.", "://", 1), "/")
}

// SiteSEO runs checks that compare pages with each other. The first page is
// treated as the crawl root and never reported as an orphan.
func SiteSEO(pages []crawl.PageDoc) []Finding {
	var out []Finding
	byTitle := map[string][]string{}
	byDesc := map[string][]string{}
	status := map[string]int{}
	inbound := map[string]int{}
	for _, p := range pages {
		status[normURL(p.URL)] = p.Status
		if p.FinalURL != "" {
			status[normURL(p.FinalURL)] = p.Status
		}
		if p.Status != 200 {
			continue
		}
		if t := strings.TrimSpace(p.Title); t != "" {
			byTitle[t] = append(byTitle[t], p.URL)
		}
		if d := strings.TrimSpace(p.MetaDescription); d != "" {
			byDesc[d] = append(byDesc[d], p.URL)
		}
	}
	for _, p := range pages {
		for _, l := range p.OutLinks {
			k := normURL(l)
			if k == normURL(p.URL) {
				continue
			}
			inbound[k]++
			if st, ok := status[k]; ok && st >= 400 {
				out = append(out, find("BROKEN_INTERNAL_LINK", p.URL, map[string]any{"target": l, "status": st}))
			}
		}
	}
	out = append(out, dupFindings("DUPLICATE_TITLE", "title", byTitle)...)
	out = append(out, dupFindings("DUPLICATE_META_DESCRIPTION", "description", byDesc)...)
	haveLinks := false
	for _, p := range pages {
		if len(p.OutLinks) > 0 {
			haveLinks = true
			break
		}
	}
	// Orphans can only be judged when the crawl recorded links; older crawls
	// did not, and would report every page.
	if len(pages) >= 3 && haveLinks {
		for i, p := range pages {
			if i == 0 || p.Status != 200 {
				continue
			}
			if inbound[normURL(p.URL)] == 0 {
				out = append(out, find("ORPHAN_PAGE", p.URL, nil))
			}
		}
	}
	return out
}

func dupFindings(code, field string, by map[string][]string) []Finding {
	keys := make([]string, 0, len(by))
	for k, urls := range by {
		if len(urls) > 1 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := make([]Finding, 0, len(keys))
	for _, k := range keys {
		out = append(out, find(code, "", map[string]any{field: k, "urls": by[k]}))
	}
	return out
}
