// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"fmt"
)

// CheckGACompatibility validates the actual property, including unavailable
// metrics, before spending a backfill budget. An incomplete response is an
// error, never evidence that a template is supported.
func (c *Client) CheckGACompatibility(ctx context.Context, token, property, report string) error {
	id, ok := GAPropertyKey(property)
	if !ok {
		return fmt.Errorf("ga HTTP 400 invalid property")
	}
	spec, ok := gaFactSpecByName(report)
	if !ok {
		return fmt.Errorf("unknown ga report %s", report)
	}
	body := struct {
		Dimensions []gaName `json:"dimensions"`
		Metrics    []gaName `json:"metrics"`
	}{names(spec.Dimensions), names(spec.Metrics)}
	raw, err := c.postJSON(ctx, token, "https://analyticsdata.googleapis.com/v1beta/properties/"+id+":checkCompatibility", "ga", body)
	if err != nil {
		return err
	}
	if c.OnPage != nil {
		request, _ := json.Marshal(body)
		c.OnPage("ga4/compatibility/"+report, string(request), string(raw))
	}
	var response struct {
		Dimensions []struct {
			Metadata struct {
				Name string `json:"apiName"`
			} `json:"dimensionMetadata"`
			Compatibility string `json:"compatibility"`
		} `json:"dimensionCompatibilities"`
		Metrics []struct {
			Metadata struct {
				Name string `json:"apiName"`
			} `json:"metricMetadata"`
			Compatibility string `json:"compatibility"`
		} `json:"metricCompatibilities"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return fmt.Errorf("ga compatibility: %w", err)
	}
	dimensions, metrics := map[string]string{}, map[string]string{}
	for _, v := range response.Dimensions {
		dimensions[v.Metadata.Name] = v.Compatibility
	}
	for _, v := range response.Metrics {
		metrics[v.Metadata.Name] = v.Compatibility
	}
	check := func(expected []string, got map[string]string) error {
		for _, name := range expected {
			switch got[name] {
			case "COMPATIBLE":
			case "INCOMPATIBLE":
				return fmt.Errorf("%w: %s", ErrSliceUnsupported, name)
			default:
				return fmt.Errorf("ga compatibility missing result for %s", name)
			}
		}
		return nil
	}
	if err := check(spec.Dimensions, dimensions); err != nil {
		return err
	}
	return check(spec.Metrics, metrics)
}
