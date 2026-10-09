// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

const officialWindowDays = 28

type gscDatePage struct {
	Aggregation string      `json:"responseAggregationType"`
	Rows        []gscAPIRow `json:"rows"`
	Metadata    struct {
		FirstIncompleteDate string `json:"firstIncompleteDate"`
		FirstIncompleteAlt  string `json:"first_incomplete_date"`
	} `json:"metadata"`
}

func (p gscDatePage) incomplete() string {
	if p.Metadata.FirstIncompleteDate != "" {
		return p.Metadata.FirstIncompleteDate
	}
	return p.Metadata.FirstIncompleteAlt
}

// FetchGSCDate requests the web date slice. Totals must come from this response, not from query rows.
func (c *Client) FetchGSCDate(ctx context.Context, token, site, start, end string) ([]model.GscFact, string, int, bool, error) {
	rows, _, incomplete, n, hit, err := c.fetchGSCDateQuality(ctx, token, site, start, end)
	return rows, incomplete, n, hit, err
}

func (c *Client) fetchGSCDateQuality(ctx context.Context, token, site, start, end string) ([]model.GscFact, model.GoogleQuality, string, int, bool, error) {
	quality := model.GoogleQuality{}
	if strings.TrimSpace(site) == "" {
		return nil, quality, "", 0, false, nil
	}
	limit := c.gscRowLimit()
	body := gscFactBody{
		AggregationType: "byProperty", StartDate: start, EndDate: end,
		Dimensions: []string{"date"},
		Type:       "web",
		RowLimit:   limit,
		DataState:  "all",
	}
	raw, err := c.postJSON(ctx, token, gscQueryURL(site), "gsc", body)
	if err != nil {
		return nil, quality, "", 0, false, err
	}
	var page gscDatePage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, quality, "", 0, false, fmt.Errorf("gsc date: %w", err)
	}
	facts := make([]model.GscFact, 0, len(page.Rows))
	for _, row := range page.Rows {
		facts = append(facts, factFromGSC("date", "web", []string{"date"}, row))
	}
	quality.Known = page.Aggregation != ""
	if page.Aggregation != "" {
		quality.Aggregations = []string{page.Aggregation}
	}
	if page.Aggregation != "" && page.Aggregation != "byProperty" {
		return nil, quality, "", 0, false, fmt.Errorf("gsc aggregation mismatch: %s", page.Aggregation)
	}
	return facts, quality, page.incomplete(), len(page.Rows), len(page.Rows) >= limit, nil
}

func (c *Client) FetchGADate(ctx context.Context, token, property, start, end string) ([]model.GaDaily, int, bool, error) {
	rows, _, count, hit, err := c.fetchGADateQuality(ctx, token, property, start, end)
	return rows, count, hit, err
}

func (c *Client) fetchGADateQuality(ctx context.Context, token, property, start, end string) ([]model.GaDaily, model.GoogleQuality, int, bool, error) {
	quality := model.GoogleQuality{}
	id, ok := GAPropertyKey(property)
	if !ok {
		return nil, quality, 0, false, fmt.Errorf("ga HTTP 400 invalid property")
	}
	metrics := []string{"sessions", "engagedSessions", "keyEvents"}
	limit := c.gaFactLimit()
	endpoint := "https://analyticsdata.googleapis.com/v1beta/properties/" + id + ":runReport"
	body := gaReportBody{
		DateRanges:          []gaDateRange{{StartDate: start, EndDate: end}},
		Dimensions:          names([]string{"date"}),
		Metrics:             names(metrics),
		Limit:               limit,
		ReturnPropertyQuota: true,
	}
	raw, err := c.postJSON(ctx, token, endpoint, "ga", body)
	if err != nil {
		return nil, quality, 0, false, err
	}
	if c.OnPage != nil {
		req, _ := json.Marshal(body)
		c.OnPage("ga4/daily", string(req), string(raw))
	}
	var page gaFactPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, quality, 0, false, fmt.Errorf("ga date: %w", err)
	}
	quality = qualityFromGA(page.Metadata)
	if page.Metadata != nil && (len(page.Metadata.Truncation) > 0 || page.Metadata.EmptyReason != "") {
		return nil, quality, 0, true, ErrIncompleteReport
	}
	out := make([]model.GaDaily, 0, len(page.Rows))
	for _, row := range page.Rows {
		fact := factFromGA("daily", gaFactSpec{Dimensions: []string{"date"}, Metrics: metrics}, row)
		out = append(out, model.GaDaily{
			Property:    id,
			Day:         fact.Day,
			Sessions:    fact.Sessions,
			Engaged:     metricPtr(row, metrics, "engagedSessions"),
			KeyEvents:   metricPtr(row, metrics, "keyEvents"),
			Conversions: metricPtr(row, metrics, "conversions"),
		})
	}
	return out, quality, len(page.Rows), len(page.Rows) >= limit || len(page.Rows) < page.RowCount, nil
}

