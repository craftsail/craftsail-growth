// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type gaAccountList struct {
	AccountSummaries []struct {
		PropertySummaries []struct {
			Property    string `json:"property"`
			DisplayName string `json:"displayName"`
		} `json:"propertySummaries"`
	} `json:"accountSummaries"`
}

type gaStreamList struct {
	DataStreams []struct {
		WebStreamData struct {
			DefaultURI string `json:"defaultUri"`
		} `json:"webStreamData"`
	} `json:"dataStreams"`
}

// Discover binds a project site to a Search Console property and GA4 property the token can access.
func (c *Client) Discover(token, projectSite string) (gscSite, gaProperty string, err error) {
	host := siteHost(projectSite)
	sites, err := c.FetchSites(token)
	if err != nil {
		return "", "", err
	}
	gscSite = ResolveGSCSite(sites, projectSite, "")
	props, listErr := c.listGAProperties(token)
	if listErr != nil {
		return gscSite, "", listErr
	}
	for _, prop := range props {
		if len(props) > 15 && gaProperty != "" {
			break
		}
		streams, err := c.listStreams(token, prop.ID)
		if err != nil {
			continue
		}
		for _, uri := range streams {
			if host != "" && siteHost(uri) == host {
				return gscSite, prop.ID, nil
			}
		}
		if gaProperty == "" && host != "" && strings.Contains(strings.ToLower(prop.Name), host) {
			gaProperty = prop.ID
		}
	}
	return gscSite, gaProperty, nil
}

type gaProp struct {
	ID   string
	Name string
}

func (c *Client) listGAProperties(token string) ([]gaProp, error) {
	b, err := c.getJSON(token, "https://analyticsadmin.googleapis.com/v1beta/accountSummaries?pageSize=50", "ga")
	if err != nil {
		return nil, err
	}
	var list gaAccountList
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, fmt.Errorf("ga accounts: %w", err)
	}
	var out []gaProp
	for _, acc := range list.AccountSummaries {
		for _, p := range acc.PropertySummaries {
			id := strings.TrimPrefix(p.Property, "properties/")
			if id != "" {
				out = append(out, gaProp{ID: id, Name: p.DisplayName})
			}
		}
	}
	return out, nil
}

func (c *Client) listStreams(token, propertyID string) ([]string, error) {
	raw := "https://analyticsadmin.googleapis.com/v1beta/properties/" + url.PathEscape(propertyID) + "/dataStreams?pageSize=20"
	b, err := c.getJSON(token, raw, "ga")
	if err != nil {
		return nil, err
	}
	var list gaStreamList
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	var out []string
	for _, s := range list.DataStreams {
		if s.WebStreamData.DefaultURI != "" {
			out = append(out, s.WebStreamData.DefaultURI)
		}
	}
	return out, nil
}
