// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/sitemap"
)

func contentGroup(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "page"
	}
	p := strings.ToLower(u.Path)
	switch {
	case strings.HasSuffix(p, ".pdf"):
		return "document"
	case strings.Contains(p, "/blog/") || strings.Contains(p, "/articles/"):
		return "article"
	case strings.Contains(p, "/docs/") || strings.Contains(p, "/help/"):
		return "documentation"
	case strings.Contains(p, "/products/"):
		return "product"
	case p == "" || p == "/":
		return "homepage"
	}
	return "page"
}
func sitemapAllowed(property, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	if strings.HasPrefix(property, "sc-domain:") {
		return urlInProperty(property, raw)
	}
	p, err := url.Parse(property)
	return err == nil && strings.EqualFold(p.Host, u.Host) && strings.EqualFold(p.Scheme, u.Scheme)
}
func (s *Service) fetchDiscovery(ctx context.Context, property, raw string) (*http.Response, error) {
	if !sitemapAllowed(property, raw) {
		return nil, fmt.Errorf("sitemap outside the configured property host")
	}
	c := s.SiteHTTP
	if c == nil {
		c = &http.Client{Timeout: 15 * time.Second}
	}
	client := *c
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !sitemapAllowed(property, req.URL.String()) {
			return fmt.Errorf("sitemap redirect outside property or too many redirects")
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Craftsail-Growth/1.0")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		res.Body.Close()
		return nil, fmt.Errorf("sitemap HTTP %d", res.StatusCode)
	}
	return res, nil
}

// Discovery is bounded per turn, not per site. Child sitemap work survives a
// process restart, and a malformed document never replaces its prior URL set.
func (s *Service) discoverIndexSitemaps(ctx context.Context, p *model.Project, property string, now time.Time) error {
	seedKey := model.RowKey("sitemap-seeds", property)
	seeded, err := s.rows.GetSync(ctx, p.ID, seedKey)
	if err != nil {
		return err
	}
	if seeded.IsZero() || seeded.Before(dateOnly(now)) {
		root := strings.TrimSpace(p.Site)
		if p.NoSite {
			root = ""
		}
		if root == "" {
			if strings.HasPrefix(property, "sc-domain:") {
				root = "https://" + strings.TrimPrefix(property, "sc-domain:")
			} else {
				root = property
			}
		}
		u, err := url.Parse(root)
		if err != nil {
			return err
		}
		u.Path = "/"
		u.RawQuery = ""
		u.Fragment = ""
		seeds := []string{u.ResolveReference(&url.URL{Path: "/sitemap.xml"}).String(), u.ResolveReference(&url.URL{Path: "/sitemap_index.xml"}).String()}
		if res, err := s.fetchDiscovery(ctx, property, u.ResolveReference(&url.URL{Path: "/robots.txt"}).String()); err == nil {
			body, readErr := io.ReadAll(io.LimitReader(res.Body, 1024*1024))
			res.Body.Close()
			if readErr == nil {
				for _, line := range strings.Split(string(body), "\n") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "sitemap") {
						v := strings.TrimSpace(parts[1])
						if sitemapAllowed(property, v) {
							seeds = append(seeds, v)
						}
					}
				}
			}
		}
		if maps, err := s.rows.ListSitemaps(ctx, p.ID, property); err == nil {
			for _, m := range maps {
				if sitemapAllowed(property, m.Path) {
					seeds = append(seeds, m.Path)
				}
			}
		}
		if err := s.rows.QueueSitemaps(ctx, p.ID, property, seeds); err != nil {
			return err
		}
		if err := s.rows.PutSync(ctx, p.ID, seedKey, dateOnly(now)); err != nil {
			return err
		}
	}
	for i := 0; i < 5; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		due, err := s.rows.DueSitemaps(ctx, p.ID, property, now.Unix(), 1)
		if err != nil {
			return err
		}
		if len(due) == 0 {
			return nil
		}
		scan := due[0]
		part, cancel := context.WithTimeout(ctx, 15*time.Second)
		res, fetchErr := s.fetchDiscovery(part, property, scan.URL)
		var doc sitemap.Document
		if fetchErr == nil {
			doc, fetchErr = sitemap.Parse(res.Body)
			res.Body.Close()
		}
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		failure := ""
		if fetchErr != nil {
			failure = fetchErr.Error()
		}
		scan.IsIndex = doc.Index
		var children []string
		var urls []model.IndexURL
		var sources []model.SitemapURL
		seen := map[string]bool{}
		if fetchErr == nil {
			for _, entry := range doc.Entries {
				raw := entry.URL
				if seen[raw] {
					continue
				}
				seen[raw] = true
				if doc.Index {
					if sitemapAllowed(property, raw) {
						children = append(children, raw)
					}
					continue
				}
				if !urlInProperty(property, raw) {
					continue
				}
				// Extension labels describe URL shape, not fetched MIME type.
				urls = append(urls, model.IndexURL{ProjectID: p.ID, Property: property, URL: raw, FromSitemap: true, ContentGroup: contentGroup(raw), FirstSeenAt: now.Unix(), LastSeenAt: now.Unix()})
				sources = append(sources, model.SitemapURL{ProjectID: p.ID, SitemapID: scan.ID, KeyHash: model.RowKey(property, scan.URL, raw), URLKey: model.RowKey(property, raw), URL: raw, LastModified: entry.LastModified, Present: true, SeenAt: now.Unix()})
			}
		}
		if err := s.rows.SaveSitemap(ctx, scan, children, urls, sources, now.Unix(), failure); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) AddSitemap(ctx context.Context, slug, raw string) error {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	property := s.indexProperty(ctx, p)
	u, err := url.Parse(raw)
	if err != nil || !sitemapAllowed(property, raw) || u.Fragment != "" || path.Ext(u.Path) == "" {
		return fmt.Errorf("invalid sitemap URL for this property")
	}
	return s.rows.QueueSitemaps(ctx, p.ID, property, []string{raw})
}
