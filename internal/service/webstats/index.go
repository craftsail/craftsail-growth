// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
)

const indexInspectCap = 30

type gscSitemapList struct {
	Sitemap []gscSitemapItem `json:"sitemap"`
}

type gscSitemapItem struct {
	IsIndex   bool           `json:"isSitemapsIndex"`
	Path      string         `json:"path"`
	IsPending bool           `json:"isPending"`
	Errors    flexFloat      `json:"errors"`
	Warnings  flexFloat      `json:"warnings"`
	Contents  []gscSitemapCt `json:"contents"`
}

type gscSitemapCt struct {
	Submitted flexFloat `json:"submitted"`
}

type inspectBody struct {
	InspectionURL string `json:"inspectionUrl"`
	SiteURL       string `json:"siteUrl"`
	LanguageCode  string `json:"languageCode"`
}

type inspectResponse struct {
	InspectionResult struct {
		IndexStatusResult struct {
			Verdict         string `json:"verdict"`
			GoogleCanonical string `json:"googleCanonical"`
			UserCanonical   string `json:"userCanonical"`
			RobotsTxtState  string `json:"robotsTxtState"`
			PageFetchState  string `json:"pageFetchState"`
			CoverageState   string `json:"coverageState"`
			IndexingState   string `json:"indexingState"`
			LastCrawlTime   string `json:"lastCrawlTime"`
		} `json:"indexStatusResult"`
	} `json:"inspectionResult"`
}

type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(n)
	return nil
}

