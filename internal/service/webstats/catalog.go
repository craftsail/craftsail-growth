// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"encoding/json"
	"strings"
)

type GSCChoice struct {
	SiteURL    string `json:"site_url"`
	Level      string `json:"level"`
	Selectable bool   `json:"selectable"`
	Suggested  bool   `json:"suggested"`
}

type GAChoice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Suggested bool   `json:"suggested"`
}

type gscSiteEntry struct {
	SiteURL         string `json:"siteUrl"`
	PermissionLevel string `json:"permissionLevel"`
}

type gscSitePayload struct {
	SiteEntry []gscSiteEntry `json:"siteEntry"`
}

func ParseGSCSites(body []byte) []GSCChoice {
	var list gscSitePayload
	if err := json.Unmarshal(body, &list); err != nil {
		return nil
	}
	out := make([]GSCChoice, 0, len(list.SiteEntry))
	for _, e := range list.SiteEntry {
		site := strings.TrimSpace(e.SiteURL)
		if site == "" {
			continue
		}
		out = append(out, GSCChoice{
			SiteURL:    site,
			Level:      e.PermissionLevel,
			Selectable: e.PermissionLevel != "siteUnverifiedUser",
		})
	}
	return out
}

func (c *Client) ListChoices(token, projectSite string) (gsc []GSCChoice, ga []GAChoice, err error) {
	b, err := c.getJSON(token, "https://www.googleapis.com/webmasters/v3/sites", "gsc")
	if err != nil {
		return nil, nil, err
	}
	gsc = ParseGSCSites(b)
	if gsc == nil {
		gsc = []GSCChoice{}
	}
	host := siteHost(projectSite)
	for i := range gsc {
		if host != "" && urlInProperty(gsc[i].SiteURL, projectSite) {
			gsc[i].Suggested = true
		}
	}
	props, listErr := c.listGAProperties(token)
	if listErr != nil {
		return gsc, []GAChoice{}, listErr
	}
	ga = make([]GAChoice, 0, len(props))
	for i, prop := range props {
		choice := GAChoice{ID: prop.ID, Name: prop.Name}
		if host != "" && i < 20 {
			streams, err := c.listStreams(token, prop.ID)
			if err == nil {
				for _, uri := range streams {
					if siteHost(uri) == host {
						choice.Suggested = true
						break
					}
				}
			}
		}
		ga = append(ga, choice)
	}
	return gsc, ga, nil
}