func metricPtr(row gaAPIRow, metrics []string, name string) *float64 {
	for i, metric := range metrics {
		if metric != name || i >= len(row.MetricValues) {
			continue
		}
		raw := strings.TrimSpace(row.MetricValues[i].Value)
		if raw == "" {
			return nil
		}
		n, err := strconv.ParseFloat(raw, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return nil
		}
		return &n
	}
	return nil
}

func (c *Client) FetchGATimezone(ctx context.Context, token, property string) string {
	id, ok := GAPropertyKey(property)
	if !ok || c == nil {
		return ""
	}
	endpoint := "https://analyticsadmin.googleapis.com/v1beta/properties/" + id
	b, err := c.getJSONContext(ctx, token, endpoint, "ga")
	if err != nil {
		return ""
	}

	var doc struct {
		TimeZone string `json:"timeZone"`
	}
	if json.Unmarshal(b, &doc) != nil {
		return ""
	}
	return strings.TrimSpace(doc.TimeZone)
}

func (s *Service) SyncOfficial(ctx context.Context, projectID uint64, token, gscSite, gaProp string, now time.Time) string {
	ctx = s.trafficQuotaContext(ctx, gscSite, gaProp)
	var notes []string
	if strings.TrimSpace(gscSite) != "" {
		if note := s.syncGSCOfficial(ctx, projectID, token, gscSite, now); note != "" {
			notes = append(notes, note)
		}
	}
	if strings.TrimSpace(gaProp) != "" {
		if note := s.syncGAOfficial(ctx, projectID, token, gaProp, now); note != "" {
			notes = append(notes, note)
		}
	}
	return strings.Join(notes, "；")
}

func (s *Service) syncGSCOfficial(ctx context.Context, projectID uint64, token, site string, now time.Time) string {
	if errors.Is(context.Cause(ctx), ErrSyncBudget) {
		return ""
	}
	key, ok := GSCPropertyKey(site)
	if !ok {
		return s.failImport(ctx, projectID, "gsc", site, fmt.Errorf("gsc HTTP 400 invalid property"))
	}
	if err := s.rows.ActivateProperty(ctx, projectID, "gsc", key, ""); err != nil {
		return s.failImport(ctx, projectID, "gsc", key, err)
	}
	_, incomplete, _, _, err := s.fetchGSCDate(ctx, token, key, now.UTC().AddDate(0, 0, -7).Format("2006-01-02"), now.UTC().Format("2006-01-02"))
	if err != nil {
		return s.failImport(ctx, projectID, "gsc", key, err)
	}
	through, boundary := FinalizedThrough(now, incomplete)
	err = s.syncReport(ctx, projectID, "gsc", key, "daily", "web", through.AddDate(0, -16, 0), through, 28, 2, func(partCtx context.Context, part dateChunk) (repo.SyncBatch, error) {
		facts, quality, _, _, hit, err := s.fetchGSCDateQuality(partCtx, token, key, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
		if err != nil {
			return repo.SyncBatch{}, err
		}
		if hit {
			return repo.SyncBatch{}, ErrIncompleteReport
		}
		rows := GSCDailiesFromFacts(key, facts)
		seen := map[string]bool{}
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].FetchedAt = now.Unix()
			seen[rows[i].Day.Format("2006-01-02")] = true
		}
		// A completed date-only response establishes genuine zero days too.
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			if !seen[d.Format("2006-01-02")] {
				rows = append(rows, model.GscDaily{ProjectID: projectID, Property: key, SearchType: "web", Day: d, FetchedAt: now.Unix()})
			}
		}
		return repo.SyncBatch{GSCDaily: rows, Quality: quality}, nil
	})
	if err != nil {
		return s.failImport(ctx, projectID, "gsc", key, err)
	}
	if err := s.writeWindow(ctx, projectID, "gsc", key, through, now); err != nil {
		return s.failImport(ctx, projectID, "gsc", key, err)
	}
	if err := s.saveImport(ctx, projectID, "gsc", key, "completed", "", "", "", &through, boundary); err != nil {
		return err.Error()
	}
	if err := s.promoteSearchStage(ctx, projectID, key, through); err != nil {
		return s.failImport(ctx, projectID, "gsc", key, err)
	}
	return ""
}

