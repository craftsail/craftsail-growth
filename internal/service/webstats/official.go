// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

const officialWindowDays = 28
const officialTailDays = 3

type gscDatePage struct {
	Rows     []gscAPIRow `json:"rows"`
	Metadata struct {
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
	if strings.TrimSpace(site) == "" {
		return nil, "", 0, false, nil
	}
	limit := c.gscRowLimit()
	body := gscFactBody{
		StartDate: start, EndDate: end,
		Dimensions: []string{"date"},
		Type:       "web",
		RowLimit:   limit,
		DataState:  "all",
	}
	raw, err := c.postJSON(ctx, token, gscQueryURL(site), "gsc", body)
	if err != nil {
		return nil, "", 0, false, err
	}
	var page gscDatePage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, "", 0, false, fmt.Errorf("gsc date: %w", err)
	}
	facts := make([]model.GscFact, 0, len(page.Rows))
	for _, row := range page.Rows {
		facts = append(facts, factFromGSC("date", "web", []string{"date"}, row))
	}
	return facts, page.incomplete(), len(page.Rows), len(page.Rows) >= limit, nil
}

func (c *Client) FetchGADate(ctx context.Context, token, property, start, end string) ([]model.GaDaily, int, bool, error) {
	id, ok := GAPropertyKey(property)
	if !ok {
		return nil, 0, false, fmt.Errorf("ga HTTP 400 invalid property")
	}
	metrics := []string{"sessions", "engagedSessions", "keyEvents"}
	limit := c.gaFactLimit()
	endpoint := "https://analyticsdata.googleapis.com/v1beta/properties/" + id + ":runReport"
	body := gaReportBody{
		DateRanges: []gaDateRange{{StartDate: start, EndDate: end}},
		Dimensions: names([]string{"date"}),
		Metrics:    names(metrics),
		Limit:      limit,
	}
	raw, err := c.postJSON(ctx, token, endpoint, "ga", body)
	if err != nil {
		return nil, 0, false, err
	}
	var page gaFactPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, 0, false, fmt.Errorf("ga date: %w", err)
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
	return out, len(page.Rows), len(page.Rows) >= limit, nil
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
		n := parseFloat(raw)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	cli, err := c.httpClient()
	if err != nil {
		return ""
	}
	res, err := cli.Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 400 {
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

// officialPlan fills the visible 56-day report first. Later syncs refresh the tail and walk one older month.
func officialPlan(lastEnd, earliest, endCap, historyStart time.Time) []dateChunk {
	endCap = dateOnly(endCap)
	historyStart = dateOnly(historyStart)
	if endCap.Before(historyStart) {
		return nil
	}
	reportStart := endCap.AddDate(0, 0, -(officialWindowDays*2 - 1))
	if reportStart.Before(historyStart) {
		reportStart = historyStart
	}
	if lastEnd.IsZero() {
		return []dateChunk{{start: reportStart, end: endCap}}
	}
	var out []dateChunk
	tailStart := endCap.AddDate(0, 0, -(officialTailDays - 1))
	if tailStart.Before(historyStart) {
		tailStart = historyStart
	}
	if !tailStart.After(endCap) {
		out = append(out, dateChunk{start: tailStart, end: endCap})
	}
	floor := dateOnly(earliest)
	if floor.IsZero() {
		floor = reportStart
	}
	if floor.After(historyStart) {
		olderEnd := floor.AddDate(0, 0, -1)
		olderStart := olderEnd.AddDate(0, 0, -27)
		if olderStart.Before(historyStart) {
			olderStart = historyStart
		}
		if !olderStart.After(olderEnd) {
			out = append(out, dateChunk{start: olderStart, end: olderEnd})
		}
	}
	return out
}

func (s *Service) SyncOfficial(ctx context.Context, projectID uint64, token, gscSite, gaProp string, now time.Time) string {
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
	key, ok := GSCPropertyKey(site)
	if !ok {
		s.saveImport(ctx, projectID, "gsc", site, "paused", "config_invalid", "config_invalid", "Choose the property again", nil, "")
		return "Choose the Search Console property again"
	}
	if err := s.rows.ActivateProperty(ctx, projectID, "gsc", key, ""); err != nil {
		return err.Error()
	}
	probeEnd := now.UTC().Format("2006-01-02")
	probeStart := now.UTC().AddDate(0, 0, -7).Format("2006-01-02")
	_, incomplete, _, _, err := s.fetchGSCDate(ctx, token, key, probeStart, probeEnd)
	if err != nil {
		return s.failImport(ctx, projectID, "gsc", key, err)
	}
	through, source := FinalizedThrough(now, incomplete)
	history := through.AddDate(0, -16, 0)
	last, earliest, err := s.officialCursors(ctx, projectID, "gsc/official/web", "gsc/official/earliest")
	if err != nil {
		return err.Error()
	}
	parts := officialPlan(last, earliest, through, history)
	returned := 0
	capHit := false
	for _, part := range parts {
		facts, _, n, hit, err := s.fetchGSCDate(ctx, token, key, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
		if err != nil {
			return s.failImport(ctx, projectID, "gsc", key, err)
		}
		returned += n
		capHit = capHit || hit
		rows := GSCDailiesFromFacts(key, facts)
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].FetchedAt = now.Unix()
		}
		if err := s.rows.UpsertGscDaily(ctx, rows); err != nil {
			return err.Error()
		}
		if err := s.rows.PutSync(ctx, projectID, "gsc/official/web", part.end); err != nil {
			return err.Error()
		}
		cursor := part.end
		s.saveImport(ctx, projectID, "gsc", key, "running", "", "", "", &cursor, source)
	}
	s.rememberOfficial(ctx, projectID, "gsc/official/web", "gsc/official/earliest", through, parts, earliest)
	s.writeWindow(ctx, projectID, "gsc", key, through, now)
	state := "completed"
	if returned == 0 && last.IsZero() {
		state = "waiting_for_first_data"
	}
	s.saveImport(ctx, projectID, "gsc", key, state, "", "", "", &through, source)
	if capHit {
		return "Search Console daily totals hit the row limit"
	}
	return ""
}

func (s *Service) syncGAOfficial(ctx context.Context, projectID uint64, token, property string, now time.Time) string {
	key, ok := GAPropertyKey(property)
	if !ok {
		s.saveImport(ctx, projectID, "ga4", property, "paused", "config_invalid", "config_invalid", "Choose the property again", nil, "")
		return "Choose the GA4 property again"
	}
	tz := ""
	if s.FetchGADate == nil && s.client() != nil {
		tz = s.client().FetchGATimezone(ctx, token, key)
	}
	if err := s.rows.ActivateProperty(ctx, projectID, "ga4", key, tz); err != nil {
		return err.Error()
	}
	through := gaFinalizedThrough(now, tz)
	history := through.AddDate(0, -16, 0)
	last, earliest, err := s.officialCursors(ctx, projectID, "ga/official/daily", "ga/official/earliest")
	if err != nil {
		return err.Error()
	}
	parts := officialPlan(last, earliest, through, history)
	returned := 0
	for _, part := range parts {
		rows, n, _, err := s.fetchGADate(ctx, token, key, part.start.Format("2006-01-02"), part.end.Format("2006-01-02"))
		if err != nil {
			return s.failImport(ctx, projectID, "ga4", key, err)
		}
		returned += n
		for i := range rows {
			rows[i].ProjectID = projectID
			rows[i].Property = key
			rows[i].Timezone = tz
			rows[i].FetchedAt = now.Unix()
		}
		if err := s.rows.UpsertGaDaily(ctx, rows); err != nil {
			return err.Error()
		}
		if err := s.rows.PutSync(ctx, projectID, "ga/official/daily", part.end); err != nil {
			return err.Error()
		}
		cursor := part.end
		s.saveImport(ctx, projectID, "ga4", key, "running", "", "property", "", &cursor, tz)
	}
	s.rememberOfficial(ctx, projectID, "ga/official/daily", "ga/official/earliest", through, parts, earliest)
	s.writeWindow(ctx, projectID, "ga4", key, through, now)
	state := "completed"
	if returned == 0 && last.IsZero() {
		state = "waiting_for_first_data"
	}
	s.saveImport(ctx, projectID, "ga4", key, state, "", "property", "", &through, tz)
	return ""
}

func (s *Service) officialCursors(ctx context.Context, projectID uint64, endKey, earlyKey string) (time.Time, time.Time, error) {
	last, err := s.rows.GetSync(ctx, projectID, endKey)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	earliest, err := s.rows.GetSync(ctx, projectID, earlyKey)
	return last, earliest, err
}

func (s *Service) rememberOfficial(ctx context.Context, projectID uint64, endKey, earlyKey string, through time.Time, parts []dateChunk, prevEarliest time.Time) {
	_ = s.rows.PutSync(ctx, projectID, endKey, through)
	oldest := dateOnly(prevEarliest)
	for _, part := range parts {
		if oldest.IsZero() || part.start.Before(oldest) {
			oldest = dateOnly(part.start)
		}
	}
	if !oldest.IsZero() {
		_ = s.rows.PutSync(ctx, projectID, earlyKey, oldest)
	}
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

func (s *Service) writeWindow(ctx context.Context, projectID uint64, source, property string, through, now time.Time) {
	from := through.AddDate(0, 0, -(officialWindowDays*2 - 1))
	row := &model.WebWindow{
		ProjectID: projectID, Source: source, Property: property,
		WindowDays: officialWindowDays, FinalizedThrough: through, ComputedAt: now.Unix(),
	}
	if source == "gsc" {
		days, err := s.rows.ListGscDaily(ctx, projectID, property, from, through)
		if err != nil {
			return
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
			return
		}
		win := WindowFromGA(days, through, officialWindowDays)
		row.Sessions = win.Sessions
		row.PreviousSessions = win.PreviousSessions
		row.CoveredDays = win.CoveredDays
	}
	_ = s.rows.UpsertWindow(ctx, row)
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
	class, label := classifyErr(err)
	reason := "error"
	if class == "needs_reauth" || class == "rate_limited" || class == "config_invalid" {
		reason = class
	}
	s.saveImport(ctx, projectID, source, property, "paused", reason, class, publicGoogleError(err), nil, "")
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

func (s *Service) saveImport(ctx context.Context, projectID uint64, source, property, state, reason, class, last string, through *time.Time, boundary string) {
	row := &model.WebImport{
		ProjectID: projectID, Source: source, Property: property,
		State: state, PausedReason: reason, LastErrorClass: class, LastError: last,
		BoundarySource: boundary, UpdatedAt: time.Now().Unix(),
	}
	if through != nil {
		day := dateOnly(*through)
		row.FinalizedThrough = &day
		row.CursorDate = &day
	}
	_ = s.rows.UpsertImport(ctx, row)
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
