// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

var now = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) // Wednesday

func i64(v int64) *int64     { return &v }
func f64(v float64) *float64 { return &v }
func day(d int) int64        { return now.Unix() - int64(d)*86400 }

func page(published, indexedAfter int, impressions bool) model.IndexURL {
	p := model.IndexURL{PublishedAt: i64(day(published))}
	if indexedAfter >= 0 {
		p.FirstIndexedAt = i64(day(published) + int64(indexedAfter)*86400)
	}
	if impressions {
		p.FirstImpressionAt = i64(day(published) + 86400)
	}
	return p
}

func TestNewFast(t *testing.T) {
	cases := []struct {
		name  string
		pages []model.IndexURL
		want  string
	}{
		{"fast with impressions", []model.IndexURL{page(20, 1, true), page(15, 2, true), page(10, 3, false)}, "met"},
		{"slow", []model.IndexURL{page(20, 9, true), page(15, 8, true), page(10, -1, true)}, "unmet"},
		{"mostly no impressions", []model.IndexURL{page(20, 1, false), page(15, 1, false), page(10, 1, true)}, "unmet"},
		{"too few", []model.IndexURL{page(20, 1, true), page(15, 1, true)}, "no_data"},
		{"young pages are not judged", []model.IndexURL{page(20, 1, true), page(15, 1, true), page(1, -1, false)}, "no_data"},
	}
	for _, c := range cases {
		got := newFast(Facts{Now: now, Search: &webstats.PlaybookSearch{NewPages: c.pages}})
		if got.State != c.want {
			t.Errorf("%s: %s, want %s", c.name, got.State, c.want)
		}
	}
	if newFast(Facts{Now: now}).State != "no_data" {
		t.Error("no search facts must be no_data")
	}
}

func TestSearchSignals(t *testing.T) {
	share := func(v ...float64) []*float64 {
		out := []*float64{}
		for _, x := range v {
			if x < 0 {
				out = append(out, nil)
			} else {
				out = append(out, f64(x))
			}
		}
		return out
	}
	cases := []struct {
		name string
		got  Signal
		want string
	}{
		{"coverage met", imprCoverageSignal(&webstats.PlaybookSearch{Indexed: 10, IndexedWithImpressions: 6}), "met"},
		{"coverage unmet", imprCoverageSignal(&webstats.PlaybookSearch{Indexed: 10, IndexedWithImpressions: 4}), "unmet"},
		{"coverage nothing indexed", imprCoverageSignal(&webstats.PlaybookSearch{IndexedWithImpressions: 4}), "no_data"},
		{"coverage none with impressions", imprCoverageSignal(&webstats.PlaybookSearch{Indexed: 10}), "unmet"},
		{"share rising", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.1, 0.15, 0.2, 0.25, 0.3)}), "met"},
		{"share high", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(-1, -1, -1, -1, 0.55)}), "met"},
		{"share flat", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.1, 0.15, 0.15, 0.25, 0.3)}), "unmet"},
		{"share gap", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.1, -1, 0.2, 0.25, 0.3)}), "unmet"},
		{"share latest unknown", visibleShareSignal(&webstats.PlaybookSearch{VisibleShare: share(0.6, -1)}), "no_data"},
		{"ga clean", gaThresholdSignal(&webstats.PlaybookSearch{GAThresholded: new(bool)}), "met"},
		{"ga unknown", gaThresholdSignal(&webstats.PlaybookSearch{}), "no_data"},
		{"recognized", recognizedSignal(27, 30), "met"},
		{"not recognized", recognizedSignal(15, 30), "unmet"},
		{"recognition small sample", recognizedSignal(20, 20), "no_data"},
	}
	for _, c := range cases {
		if c.got.State != c.want {
			t.Errorf("%s: %s, want %s", c.name, c.got.State, c.want)
		}
	}
}

func TestStagesAndChecks(t *testing.T) {
	base := Facts{Now: now, Layers: map[string]string{}, Rules: map[string]bool{}, Confirmed: map[string]bool{}, Search: &webstats.PlaybookSearch{}}

	s := Judge(base)
	if s.SeoStage != "seoNew" || s.AiStage != "aiReach" || s.Checks["first_check"].State != "todo" {
		t.Fatalf("empty project: %+v", s)
	}
	if s.Signals["crawlUp"].State != "manual" {
		t.Fatalf("manual signal: %+v", s.Signals["crawlUp"])
	}

	noSite := base
	noSite.NoSite, noSite.Search = true, nil
	if s := Judge(noSite); s.SeoStage != "" || s.AiStage != "aiKnown" {
		t.Fatalf("no site: seo %q ai %q", s.SeoStage, s.AiStage)
	}

	est := base
	est.Search = &webstats.PlaybookSearch{Established: true}
	if Judge(est).SeoStage != "seoGrow" {
		t.Fatal("established search stage must count as passed")
	}

	six := base
	six.Confirmed = map[string]bool{"sitemapStable": true, "discoveredDown": true, "crawlUp": true, "brandFilter": true}
	six.Search = &webstats.PlaybookSearch{Indexed: 10, IndexedWithImpressions: 8, GAThresholded: new(bool)}
	if s := Judge(six); s.SeoMet != 6 || s.SeoStage != "seoGrow" {
		t.Fatalf("six signals: met %d stage %s", s.SeoMet, s.SeoStage)
	}

	ai := base
	ai.AuditDone = true
	ai.Layers = map[string]string{"access": "ok", "discover": "ok", "understand": "warn"}
	ai.RecognitionX, ai.RecognitionN = 28, 30
	s = Judge(ai)
	if s.AiStage != "aiRecommend" || s.Checks["audit_layers_ok"].State != "todo" || s.Checks["audit_access_ok"].State != "done" {
		t.Fatalf("ai: %s %+v", s.AiStage, s.Checks)
	}
	ai.Layers["discover"] = "blocked"
	if Judge(ai).AiStage != "aiReach" {
		t.Fatal("a blocked layer does not pass")
	}

	week := base
	monday := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	week.ReportOn = &monday
	week.Search = &webstats.PlaybookSearch{LastIndexSuccess: i64(day(8))}
	s = Judge(week)
	if s.Checks["report_this_week"].State != "done" || s.Checks["index_checked_7d"].State != "todo" {
		t.Fatalf("week checks: %+v", s.Checks)
	}
	sunday := monday.AddDate(0, 0, -1)
	week.ReportOn = &sunday
	if Judge(week).Checks["report_this_week"].State != "todo" {
		t.Fatal("last week's report does not count")
	}
	if Judge(noSite).Checks["index_checked_7d"].State != "no_data" {
		t.Fatal("no site has no index check")
	}
}