// urlInProperty reports whether page belongs to a Search Console property.
// sc-domain covers the domain and its subdomains. A URL-prefix property must be a prefix of the page.
func urlInProperty(site, page string) bool {
	site = strings.TrimSpace(site)
	page = strings.TrimSpace(page)
	u, err := url.Parse(page)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return false
	}
	if strings.HasPrefix(strings.ToLower(site), "sc-domain:") {
		domain := strings.TrimPrefix(strings.ToLower(site), "sc-domain:")
		host := strings.ToLower(u.Hostname())
		return host == domain || strings.HasSuffix(host, "."+domain)
	}
	prop, err := url.Parse(site)
	if err != nil || prop.Host == "" {
		return false
	}
	if !strings.EqualFold(u.Scheme, prop.Scheme) || !strings.EqualFold(u.Host, prop.Host) {
		return false
	}
	prefix := prop.EscapedPath()
	if prefix == "" || prefix == "/" {
		return true
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	prefix = strings.TrimRight(prefix, "/")
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

type gscSiteList struct {
	SiteEntry []struct {
		SiteURL string `json:"siteUrl"`
	} `json:"siteEntry"`
}

func (c *Client) FetchSites(token string) ([]string, error) {
	return c.FetchSitesContext(context.Background(), token)
}
func (c *Client) FetchSitesContext(ctx context.Context, token string) ([]string, error) {
	b, err := c.getJSONContext(ctx, token, "https://www.googleapis.com/webmasters/v3/sites", "gsc")
	if err != nil {
		return nil, err
	}
	var list gscSiteList
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("gsc sites: %w", err)
	}
	out := make([]string, 0, len(list.SiteEntry))
	for _, e := range list.SiteEntry {
		if s := strings.TrimSpace(e.SiteURL); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// ResolveGSCSite picks the Search Console property for a project site.
// A hand-filled property is kept only when it is the same host. Otherwise the official site is enough.
func ResolveGSCSite(granted []string, projectSite, explicit string) string {
	host := siteHost(projectSite)
	if host == "" {
		host = siteHost(explicit)
	}
	if explicit = strings.TrimSpace(explicit); explicit != "" && sameSite(explicit, host) {
		return explicit
	}
	if match := pickGranted(granted, host, projectSite); match != "" {
		return match
	}
	if host != "" {
		return "sc-domain:" + host
	}
	return explicit
}

func pickGranted(granted []string, host, projectSite string) string {
	var domain, prefix string
	for _, raw := range granted {
		s := strings.TrimSpace(raw)
		if strings.EqualFold(s, "sc-domain:"+host) {
			domain = s
		}
		if urlInProperty(s, projectSite) || urlInProperty(s, "https://"+host+"/") {
			if !strings.HasPrefix(strings.ToLower(s), "sc-domain:") {
				prefix = s
			}
		}
	}
	if domain != "" {
		return domain
	}
	return prefix
}

func siteHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "sc-domain:") {
		return strings.TrimPrefix(strings.ToLower(raw), "sc-domain:")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func sameSite(property, host string) bool {
	if host == "" {
		return strings.TrimSpace(property) != ""
	}
	return siteHost(property) == host
}

func (c *Client) FetchSitemaps(token, site string) ([]model.GscSitemap, error) {
	return c.FetchSitemapsContext(context.Background(), token, site)
}
func (c *Client) FetchSitemapsContext(ctx context.Context, token, site string) ([]model.GscSitemap, error) {
	if strings.TrimSpace(site) == "" {
		return nil, nil
	}
	rawURL := "https://www.googleapis.com/webmasters/v3/sites/" + url.QueryEscape(site) + "/sitemaps"
	b, err := c.getJSONContext(ctx, token, rawURL, "gsc")
	if err != nil {
		return nil, err
	}
	var list gscSitemapList
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("gsc sitemaps: %w", err)
	}
	out := make([]model.GscSitemap, 0, len(list.Sitemap))
	raw := string(b)
	for _, item := range list.Sitemap {
		var submitted float64
		for _, ct := range item.Contents {
			submitted += float64(ct.Submitted)
		}
		out = append(out, model.GscSitemap{
			Path:      item.Path,
			Property:  site,
			IsIndex:   item.IsIndex,
			Submitted: submitted,
			Errors:    float64(item.Errors),
			Warnings:  float64(item.Warnings),
			Pending:   item.IsPending,
			Raw:       raw,
		})
	}
	return out, nil
}

func (c *Client) InspectURL(token, site, page string) (model.GscIndex, error) {
	return c.InspectURLContext(context.Background(), token, site, page)
}
func (c *Client) InspectURLContext(ctx context.Context, token, site, page string) (model.GscIndex, error) {
	b, err := c.postJSON(ctx, token, "https://searchconsole.googleapis.com/v1/urlInspection/index:inspect", "gsc", inspectBody{
		InspectionURL: page,
		SiteURL:       site,
		LanguageCode:  "en-US",
	})
	if err != nil {
		return model.GscIndex{}, err
	}
	var res inspectResponse
	if err := json.Unmarshal(b, &res); err != nil {
		return model.GscIndex{}, fmt.Errorf("gsc inspect: %w", err)
	}
	st := res.InspectionResult.IndexStatusResult
	if st.Verdict == "" {
		return model.GscIndex{}, fmt.Errorf("gsc inspect: missing index status verdict")
	}
	return model.GscIndex{
		URL:             page,
		Property:        site,
		GoogleCanonical: st.GoogleCanonical,
		UserCanonical:   st.UserCanonical,
		RobotsTxtState:  st.RobotsTxtState,
		PageFetchState:  st.PageFetchState,
		Verdict:         st.Verdict,
		CoverageState:   st.CoverageState,
		IndexingState:   st.IndexingState,
		LastCrawl:       st.LastCrawlTime,
		Raw:             string(b),
	}, nil
}

func (c *Client) UserInfo(token string) ([]byte, error) {
	return c.getJSON(token, "https://www.googleapis.com/oauth2/v2/userinfo", "gsc")
}

func (c *Client) getJSON(token, rawURL, api string) ([]byte, error) {
	return c.getJSONContext(context.Background(), token, rawURL, api)
}
func (c *Client) getJSONContext(ctx context.Context, token, rawURL, api string) ([]byte, error) {
	if err := takeSyncBudget(ctx, true); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	cli, err := c.httpClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request: %w", api, err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, apiError(api, res.StatusCode, b)
	}
	return b, nil
}