func (s *Service) syncGAOfficial(ctx context.Context, projectID uint64, token, property string, now time.Time) string {
	if errors.Is(context.Cause(ctx), ErrSyncBudget) {
		return ""
	}
	key, ok := GAPropertyKey(property)
	if !ok {
		return s.failImport(ctx, projectID, "ga4", property, fmt.Errorf("ga HTTP 400 invalid property"))
	}
	tz := s.propertyTimezone(ctx, projectID, "ga4", key)
	if s.FetchGADate == nil && tz == "" {
		tz = s.client().FetchGATimezone(ctx, token, key)
	}
	if err := s.rows.ActivateProperty(ctx, projectID, "ga4", key, tz); err != nil {
		return s.failImport(ctx, projectID, "ga4", key, err)
	}
	through := gaFinalizedThrough(now, tz)
	err := s.syncReport(ctx, projectID, "ga4", key, "daily", "", through.AddDate(0, -16, 0), through, 28, 2, func(partCtx context.Context, part dateChunk) (repo.SyncBatch, error) {
		var rows []model.GaDaily
		var quality model.GoogleQuality
		var hit bool
		var err error
		if s.FetchGADate != nil {
			rows, _, hit, err = s.fetchGADate(partCtx, token, key, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
		} else {
			rows, quality, _, hit, err = s.client().fetchGADateQuality(partCtx, token, key, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
		}
		if err != nil {
			return repo.SyncBatch{}, err
		}
		if hit {
			return repo.SyncBatch{}, ErrIncompleteReport
		}
		seen := map[string]bool{}
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].Property = key
			rows[i].Timezone = tz
			rows[i].FetchedAt = now.Unix()
			seen[rows[i].Day.Format("2006-01-02")] = true
		}
		for d := part.start; !d.After(part.end); d = d.AddDate(0, 0, 1) {
			if !seen[d.Format("2006-01-02")] && quality.Known && !quality.Sampled && !quality.Thresholded && !quality.OtherRow && !quality.Restricted {
				rows = append(rows, model.GaDaily{ProjectID: projectID, Property: key, Day: d, Timezone: tz, FetchedAt: now.Unix()})
			}
		}
		return repo.SyncBatch{GADaily: rows, Quality: quality}, nil
	})
	if err != nil {
		return s.failImport(ctx, projectID, "ga4", key, err)
	}
	if err := s.writeWindow(ctx, projectID, "ga4", key, through, now); err != nil {
		return s.failImport(ctx, projectID, "ga4", key, err)
	}
	if err := s.saveImport(ctx, projectID, "ga4", key, "completed", "", "", "", &through, "property"); err != nil {
		return err.Error()
	}
	return ""
}

func gaFinalizedThrough(now time.Time, timezone string) time.Time {
	loc := time.UTC
	if timezone != "" {
		if loaded, err := time.LoadLocation(timezone); err == nil {
			loc = loaded
		}
	}
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
}

func (s *Service) writeWindow(ctx context.Context, projectID uint64, source, property string, through, now time.Time) error {
	from := through.AddDate(0, 0, -(officialWindowDays*2 - 1))
	row := &model.WebWindow{
		ProjectID: projectID, Source: source, Property: property,
		WindowDays: officialWindowDays, FinalizedThrough: through, ComputedAt: now.Unix(),
	}
	if source == "gsc" {
		days, err := s.rows.ListGscDaily(ctx, projectID, property, from, through)
		if err != nil {
			return err
		}
		win := WindowFromGSC(days, through, officialWindowDays)
		row.Clicks = win.Clicks
		row.Impressions = win.Impressions
		row.PreviousClicks = win.PreviousClicks
		row.PreviousImpressions = win.PreviousImpressions
		row.CoveredDays = win.CoveredDays
	} else {
		days, err := s.rows.ListGaDaily(ctx, projectID, property, from, through)
		if err != nil {
			return err
		}
		win := WindowFromGA(days, through, officialWindowDays)
		row.Sessions = win.Sessions
		row.PreviousSessions = win.PreviousSessions
		row.CoveredDays = win.CoveredDays
	}
	return s.rows.UpsertWindow(ctx, row)
}

