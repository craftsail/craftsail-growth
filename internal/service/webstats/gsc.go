// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const defaultGSCRowLimit = 25000

type gscQueryPage struct {
	Aggregation string      `json:"responseAggregationType"`
	Rows        []gscAPIRow `json:"rows"`
}

type gscAPIRow struct {
	Keys        []string `json:"keys"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	CTR         float64  `json:"ctr"`
	Position    float64  `json:"position"`
}

// FetchGSC pages searchAnalytics for site over [start, end] (YYYY-MM-DD).
// An empty site returns (nil, nil). ProjectID and KeyHash are left unset.
func (c *Client) gscRowLimit() int {
	if c != nil && c.RowLimit > 0 {
		return c.RowLimit
	}
	return defaultGSCRowLimit
}

func gscQueryURL(site string) string {
	// QueryEscape encodes ':'; PathEscape does not, and GSC returns 400.
	return "https://searchconsole.googleapis.com/webmasters/v3/sites/" + url.QueryEscape(site) + "/searchAnalytics/query"
}

func (c *Client) postJSON(ctx context.Context, token, rawURL, api string, body any) ([]byte, error) {
	if err := takeSyncBudget(ctx, true); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	cli, err := c.httpClient()
	if err != nil {
		return nil, err
	}
	res, err := cli.Do(req)
	if err != nil {
		return nil, syncBudgetError(ctx, fmt.Errorf("%s request: %w", api, err))
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
