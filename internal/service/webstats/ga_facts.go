// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

const defaultGAFactLimit = 100000

type gaFactSpec struct {
	Dimensions []string
	Metrics    []string
}

func gaFactSpecByName(report string) (gaFactSpec, bool) {
	switch report {
	case "channel":
		return gaFactSpec{Dimensions: []string{"date", "sessionDefaultChannelGroup", "sessionSource", "sessionMedium"}, Metrics: []string{"sessions", "engagedSessions", "keyEvents", "userEngagementDuration"}}, true
	case "landing":
		return gaFactSpec{Dimensions: []string{"date", "landingPagePlusQueryString"}, Metrics: []string{"sessions", "engagedSessions", "keyEvents", "userEngagementDuration"}}, true
	case "session":
		return gaFactSpec{
			Dimensions: []string{"date", "sessionSource", "sessionMedium", "sessionCampaignName", "sessionDefaultChannelGroup", "landingPagePlusQueryString", "country", "deviceCategory"},
			Metrics:    []string{"sessions", "engagedSessions", "activeUsers", "newUsers", "screenPageViews", "keyEvents", "totalRevenue", "engagementRate", "bounceRate", "userEngagementDuration"},
		}, true
	case "page":
		return gaFactSpec{
			Dimensions: []string{"date", "pagePath", "pageTitle", "country", "deviceCategory"},
			Metrics:    []string{"screenPageViews", "sessions", "engagedSessions", "activeUsers", "keyEvents"},
		}, true
	case "event":
		return gaFactSpec{
			Dimensions: []string{"date", "eventName"},
			Metrics:    []string{"eventCount", "eventValue", "keyEvents"},
		}, true
	case "hour":
		return gaFactSpec{
			Dimensions: []string{"dateHour", "sessionSource", "landingPagePlusQueryString"},
			Metrics:    []string{"sessions", "engagedSessions", "screenPageViews"},
		}, true
	default:
		return gaFactSpec{}, false
	}
}

func (c *Client) gaFactLimit() int {
	if c != nil && c.GALimit > 0 {
		return c.GALimit
	}
	return defaultGAFactLimit
}

func (c *Client) FetchGAFacts(ctx context.Context, token, property, report, start, end string) ([]model.GaFact, error) {
	rows, _, err := c.FetchGAReport(ctx, token, property, report, start, end)
	return rows, err
}

func (c *Client) FetchGAReport(ctx context.Context, token, property, report, start, end string) ([]model.GaFact, model.GoogleQuality, error) {
	quality := model.GoogleQuality{Known: true}
	id := strings.TrimPrefix(strings.TrimSpace(property), "properties/")
	if id == "" {
		return nil, quality, nil
	}
	spec, ok := gaFactSpecByName(report)
	if !ok {
		return nil, quality, fmt.Errorf("unknown ga report %s", report)
	}
	limit := c.gaFactLimit()
	endpoint := "https://analyticsdata.googleapis.com/v1beta/properties/" + id + ":runReport"
	var out []model.GaFact
	expected := 0
	for offset := 0; ; {
		body := gaReportBody{
			DateRanges:          []gaDateRange{{StartDate: start, EndDate: end}},
			Dimensions:          names(spec.Dimensions),
			Metrics:             names(spec.Metrics),
			Limit:               limit,
			Offset:              offset,
			ReturnPropertyQuota: true,
		}
		b, err := c.postJSON(ctx, token, endpoint, "ga", body)
		if err != nil {
			return nil, quality, err
		}
		if c != nil && c.OnPage != nil {
			req, _ := json.Marshal(body)
			c.OnPage(report, string(req), string(b))
		}
		var page gaFactPage
		if err := json.Unmarshal(b, &page); err != nil {
			return nil, quality, fmt.Errorf("ga facts: %w", err)
		}
		mergeQuality(&quality, qualityFromGA(page.Metadata))
		if page.Metadata != nil && (len(page.Metadata.Truncation) > 0 || page.Metadata.EmptyReason != "") {
			return nil, quality, ErrIncompleteReport
		}
		if page.RowCount > expected {
			expected = page.RowCount
		}
		if report == "channel" || report == "landing" {
			if page.Metadata != nil {
				for _, raw := range page.Metadata.Restrictions.Metrics {
					var restriction struct {
						Name string `json:"metricName"`
					}
					if err := json.Unmarshal(raw, &restriction); err != nil {
						return nil, quality, err
					}
					for _, metric := range spec.Metrics {
						if restriction.Name == metric {
							return nil, quality, ErrMetricRestricted
						}
					}
				}
			}
		}
		for _, row := range page.Rows {
			if report == "channel" || report == "landing" {
				if len(row.DimensionValues) != len(spec.Dimensions) || len(row.MetricValues) != len(spec.Metrics) {
					return nil, quality, ErrIncompleteReport
				}
				for _, v := range row.MetricValues {
					n, err := strconv.ParseFloat(v.Value, 64)
					if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
						return nil, quality, ErrIncompleteReport
					}
				}
			}
			out = append(out, factFromGA(report, spec, row))
		}
		if len(page.Rows) < limit {
			if len(out) < expected {
				return nil, quality, ErrIncompleteReport
			}
			return out, quality, nil
		}
		offset += len(page.Rows)
	}
}

