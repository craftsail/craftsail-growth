// SPDX-License-Identifier: AGPL-3.0-or-later

package crawl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/htmlx"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/pkg/robots"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

var AIBots = []string{"GPTBot", "OAI-SearchBot", "ClaudeBot", "Claude-SearchBot", "PerplexityBot",
	"Bytespider", "Baiduspider", "Sogou web spider", "YisouSpider", "Google-Extended"}

var AIUAProbes = map[string]string{
	"GPTBot":        "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; GPTBot/1.2; +https://openai.com/gptbot",
	"ClaudeBot":     "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; ClaudeBot/1.0; +claudebot@anthropic.com",
	"PerplexityBot": "Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; PerplexityBot/1.0; +https://perplexity.ai/perplexitybot",
	"Bytespider":    "Mozilla/5.0 (Linux; Android 5.0) AppleWebKit/537.36 (KHTML, like Gecko) Mobile Safari/537.36 (compatible; Bytespider; spider-feedback@bytedance.com)",
}

type Service struct {
	HTTP     *httputil.Client
	projects *repo.Projects
	pages    *repo.Pages
	sites    *repo.SiteSignals
	Delay    time.Duration
}

func New(db *gorm.DB, httpc *httputil.Client) *Service {
	if httpc == nil {
		httpc = httputil.New()
	}
	return &Service{
		HTTP: httpc, projects: &repo.Projects{DB: db},
		pages: &repo.Pages{DB: db}, sites: &repo.SiteSignals{DB: db},
		Delay: 500 * time.Millisecond,
	}
}

type Result struct {
	Slug         string         `json:"slug"`
	NoSite       bool           `json:"no_site"`
	PagesCrawled int            `json:"pages_crawled"`
	PagesOK      int            `json:"pages_ok"`
	Site         map[string]any `json:"site"`
}

func (s *Service) Run(ctx context.Context, slug string, maxPages int) (*Result, error) {
	p, err := s.projects.BySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("project not found")
	}
	if p.NoSite || strings.TrimSpace(p.Site) == "" {
		site := map[string]any{"slug": slug, "no_site": true}
		_ = s.sites.Upsert(ctx, p.ID, site)
		return &Result{Slug: slug, NoSite: true}, nil
	}
	root := strings.TrimRight(p.Site, "/")
	limit := maxPages
	if limit <= 0 {
		limit = p.PagesMax
	}
	if limit <= 0 {
		limit = 25
	}

	robotsTxt := s.HTTP.FetchText(htmlx.Normalize(root, "/robots.txt"))
	llmsTxt := s.HTTP.FetchText(htmlx.Normalize(root, "/llms.txt"))
	sitemapURLs := s.discoverSitemap(root, robotsTxt)
	home := s.HTTP.Fetch(root, httputil.FetchOpts{Retries: 1})
	linkURLs := []string{}
	if home.HTML != "" {
		linkURLs = DiscoverLinks(root, home.HTML, 200)
	}
	candidates := Rank(append(append(p.PagesSeed, sitemapURLs...), linkURLs...), root)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	var docs []PageDoc
	for _, u := range candidates {
		res := home
		if strings.TrimRight(u, "/") != strings.TrimRight(root, "/") {
			res = s.HTTP.Fetch(u, httputil.FetchOpts{Retries: 1})
		}
		docs = append(docs, Analyze(u, res))
		if s.Delay > 0 {
			time.Sleep(s.Delay)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}

	paths := []string{}
	for i, u := range candidates {
		if i >= 12 {
			break
		}
		pu, _ := url.Parse(u)
		path := pu.Path
		if path == "" {
			path = "/"
		}
		paths = append(paths, path)
	}
	blocked, partial := CheckRobots(robotsTxt, paths)
	uaProbe, uaBlocked := s.probeAIUA(root, home, robotsTxt)
	llmsCheck := s.checkLLMS(root, llmsTxt, robotsTxt)

	noisyRe := regexp.MustCompile(`(?i)/(search|tag|page/\d+|sessions?)($|/|\?)`)
	noisy := 0
	var noisyEx string
	for _, u := range sitemapURLs {
		if strings.Contains(u, "?") || noisyRe.MatchString(u) {
			noisy++
			if noisyEx == "" {
				noisyEx = u
			}
		}
	}

	ok := 0
	uaFb := 0
	var models []model.Page
	now := time.Now()
	for _, d := range docs {
		if d.Status == 200 {
			ok++
		}
		if d.UAFallback {
			uaFb++
		}
		models = append(models, toModel(p.ID, d, now))
	}
	if err := s.pages.Replace(ctx, p.ID, models); err != nil {
		return nil, err
	}

	site := map[string]any{
		"slug": slug, "root": root, "crawled_at": now.Format(time.RFC3339),
		"has_robots": robotsTxt != "", "has_llms_txt": llmsTxt != "",
		"has_sitemap": len(sitemapURLs) > 0, "sitemap_url_count": len(sitemapURLs),
		"robots_sitemap_declared": regexp.MustCompile(`(?im)^\s*sitemap:`).MatchString(robotsTxt),
		"sitemap_noisy_urls":      noisy, "sitemap_noisy_example": noisyEx,
		"ai_bots_blocked": blocked, "ai_bots_partial": partial,
		"ai_ua_probe": uaProbe, "ai_ua_blocked": uaBlocked,
		"llms_txt_check": llmsCheck,
		"pages_crawled":  len(docs), "pages_ok": ok, "ua_fallback_pages": uaFb,
	}
	if err := s.sites.Upsert(ctx, p.ID, site); err != nil {
		return nil, err
	}
	if err := CheckHealth(docs); err != nil {
		return &Result{Slug: slug, PagesCrawled: len(docs), PagesOK: ok, Site: site}, err
	}
	return &Result{Slug: slug, PagesCrawled: len(docs), PagesOK: ok, Site: site}, nil
}

