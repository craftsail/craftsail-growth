// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"context"
	"net/url"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
)

type CrawlPage struct {
	URL              string   `json:"url"`
	Status           int      `json:"status"`
	Score            float64  `json:"score"`
	Words            int      `json:"words"`
	H1               int      `json:"h1"`
	ImagesMissingAlt int      `json:"images_missing_alt"`
	InternalLinks    int      `json:"internal_links"`
	Millis           int      `json:"millis"`
	Issues           []string `json:"issues"`
}

type CrawlSummary struct {
	Health     int         `json:"health"`
	Pages      int         `json:"pages"`
	Critical   int         `json:"critical"`
	ContentAvg int         `json:"content_avg"`
	Orphans    int         `json:"orphans"`
	Rows       []CrawlPage `json:"rows"`
}

func (s *Service) crawlView(ctx context.Context, projectID uint64, avg float64, pages []model.AuditPage) CrawlSummary {
	stored, _ := s.pages.List(ctx, projectID)
	byURL := map[string]model.Page{}
	for _, p := range stored {
		byURL[p.URL] = p
	}
	var rows []CrawlPage
	for _, ap := range pages {
		row := CrawlPage{URL: ap.URL, Score: ap.Score, Words: ap.WordCount, Issues: issueTitles(ap.IssueCodes)}
		if pg, ok := byURL[ap.URL]; ok {
			row.Status = pg.StatusCode
			if row.Words == 0 {
				row.Words = pg.WordCount
			}
			if an := pg.Analysis; an != nil {
				row.H1 = intSliceLen(an["h1"])
				row.ImagesMissingAlt = intOf(an["images_missing_alt"])
				row.InternalLinks = intOf(an["internal_links"])
				row.Millis = intOf(an["response_time_ms"])
			}
			if row.InternalLinks == 0 && row.ImagesMissingAlt == 0 && pg.HTML != "" {
				internal, missing := crawl.CountSignals(pg.URL, pg.HTML)
				row.InternalLinks = internal
				row.ImagesMissingAlt = missing
			}
		}
		if row.Status == 0 {
			for _, code := range ap.IssueCodes {
				if strings.Contains(code, "404") || strings.Contains(strings.ToLower(code), "status") {
					row.Status = 404
				}
			}
		}
		rows = append(rows, row)
	}
	return CrawlView(rows, int(avg+0.5))
}

func intSliceLen(v any) int {
	switch t := v.(type) {
	case []any:
		return len(t)
	case []string:
		return len(t)
	default:
		return 0
	}
}

func intOf(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}

func CrawlView(pages []CrawlPage, health int) CrawlSummary {
	view := CrawlSummary{Health: health, Pages: len(pages), Rows: pages}
	if pages == nil {
		view.Rows = []CrawlPage{}
	}
	var sum float64
	for _, p := range pages {
		if p.Status >= 400 {
			view.Critical++
		}
		if p.InternalLinks == 0 && !isRoot(p.URL) {
			view.Orphans++
		}
		sum += p.Score
	}
	if len(pages) > 0 {
		view.ContentAvg = int(sum/float64(len(pages)) + 0.5)
	}
	return view
}

func isRoot(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return raw == "/" || raw == ""
	}
	path := strings.TrimSpace(u.Path)
	return path == "" || path == "/"
}

func issueTitles(codes []string) []string {
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		out = append(out, Lookup(c).Title)
	}
	return out
}
