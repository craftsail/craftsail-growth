// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"net/url"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func GSCPropertyKey(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "sc-domain:") {
		host := strings.ToLower(strings.Trim(value[len("sc-domain:"):], "/"))
		if !plainHost(host) {
			return "", false
		}
		return "sc-domain:" + host, true
	}
	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return "", false
		}
		u.Host = strings.ToLower(u.Host)
		if u.Path == "" {
			u.Path = "/"
		} else if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		u.RawQuery = ""
		u.Fragment = ""
		return u.String(), true
	}
	if strings.Contains(value, "/") || strings.Contains(value, " ") || !strings.Contains(value, ".") {
		return "", false
	}
	host := strings.ToLower(strings.Trim(value, "/"))
	if !plainHost(host) {
		return "", false
	}
	return "sc-domain:" + host, true
}

func plainHost(host string) bool {
	return host != "" && strings.Contains(host, ".") && !strings.ContainsAny(host, " /")
}

func GAPropertyKey(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "properties/")
	if s == "" || strings.HasPrefix(s, "G-") || strings.HasPrefix(s, "UA-") {
		return "", false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return s, true
}

// FinalizedThrough is the last Search Console day that can be stored as final.
// firstIncomplete is metadata.firstIncompleteDate. An empty value falls back two Pacific days.
func FinalizedThrough(now time.Time, firstIncomplete string) (time.Time, string) {
	if d, ok := parseDay(firstIncomplete); ok {
		return d.AddDate(0, 0, -1), "metadata"
	}
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PDT", -7*3600)
	}
	y, m, d := now.In(loc).Date()
	pacific := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return pacific.AddDate(0, 0, -2), "fallback"
}

func parseDay(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	d, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return d, true
}

func ClassifyGoogle(status int, msg string) (class, label string) {
	low := strings.ToLower(msg)
	if status == 401 || strings.Contains(low, "invalid_grant") || strings.Contains(low, "unauthenticated") {
		return "needs_reauth", "Reconnect Google"
	}
	if status == 429 || strings.Contains(low, "quota") || strings.Contains(low, "rate limit") {
		return "rate_limited", "Waiting for Google quota; will retry later"
	}
	if strings.Contains(low, "duplicate metric") {
		return "provider", "Duplicate request; sync again"
	}
	if status == 403 || strings.Contains(low, "invalid site") || strings.Contains(low, "invalid property") || strings.Contains(low, "property id") {
		return "config_invalid", "Choose the property again"
	}
	if status == 400 && (strings.Contains(low, "invalid") || strings.Contains(low, "not found")) {
		return "config_invalid", "Choose the property again"
	}
	return "provider", "Request failed; sync again"
}

// GSCDailiesFromFacts keeps only the web date slice. Query and other search types are drill-down.
func GSCDailiesFromFacts(property string, facts []model.GscFact) []model.GscDaily {
	out := make([]model.GscDaily, 0)
	for _, fact := range facts {
		if fact.Slice != "date" || fact.SearchType != "web" {
			continue
		}
		out = append(out, model.GscDaily{
			ProjectID:   fact.ProjectID,
			Property:    property,
			SearchType:  "web",
			Day:         fact.Day,
			Clicks:      fact.Clicks,
			Impressions: fact.Impressions,
			CTR:         fact.CTR,
			Position:    fact.Position,
			FetchedAt:   fact.FetchedAt,
		})
	}
	return out
}

type WindowTotals struct {
	Clicks              float64
	Impressions         float64
	Sessions            float64
	PreviousClicks      float64
	PreviousImpressions float64
	PreviousSessions    float64
	CoveredDays         int
}

func WindowFromGSC(rows []model.GscDaily, through time.Time, days int) WindowTotals {
	return windowSpan(days, through, func(d time.Time) (clicks, impressions, sessions float64, ok bool) {
		for _, row := range rows {
			if dateOnly(row.Day).Equal(d) {
				return row.Clicks, row.Impressions, 0, true
			}
		}
		return 0, 0, 0, false
	})
}