func CheckRobots(robotsTxt string, samplePaths []string) ([]string, []map[string]any) {
	groups := robots.Parse(robotsTxt)
	var blocked []string
	var partial []map[string]any
	for _, bot := range AIBots {
		ok, _ := robots.Decision(groups, bot, "/")
		if !ok {
			blocked = append(blocked, bot)
			continue
		}
		var bad []string
		var rule string
		for _, p := range samplePaths {
			ok, r := robots.Decision(groups, bot, p)
			if !ok {
				if rule == "" {
					rule = r
				}
				if len(bad) < 3 {
					bad = append(bad, p)
				}
			}
		}
		if len(bad) > 0 {
			n := 0
			for _, p := range samplePaths {
				ok, _ := robots.Decision(groups, bot, p)
				if !ok {
					n++
				}
			}
			partial = append(partial, map[string]any{"bot": bot, "rule": rule, "paths": bad, "count": n, "sampled": len(samplePaths)})
		}
	}
	return blocked, partial
}

func CheckHealth(pages []PageDoc) error {
	if len(pages) == 0 {
		return nil
	}
	ok := 0
	for _, p := range pages {
		if p.Status == 200 {
			ok++
		}
	}
	if ok == 0 {
		return fmt.Errorf("crawl failed: no page returned 200")
	}
	if len(pages) >= 5 && float64(ok)/float64(len(pages)) < 0.2 {
		return fmt.Errorf("crawl failed: only %d of %d pages reachable (under 20%%)", ok, len(pages))
	}
	return nil
}

func (s *Service) discoverSitemap(root, robotsTxt string) []string {
	var urls []string
	seen := map[string]bool{}
	queue := []string{htmlx.Normalize(root, "/sitemap.xml"), htmlx.Normalize(root, "/sitemap_index.xml")}
	re := regexp.MustCompile(`(?im)^\s*sitemap:\s*(\S+)`)
	for _, m := range re.FindAllStringSubmatch(robotsTxt, -1) {
		queue = append(queue, strings.TrimSpace(m[1]))
	}
	locRe := regexp.MustCompile(`<loc>\s*([^<]+?)\s*</loc>`)
	for len(queue) > 0 && len(urls) < 300 && len(seen) < 8 {
		sm := queue[0]
		queue = queue[1:]
		if sm == "" || seen[sm] {
			continue
		}
		seen[sm] = true
		xml := s.HTTP.FetchText(sm)
		if xml == "" {
			continue
		}
		locs := locRe.FindAllStringSubmatch(xml, -1)
		var found []string
		for _, m := range locs {
			found = append(found, m[1])
		}
		if strings.Contains(xml, "<sitemapindex") {
			if len(found) > 20 {
				found = found[:20]
			}
			queue = append(queue, found...)
		} else {
			urls = append(urls, found...)
		}
	}
	return urls
}