func TestSitemapOK(t *testing.T) {
	cases := []struct {
		name      string
		auditDone bool
		rules     map[string]bool
		want      bool
	}{
		{"no audit yet", false, map[string]bool{}, false},
		{"clean audit", true, map[string]bool{}, true},
		{"no sitemap", true, map[string]bool{"NO_SITEMAP": true}, false},
		{"not declared", true, map[string]bool{"SITEMAP_NOT_DECLARED": true}, false},
		{"low value urls", true, map[string]bool{"SITEMAP_LOW_VALUE_URLS": true}, false},
		{"unrelated rule only", true, map[string]bool{"NO_LLMS_TXT": true}, true},
	}
	for _, c := range cases {
		got := done(sitemapOK(Facts{AuditDone: c.auditDone, Rules: c.rules}))
		want := "todo"
		if c.want {
			want = "done"
		}
		if got.State != want {
			t.Errorf("%s: %s, want %s", c.name, got.State, want)
		}
	}
}

func TestRecognitionAndReviewChecks(t *testing.T) {
	base := Facts{Now: now, Layers: map[string]string{}, Rules: map[string]bool{}, Confirmed: map[string]bool{}}

	// recognition_n reads the 30-day count now, not a 7-day one.
	low := base
	low.RecognitionN = recognitionN - 1
	if Judge(low).Checks["recognition_n"].State != "todo" {
		t.Fatal("below the 30-day threshold must not be done")
	}
	high := base
	high.RecognitionN = recognitionN
	if Judge(high).Checks["recognition_n"].State != "done" {
		t.Fatal("at least recognitionN answers in 30 days must be done")
	}

	// experiments_reviewed is no_data, not done, when the project has no
	// observations at all; once there are observations, it reads DueObservations.
	none := base
	if Judge(none).Checks["experiments_reviewed"].State != "no_data" {
		t.Fatal("no observations at all must be no_data")
	}
	reviewed := base
	reviewed.Observations = 3
	if Judge(reviewed).Checks["experiments_reviewed"].State != "done" {
		t.Fatal("observations exist and none are due: done")
	}
	reviewed.DueObservations = 1
	if Judge(reviewed).Checks["experiments_reviewed"].State != "todo" {
		t.Fatal("a due observation makes it todo")
	}
}

func TestReportThisWeekLocalTimeZone(t *testing.T) {
	// Reports are stamped with the server's local time.Now(), not UTC, so
	// the "this week" cutoff must be computed in that same local zone:
	// forcing either side of the comparison into UTC can shift which
	// calendar day (and so which week) a time near local midnight falls
	// into, letting a report from the week before incorrectly count.
	loc := time.FixedZone("test+9", 9*3600)
	nowLocal := time.Date(2026, 10, 5, 2, 0, 0, 0, loc)    // Monday 02:00 local, just after the week started
	lastSunday := time.Date(2026, 10, 4, 20, 0, 0, 0, loc) // Sunday evening local: last week
	f := Facts{Now: nowLocal, Layers: map[string]string{}, Rules: map[string]bool{}, Confirmed: map[string]bool{}, ReportOn: &lastSunday}
	if Judge(f).Checks["report_this_week"].State != "todo" {
		t.Fatal("last week's local Sunday report must not count as this week's")
	}
	thisMonday := time.Date(2026, 10, 5, 1, 0, 0, 0, loc) // Monday 01:00 local: this week
	f.ReportOn = &thisMonday
	if Judge(f).Checks["report_this_week"].State != "done" {
		t.Fatal("this week's local Monday report must count")
	}
}

func TestObservationDue(t *testing.T) {
	o := model.Observation{FollowupThrough: "2026-10-05", Metric: "technical"}
	if !observationDue(o, now) {
		t.Fatal("technical window ended two days ago")
	}
	future := model.Observation{FollowupThrough: "2026-10-10", Metric: "technical"}
	if observationDue(future, now) {
		t.Fatal("technical window has not ended yet")
	}
	o.Metric = "clicks"
	if observationDue(o, now) {
		t.Fatal("search metrics wait for data to settle")
	}
	o.Results = []model.ObservationResult{{}}
	if observationDue(o, now.AddDate(0, 0, 10)) {
		t.Fatal("evaluated observation is not due")
	}
}
