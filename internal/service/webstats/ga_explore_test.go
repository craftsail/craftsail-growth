// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGAExploreCoverageQualityAndValidation(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	s.Now = func() time.Time { return time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC) }
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Analytics", URL: "https://a.com/"})
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	if err := project.New(db).Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 3)
	if err := s.syncReport(ctx, p.ID, "ga4", "123", "channel", "", start, end, 7, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		return repo.SyncBatch{Quality: model.GoogleQuality{Known: true, TimeZones: []string{"UTC"}}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	input := ExploreInput{From: "2026-09-20", Through: "2026-09-21"}
	out, err := s.ExploreGA(ctx, p.Slug, "channel", input)
	if err != nil || !out.Comparable {
		t.Fatalf("covered %#v %v", out, err)
	}
	reports, _ := s.rows.SyncReports(ctx, p.ID, "ga4", "123")
	raw, _ := json.Marshal(model.GoogleQuality{Known: true, Sampled: true, TimeZones: []string{"UTC"}})
	if err := db.Model(&model.WebSyncDay{}).Where("report_id = ? AND date(day) = ?", reports[0].ID, "2026-09-18").Update("quality_json", string(raw)).Error; err != nil {
		t.Fatal(err)
	}
	out, err = s.ExploreGA(ctx, p.Slug, "channel", input)
	if err != nil || out.Comparable || out.Quality.Sampled || !out.PreviousQuality.Sampled {
		t.Fatalf("period quality %#v %v", out, err)
	}
	for _, quality := range []model.GoogleQuality{{Known: true}, {Known: true, TimeZones: []string{"America/Sao_Paulo"}}} {
		raw, _ := json.Marshal(quality)
		if err := db.Model(&model.WebSyncDay{}).Where("report_id = ? AND date(day) < ?", reports[0].ID, "2026-09-20").Update("quality_json", string(raw)).Error; err != nil {
			t.Fatal(err)
		}
		result, err := s.ExploreGA(ctx, p.Slug, "channel", input)
		if err != nil || result.Comparable {
			t.Fatalf("missing/mixed timezone %#v %v", result, err)
		}
	}
	out, err = s.ExploreGA(ctx, p.Slug, "landing", input)
	if err != nil || out.Comparable || out.ReportState != "missing" || out.Total != 0 {
		t.Fatalf("no fallback %#v %v", out, err)
	}
	for _, in := range []ExploreInput{{Country: "usa"}, {Device: "mobile"}, {Value: "/"}, {From: "2026-01-01"}, {Page: -1}, {PageSize: 201}, {Sort: "revenue"}, {Direction: "SQL"}} {
		if _, err := s.ExploreGA(ctx, p.Slug, "channel", in); err == nil {
			t.Fatalf("accepted %#v", in)
		}
	}
}
func TestGACompatibility(t *testing.T) {
	for _, state := range []string{"COMPATIBLE", "INCOMPATIBLE", "missing"} {
		t.Run(state, func(t *testing.T) {
			c := &Client{HTTP: &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
				if req.URL.Path != "/v1beta/properties/123:checkCompatibility" {
					t.Fatal(req.URL)
				}
				var body struct{ Dimensions, Metrics []gaName }
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				spec, _ := gaFactSpecByName("landing")
				if len(body.Dimensions) != len(spec.Dimensions) || len(body.Metrics) != len(spec.Metrics) {
					t.Fatalf("body %#v", body)
				}
				response := map[string]any{}
				var dims, metrics []any
				for _, name := range spec.Dimensions {
					dims = append(dims, map[string]any{"dimensionMetadata": map[string]string{"apiName": name}, "compatibility": "COMPATIBLE"})
				}
				for _, name := range spec.Metrics {
					if state != "missing" {
						metrics = append(metrics, map[string]any{"metricMetadata": map[string]string{"apiName": name}, "compatibility": state})
					}
				}
				response["dimensionCompatibilities"] = dims
				response["metricCompatibilities"] = metrics
				b, _ := json.Marshal(response)
				return 200, string(b)
			})}}
			err := c.CheckGACompatibility(context.Background(), "token", "properties/123", "landing")
			if state == "COMPATIBLE" && err != nil || state != "COMPATIBLE" && err == nil {
				t.Fatalf("state %s err %v", state, err)
			}
			if state == "INCOMPATIBLE" && !errors.Is(err, ErrSliceUnsupported) {
				t.Fatal(err)
			}
		})
	}
}
func TestGALandingRejectsMissingAndRestrictedMetrics(t *testing.T) {
	for _, fixture := range []struct {
		metrics, metadata string
		want              error
	}{
		{`[{"value":"10"},{"value":"2"},{"value":"3"},{"value":"20"}]`, `{"timeZone":"UTC"}`, nil},
		{`[{"value":"10"}]`, `{}`, ErrIncompleteReport},
		{`[{"value":"10"},{"value":"2"},{"value":"3"},{"value":"NaN"}]`, `{}`, ErrIncompleteReport},
		{`[{"value":"10"},{"value":"2"},{"value":"3"},{"value":"0"}]`, `{"schemaRestrictionResponse":{"activeMetricRestrictions":[{"metricName":"userEngagementDuration","restrictedMetricTypes":["REVENUE_DATA"]}]}}`, ErrMetricRestricted},
	} {
		c := &Client{HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (int, string) {
			return 200, `{"rowCount":1,"metadata":` + fixture.metadata + `,"rows":[{"dimensionValues":[{"value":"20260920"},{"value":"/A?x=1"}],"metricValues":` + fixture.metrics + `}]}`
		})}}
		rows, _, err := c.FetchGAReport(context.Background(), "token", "123", "landing", "2026-09-20", "2026-09-20")
		if !errors.Is(err, fixture.want) {
			t.Fatalf("err %v want %v", err, fixture.want)
		}
		if fixture.want == nil && (len(rows) != 1 || rows[0].Landing != "/A?x=1" || rows[0].EngagementDuration == nil || *rows[0].EngagementDuration != 20) {
			t.Fatalf("rows %#v", rows)
		}
	}
}