func (s *Service) probeAIUA(root string, home httputil.Result, robotsTxt string) (map[string]int, []string) {
	probe := map[string]int{}
	var blocked []string
	if home.Status != 200 {
		return probe, blocked
	}
	groups := robots.Parse(robotsTxt)
	for bot, ua := range AIUAProbes {
		ok, _ := robots.Decision(groups, bot, "/")
		if !ok {
			continue
		}
		res := s.HTTP.Fetch(root, httputil.FetchOpts{Timeout: 10 * time.Second, Retries: 0, UA: ua})
		probe[bot] = res.Status
		switch res.Status {
		case 401, 403, 406, 429, 451, 503:
			blocked = append(blocked, bot)
		}
		if s.Delay > 0 {
			time.Sleep(s.Delay)
		}
	}
	return probe, blocked
}

func (s *Service) checkLLMS(root, llms, robotsTxt string) map[string]any {
	if llms == "" {
		return nil
	}
	urlRe := regexp.MustCompile(`https?://[^\s)\]>"'\x60]+`)
	var urls []string
	seen := map[string]bool{}
	for _, u := range urlRe.FindAllString(llms, -1) {
		u = strings.TrimRight(u, ".,;:")
		if htmlx.SameSite(root, u) && httputil.Fetchable(u) && !seen[u] {
			seen[u] = true
			urls = append(urls, u)
		}
	}
	groups := robots.Parse(robotsTxt)
	var broken, robotsBlocked []map[string]any
	sample := urls
	if len(sample) > 6 {
		sample = sample[:6]
	}
	for _, u := range sample {
		pu, _ := url.Parse(u)
		path := pu.Path
		if path == "" {
			path = "/"
		}
		var denied []string
		for _, b := range []string{"GPTBot", "ClaudeBot", "PerplexityBot"} {
			ok, _ := robots.Decision(groups, b, path)
			if !ok {
				denied = append(denied, b)
			}
		}
		if len(denied) > 0 {
			robotsBlocked = append(robotsBlocked, map[string]any{"url": u, "bots": denied})
		}
		res := s.HTTP.Fetch(u, httputil.FetchOpts{Timeout: 10 * time.Second, Retries: 0})
		if res.Status != 200 {
			broken = append(broken, map[string]any{"url": u, "status": res.Status})
		}
		if s.Delay > 0 {
			time.Sleep(300 * time.Millisecond)
		}
	}
	return map[string]any{"total_links": len(urls), "checked": len(sample), "broken": broken, "robots_blocked": robotsBlocked}
}

func toModel(projectID uint64, d PageDoc, now time.Time) model.Page {
	b, _ := json.Marshal(d)
	var analysis map[string]any
	_ = json.Unmarshal(b, &analysis)
	delete(analysis, "html")
	ft := now.Unix()
	if d.FetchedAt != "" {
		if t, err := time.Parse(time.RFC3339, d.FetchedAt); err == nil {
			ft = t.Unix()
		}
	}
	u := d.URL
	if len(u) > 512 {
		u = u[:512]
	}
	return model.Page{
		ProjectID:  projectID,
		URL:        u,
		Title:      d.Title,
		StatusCode: d.Status,
		FinalURL:   d.FinalURL,
		HTML:       d.HTML,
		Text:       d.Text,
		FetchedAt:  &ft,
		FetchError: d.Error,
		WordCount:  d.WordCount,
		Analysis:   analysis,
	}
}

func FromModel(p model.Page) PageDoc {
	b, _ := json.Marshal(p.Analysis)
	var d PageDoc
	_ = json.Unmarshal(b, &d)
	d.URL = p.URL
	d.FinalURL = p.FinalURL
	d.Status = p.StatusCode
	d.HTML = p.HTML
	d.Text = p.Text
	d.WordCount = p.WordCount
	d.Error = p.FetchError
	if d.H1 == nil {
		d.H1 = []string{}
	}
	if d.H2 == nil {
		d.H2 = []string{}
	}
	return d
}