func (s *Service) fetchGSCDate(ctx context.Context, token, site, start, end string) ([]model.GscFact, string, int, bool, error) {
	if s != nil && s.FetchGSCDate != nil {
		return s.FetchGSCDate(ctx, token, site, start, end)
	}
	return s.client().FetchGSCDate(ctx, token, site, start, end)
}

func (s *Service) fetchGADate(ctx context.Context, token, property, start, end string) ([]model.GaDaily, int, bool, error) {
	if s != nil && s.FetchGADate != nil {
		return s.FetchGADate(ctx, token, property, start, end)
	}
	return s.client().FetchGADate(ctx, token, property, start, end)
}

func (s *Service) failImport(ctx context.Context, projectID uint64, source, property string, err error) string {
	if errors.Is(syncBudgetError(ctx, err), ErrSyncBudget) {
		return ""
	}
	class, label := classifyErr(err)
	reason := "error"
	if class == "needs_reauth" || class == "rate_limited" || class == "config_invalid" {
		reason = class
	}
	if saveErr := s.saveImport(ctx, projectID, source, property, "paused", reason, class, publicGoogleError(err), nil, ""); saveErr != nil {
		return label + ": " + saveErr.Error()
	}
	return label
}

func classifyErr(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	msg := err.Error()
	status := 0
	if i := strings.Index(msg, "HTTP "); i >= 0 {
		fmt.Sscanf(msg[i:], "HTTP %d", &status)
	}
	if strings.Contains(strings.ToLower(msg), "invalid_grant") {
		status = 401
	}
	return ClassifyGoogle(status, msg)
}

func publicGoogleError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if i := strings.Index(msg, "{"); i >= 0 {
		msg = msg[:i]
	}
	msg = strings.TrimSpace(msg)
	r := []rune(msg)
	if len(r) > 180 {
		return string(r[:180])
	}
	return msg
}

func (s *Service) saveImport(ctx context.Context, projectID uint64, source, property, state, reason, class, last string, through *time.Time, boundary string) error {
	row := &model.WebImport{
		ProjectID: projectID, Source: source, Property: property,
		State: state, PausedReason: reason, LastErrorClass: class, LastError: last,
		BoundarySource: boundary, UpdatedAt: time.Now().Unix(),
	}
	if through == nil {
		if previous, err := s.rows.GetImport(ctx, projectID, source); err == nil && previous != nil && previous.Property == property {
			row.FinalizedThrough = previous.FinalizedThrough
			row.CursorDate = previous.CursorDate
			row.BoundarySource = previous.BoundarySource
		}
	}

	if through != nil {
		day := dateOnly(*through)
		row.FinalizedThrough = &day
		row.CursorDate = &day
	}
	return s.rows.UpsertImport(ctx, row)
}

func (s *Service) RecordFailure(ctx context.Context, slug string, err error) {
	if err == nil || s == nil || s.projects == nil {
		return
	}
	p, getErr := s.projects.Get(ctx, slug)
	if getErr != nil {
		return
	}
	class, _ := classifyErr(err)
	if class == "provider" && !strings.Contains(err.Error(), "HTTP") && !strings.Contains(strings.ToLower(err.Error()), "invalid_grant") {
		return
	}
	for _, source := range []string{"gsc", "ga4"} {
		property := p.GscSite
		if source == "ga4" {
			property = p.GA4Property
		}
		s.failImport(ctx, p.ID, source, property, err)
	}
}

func (s *Service) fetchGSCDateQuality(ctx context.Context, token, site, start, end string) ([]model.GscFact, model.GoogleQuality, string, int, bool, error) {
	if s.FetchGSCDate != nil {
		rows, incomplete, n, hit, err := s.FetchGSCDate(ctx, token, site, start, end)
		return rows, model.GoogleQuality{}, incomplete, n, hit, err
	}
	return s.client().fetchGSCDateQuality(ctx, token, site, start, end)
}