func TestGASyncCompatibilityFailureDoesNotBlockOtherReports(t *testing.T) {
	s := New(testDB(t))
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	checks := 0
	landingRuns := 0
	s.Client = &Client{HTTP: &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
		var body struct{ Dimensions, Metrics []gaName }
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(req.URL.Path, ":checkCompatibility") {
			checks++
			var dims, metrics []any
			state := "COMPATIBLE"
			if len(body.Dimensions) == 4 {
				state = "INCOMPATIBLE"
			}
			for _, d := range body.Dimensions {
				dims = append(dims, map[string]any{"dimensionMetadata": map[string]string{"apiName": d.Name}, "compatibility": state})
			}
			for _, m := range body.Metrics {
				metrics = append(metrics, map[string]any{"metricMetadata": map[string]string{"apiName": m.Name}, "compatibility": "COMPATIBLE"})
			}
			raw, _ := json.Marshal(map[string]any{"dimensionCompatibilities": dims, "metricCompatibilities": metrics})
			return 200, string(raw)
		}
		if len(body.Dimensions) == 4 {
			t.Fatal("unsupported template fetched")
		}
		if len(body.Dimensions) == 2 && body.Dimensions[1].Name == "landingPagePlusQueryString" {
			landingRuns++
		}
		return 200, `{"rows":[],"rowCount":0,"metadata":{"timeZone":"UTC"}}`
	})}}
	if _, err := s.SyncFacts(ctx, 1, "token", "", "123", now); err != nil {
		t.Fatal(err)
	}
	if checks != 2 || landingRuns != 8 {
		t.Fatalf("checks %d landing batches %d", checks, landingRuns)
	}
	reports, err := s.rows.SyncReports(ctx, 1, "ga4", "123")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Report == "channel" && r.State != "unsupported" {
			t.Fatalf("unsupported %#v", r)
		}
		if r.Report == "landing" && r.State != "backfilling" {
			t.Fatalf("landing %#v", r)
		}
	}
}
