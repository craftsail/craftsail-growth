// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type gscFactBody struct {
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
	Dimensions []string `json:"dimensions"`
	Type       string   `json:"type"`
	RowLimit   int      `json:"rowLimit"`
	StartRow   int      `json:"startRow"`
	DataState  string   `json:"dataState"`
}

func (c *Client) FetchGSCFacts(ctx context.Context, token, site, slice, searchType, start, end string) ([]model.GscFact, error) {
	if strings.TrimSpace(site) == "" {
		return nil, nil
	}
	spec, ok := gscSliceByName(slice)
	if !ok {
		return nil, fmt.Errorf("unknown gsc slice %s", slice)
	}
	limit := c.gscRowLimit()
	endpoint := "https://www.googleapis.com/webmasters/v3/sites/" + url.QueryEscape(site) + "/searchAnalytics/query"
	var out []model.GscFact
	for startRow := 0; ; {
		body := gscFactBody{
			StartDate:  start,
			EndDate:    end,
			Dimensions: spec.Dimensions,
			Type:       searchType,
			RowLimit:   limit,
			StartRow:   startRow,
			DataState:  spec.DataState,
		}
		b, err := c.postJSON(ctx, token, endpoint, "gsc", body)
		if err != nil {
			if isUnsupportedSlice(err) {
				return nil, ErrSliceUnsupported
			}
			return nil, err
		}
		if c != nil && c.OnPage != nil {
			req, _ := json.Marshal(body)
			c.OnPage(slice+"/"+searchType, string(req), string(b))
		}
		var page gscQueryPage
		if err := json.Unmarshal(b, &page); err != nil {
			return nil, fmt.Errorf("gsc facts: %w", err)
		}
		for _, row := range page.Rows {
			out = append(out, factFromGSC(slice, searchType, spec.Dimensions, row))
		}
		if len(page.Rows) < limit {
			return out, nil
		}
		startRow += len(page.Rows)
	}
}

func factFromGSC(slice, searchType string, dims []string, row gscAPIRow) model.GscFact {
	fact := model.GscFact{Slice: slice, SearchType: searchType, Clicks: row.Clicks, Impressions: row.Impressions, CTR: row.CTR, Position: row.Position}
	for i, dim := range dims {
		if i >= len(row.Keys) {
			break
		}
		val := row.Keys[i]
		switch dim {
		case "date":
			fact.Day, _ = time.ParseInLocation("2006-01-02", val, time.UTC)
		case "hour":
			fact.Hour = val
		case "query":
			fact.Query = val
		case "page":
			fact.Page = val
		case "country":
			fact.Country = val
		case "device":
			fact.Device = val
		case "searchAppearance":
			fact.SearchAppearance = val
		}
	}
	return fact
}

func isUnsupportedSlice(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid value") || strings.Contains(msg, "not supported")
}
