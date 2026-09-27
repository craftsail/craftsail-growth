// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

type gaReportBody struct {
	DateRanges []gaDateRange `json:"dateRanges"`
	Dimensions []gaName      `json:"dimensions"`
	Metrics    []gaName      `json:"metrics"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
}

type gaDateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type gaName struct {
	Name string `json:"name"`
}

type gaAPIRow struct {
	DimensionValues []gaValue `json:"dimensionValues"`
	MetricValues    []gaValue `json:"metricValues"`
}

type gaValue struct {
	Value string `json:"value"`
}