func WindowFromGA(rows []model.GaDaily, through time.Time, days int) WindowTotals {
	return windowSpan(days, through, func(d time.Time) (clicks, impressions, sessions float64, ok bool) {
		for _, row := range rows {
			if dateOnly(row.Day).Equal(d) {
				return 0, 0, row.Sessions, true
			}
		}
		return 0, 0, 0, false
	})
}

func windowSpan(days int, through time.Time, at func(time.Time) (float64, float64, float64, bool)) WindowTotals {
	if days < 1 {
		days = 1
	}
	through = dateOnly(through)
	start := through.AddDate(0, 0, -(days - 1))
	prevEnd := start.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -(days - 1))
	var w WindowTotals
	for d := start; !d.After(through); d = d.AddDate(0, 0, 1) {
		clicks, impressions, sessions, ok := at(d)
		if !ok {
			continue
		}
		w.Clicks += clicks
		w.Impressions += impressions
		w.Sessions += sessions
		w.CoveredDays++
	}
	for d := prevStart; !d.After(prevEnd); d = d.AddDate(0, 0, 1) {
		clicks, impressions, sessions, ok := at(d)
		if !ok {
			continue
		}
		w.PreviousClicks += clicks
		w.PreviousImpressions += impressions
		w.PreviousSessions += sessions
	}
	return w
}

type SourceInput struct {
	Connected    bool
	Property     string
	State        string
	PausedReason string
	Through      *time.Time
	Calendar     string
	Backfilling  bool
}

type SourceView struct {
	Source       string `json:"source"`
	State        string `json:"state"`
	Label        string `json:"label"`
	Detail       string `json:"detail"`
	Property     string `json:"property"`
	Through      string `json:"through"`
	Calendar     string `json:"calendar"`
	CanReconnect bool   `json:"can_reconnect"`
	CanSync      bool   `json:"can_sync"`
	Backfill     bool   `json:"backfill"`
}

func PresentSource(in SourceInput) SourceView {
	view := SourceView{Property: in.Property, Calendar: in.Calendar}
	if !in.Connected {
		view.State = "not_connected"
		view.Label = "Not connected"
		view.Detail = "Connect a Google account or save a service account first."
		return view
	}
	if strings.TrimSpace(in.Property) == "" {
		view.State = "choose_property"
		view.Label = "Choose a property"
		view.Detail = "After connecting, choose a property to start the import. Rows already imported stay under their original property."
		return view
	}
	switch in.PausedReason {
	case "needs_reauth":
		view.State = "needs_reauth"
		view.Label = "Reconnect Google"
		view.Detail = "Imported data is kept. After reconnecting, the import resumes from where it stopped."
		view.CanReconnect = true
		return view
	case "config_invalid":
		view.State = "config_invalid"
		view.Label = "Choose the property again"
		view.Detail = "The stored property does not match what Google returns. Choose the property again; no need to reconnect."
		return view
	case "rate_limited":
		view.State = "rate_limited"
		view.Label = "Waiting for Google quota"
		view.Detail = "Waiting for Google quota; will retry later. Imported data is kept."
		view.CanSync = true
		return view
	case "error":
		view.State = "failed"
		view.Label = "Request failed"
		view.Detail = "Sync again. Imported data is kept."
		view.CanSync = true
		return view
	}
	if in.Through != nil && !in.Through.IsZero() {
		view.State = "ready"
		view.Label = "Imported through " + in.Through.UTC().Format("2006-01-02")
		view.Through = in.Through.UTC().Format("2006-01-02")
		view.Detail = "Totals come from date-only requests. Summing query rows gives less; that is expected."
		if in.Backfilling {
			view.Detail += " Older months continue on the next sync."
			view.Backfill = true
		}
		view.CanSync = true
		return view
	}
	view.State = "waiting"
	view.Label = "No final data yet"
	view.Detail = "Sync imports the history Google still keeps: up to 16 months for Search Console."
	view.CanSync = true
	return view
}
