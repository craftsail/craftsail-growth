// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"net/url"
	"sort"
	"strings"
)

type PageMapping struct {
	State       string             `json:"state"`
	Landing     string             `json:"landing"`
	Hosts       []repo.LandingHost `json:"hosts"`
	URLs        []string           `json:"urls"`
	GACoverage  GrainCoverage      `json:"ga_coverage"`
	GSCCoverage GrainCoverage      `json:"gsc_coverage"`
}

// Normalize only known tracking parameters. Content parameters, case and trailing
// slashes stay significant. Scheme variants remain separate raw candidates.
func pageIdentity(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ""
	}
	for k := range q {
		v := strings.ToLower(k)
		if strings.HasPrefix(v, "utm_") || v == "gclid" || v == "dclid" || v == "fbclid" || v == "msclkid" || v == "_ga" {
			q.Del(k)
		}
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	key := strings.ToLower(u.Host) + path
	if len(q) > 0 {
		key += "?" + q.Encode()
	}
	return key
}
func (s *Service) MapLanding(ctx context.Context, slug string, in ExploreInput) (*PageMapping, error) {
	landing := in.Value
	if landing == "" || len(landing) > 8192 {
		return nil, invalidSearch("landing page is required")
	}
	in.Value = ""
	in.Events = ""
	in.Country = ""
	in.Device = ""
	f, _, err := s.analyticsFilter(ctx, slug, "landing", in)
	if err != nil {
		return nil, err
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := &PageMapping{State: "unmatched", Landing: landing, URLs: []string{}}
	out.GACoverage, err = s.reportCoverage(ctx, p.ID, "ga4", f.Property, "landing_context", "", f.From, f.Through)
	if err != nil {
		return nil, err
	}
	prop := s.officialProperty(ctx, p, "gsc")
	out.GSCCoverage, err = s.grainCoverage(ctx, p.ID, prop, "page", f.From, f.Through)
	if err != nil {
		return nil, err
	}
	out.Hosts, err = s.rows.LandingHosts(ctx, p.ID, f.Property, landing, f.From, f.Through)
	if err != nil {
		return nil, err
	}
	candidates, err := s.rows.SearchPageURLs(ctx, p.ID, prop, f.From, f.Through)
	if err != nil {
		return nil, err
	}
	keys := map[string]bool{}
	hosts := map[string]bool{}
	for _, row := range out.Hosts {
		host := strings.ToLower(row.Hostname)
		if host == "" || strings.ContainsAny(host, "/?#@ ") || !strings.HasPrefix(landing, "/") || strings.HasPrefix(landing, "//") {
			continue
		}
		key := pageIdentity("https://" + host + landing)
		if key != "" {
			keys[key] = true
			hosts[host] = true
		}
	}
	for _, raw := range candidates {
		if keys[pageIdentity(raw)] {
			out.URLs = append(out.URLs, raw)
		}
	}
	sort.Strings(out.URLs)
	if len(out.URLs) == 1 {
		out.State = "candidate"
	}
	if len(out.URLs) > 1 || len(hosts) > 1 {
		out.State = "ambiguous"
	}
	q := out.GACoverage.Quality
	if out.GACoverage.State != "covered" || !q.Known || q.Sampled || q.Thresholded || q.OtherRow || q.Restricted || q.EmptyReason || !trustedSearch(out.GSCCoverage) || len(out.Hosts) > 200 {
		out.State = "unverified"
	}
	return out, nil
}
