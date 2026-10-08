// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"encoding/json"
	"github.com/craftsail/craftsail-growth/internal/model"
	"slices"
)

type gaMetadata struct {
	DataLossFromOtherRow  bool              `json:"dataLossFromOtherRow"`
	SubjectToThresholding bool              `json:"subjectToThresholding"`
	Sampling              []json.RawMessage `json:"samplingMetadatas"`
	Truncation            []json.RawMessage `json:"dataTruncationReasons"`
	Restrictions          struct {
		Metrics []json.RawMessage `json:"activeMetricRestrictions"`
	} `json:"schemaRestrictionResponse"`
	Currency    string `json:"currencyCode"`
	TimeZone    string `json:"timeZone"`
	EmptyReason string `json:"emptyReason"`
}

func qualityFromGA(meta *gaMetadata) model.GoogleQuality {
	if meta == nil {
		return model.GoogleQuality{}
	}
	q := model.GoogleQuality{Known: true, Sampled: len(meta.Sampling) > 0, Thresholded: meta.SubjectToThresholding, OtherRow: meta.DataLossFromOtherRow, Restricted: len(meta.Restrictions.Metrics) > 0, EmptyReason: meta.EmptyReason != ""}
	if meta.Currency != "" {
		q.Currencies = []string{meta.Currency}
	}
	if meta.TimeZone != "" {
		q.TimeZones = []string{meta.TimeZone}
	}
	return q
}

func mergeQuality(dst *model.GoogleQuality, src model.GoogleQuality) {
	// Missing metadata in any contributing response leaves quality unknown.
	dst.Known = dst.Known && src.Known
	dst.Sampled = dst.Sampled || src.Sampled
	dst.Thresholded = dst.Thresholded || src.Thresholded
	dst.OtherRow = dst.OtherRow || src.OtherRow
	dst.Restricted = dst.Restricted || src.Restricted
	dst.EmptyReason = dst.EmptyReason || src.EmptyReason
	merge := func(a, b []string) []string {
		for _, v := range b {
			if !slices.Contains(a, v) {
				a = append(a, v)
			}
		}
		slices.Sort(a)
		return a
	}
	dst.Currencies = merge(dst.Currencies, src.Currencies)
	dst.TimeZones = merge(dst.TimeZones, src.TimeZones)
	dst.Aggregations = merge(dst.Aggregations, src.Aggregations)
}

func comparableGA(a, b GrainCoverage) bool {
	good := func(c GrainCoverage) bool {
		q := c.Quality
		return c.State == "covered" && q.Known && !q.Sampled && !q.Thresholded && !q.OtherRow && !q.Restricted && !q.EmptyReason && len(q.TimeZones) == 1
	}
	return good(a) && good(b) && a.Quality.TimeZones[0] == b.Quality.TimeZones[0]
}