type gaFactPage struct {
	Metadata *gaMetadata `json:"metadata"`
	Rows     []gaAPIRow  `json:"rows"`
	RowCount int         `json:"rowCount"`
}

func names(in []string) []gaName {
	out := make([]gaName, len(in))
	for i, n := range in {
		out[i] = gaName{Name: n}
	}
	return out
}

func factFromGA(report string, spec gaFactSpec, row gaAPIRow) model.GaFact {
	fact := model.GaFact{Report: report}
	for i, dim := range spec.Dimensions {
		if i >= len(row.DimensionValues) {
			break
		}
		val := row.DimensionValues[i].Value
		switch dim {
		case "date":
			fact.Day, _ = time.ParseInLocation("20060102", val, time.UTC)
		case "dateHour":
			if len(val) >= 8 {
				fact.Day, _ = time.ParseInLocation("20060102", val[:8], time.UTC)
			}
			if len(val) >= 10 {
				fact.Hour = val[8:10]
			}
		case "sessionSource":
			fact.Source = val
		case "sessionMedium":
			fact.Medium = val
		case "sessionCampaignName":
			fact.Campaign = val
		case "sessionDefaultChannelGroup":
			fact.Channel = val
		case "landingPage", "landingPagePlusQueryString":
			fact.Landing = val
		case "pagePath":
			fact.PagePath = val
		case "pageTitle":
			fact.PageTitle = val
		case "country":
			fact.Country = val
		case "deviceCategory":
			fact.Device = val
		case "eventName":
			fact.EventName = val
		}
	}
	for i, metric := range spec.Metrics {
		if i >= len(row.MetricValues) {
			break
		}
		n := parseFloat(row.MetricValues[i].Value)
		switch metric {
		case "sessions":
			fact.Sessions = n
		case "engagedSessions":
			fact.Engaged = n
		case "activeUsers":
			fact.ActiveUsers = n
		case "newUsers":
			fact.NewUsers = n
		case "screenPageViews":
			fact.Views = n
		case "eventCount":
			fact.EventCount = n
		case "eventValue":
			fact.EventValue = metricPtr(row, spec.Metrics, metric)
		case "userEngagementDuration":
			fact.EngagementDuration = metricPtr(row, spec.Metrics, metric)
		case "keyEvents":
			fact.KeyEvents = n
		case "totalRevenue":
			fact.Revenue = n
		case "engagementRate":
			fact.EngagementRate = n
		case "bounceRate":
			fact.BounceRate = n
		}
	}
	return fact
}

func parseFloat(s string) float64 {
	n, _ := strconv.ParseFloat(s, 64)
	return n
}
