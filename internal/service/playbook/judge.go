// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// Thresholds mirror THRESHOLDS in web/src/features/playbook/catalog.ts,
// which the help center prints. TestThresholdsMatchCatalog keeps them equal.
const (
	newPageDays      = 3
	imprCoverage     = 50
	visibleShare     = 50
	weeks            = 4
	recognitionN     = 30
	recognitionLower = 50
	seoPassSignals   = 6
)

// Facts is everything the judges read, gathered once per request.
type Facts struct {
	Now                time.Time
	NoSite             bool
	ProductStage       string
	FirstCheck         bool
	GoogleSelected     bool
	EngineAnswered     bool
	AuditDone          bool
	Layers             map[string]string // access, discover, understand, cite → ok | warn | fail | blocked
	Rules              map[string]bool   // rule codes that fired in the latest audit
	BrandFilled        bool
	QuestionsConfirmed bool
	Competitors        int
	RecognitionX       int // branded answers naming the brand, last 30 days
	RecognitionN       int
	ReportOn           *time.Time
	OpenTasks          int
	Observations       int
	DueObservations    int
	Search             *webstats.PlaybookSearch // nil without a site
	Confirmed          map[string]bool
}

type Check struct {
	State string `json:"state"` // done | todo | no_data
	At    *int64 `json:"at,omitempty"`
}

type Signal struct {
	State     string   `json:"state"` // met | unmet | no_data | manual
	Value     *float64 `json:"value,omitempty"`
	Confirmed bool     `json:"confirmed"`
}

type Status struct {
	ProductStage string            `json:"product_stage"`
	SeoStage     string            `json:"seo_stage"` // seoNew | seoGrow | "" without a site
	AiStage      string            `json:"ai_stage"`  // aiReach | aiKnown | aiRecommend
	SeoMet       int               `json:"seo_met"`
	SeoNeeded    int               `json:"seo_needed"`
	Checks       map[string]Check  `json:"checks"`
	Signals      map[string]Signal `json:"signals"`
}

// ManualSignals have no API behind them; the user confirms them.
var ManualSignals = map[string]bool{"sitemapStable": true, "discoveredDown": true, "crawlUp": true, "brandFilter": true}

var seoSignals = []string{"newFast", "sitemapStable", "discoveredDown", "crawlUp", "imprCoverage", "visibleShare", "brandFilter", "gaThreshold"}

// Judge turns facts into the playbook status. The SEO stage passes on the
// book's signals, or when search data already qualified the property as
// established; the AI stages pass in order.
func Judge(f Facts) Status {
	sig := judgeSignals(f)
	out := Status{ProductStage: f.ProductStage, SeoNeeded: seoPassSignals, Checks: judgeChecks(f), Signals: sig}
	for _, id := range seoSignals {
		if sig[id].State == "met" {
			out.SeoMet++
		}
	}
	if !f.NoSite {
		out.SeoStage = "seoNew"
		if out.SeoMet >= seoPassSignals || (f.Search != nil && f.Search.Established) {
			out.SeoStage = "seoGrow"
		}
	}
	out.AiStage = "aiReach"
	if f.NoSite || (sig["accessPass"].State == "met" && sig["discoverPass"].State == "met") {
		out.AiStage = "aiKnown"
		if sig["recognized"].State == "met" {
			out.AiStage = "aiRecommend"
		}
	}
	return out
}

func done(ok bool) Check {
	if ok {
		return Check{State: "done"}
	}
	return Check{State: "todo"}
}

func judgeChecks(f Facts) map[string]Check {
	layer := func(key string) bool { return f.AuditDone && f.Layers[key] == "ok" }
	experimentsReviewed := Check{State: "no_data"}
	if f.Observations > 0 {
		experimentsReviewed = done(f.DueObservations == 0)
	}
	c := map[string]Check{
		"first_check":          done(f.FirstCheck),
		"google_connected":     done(f.GoogleSelected),
		"engines_answered":     done(f.EngineAnswered),
		"audit_layers_ok":      done(layer("access") && layer("discover") && layer("understand")),
		"sitemap_ok":           done(sitemapOK(f)),
		"audit_access_ok":      done(layer("access")),
		"llms_published":       done(f.AuditDone && !f.Rules["NO_LLMS_TXT"]),
		"brand_facts":          done(f.BrandFilled),
		"recognition_n":        done(f.RecognitionN >= recognitionN),
		"questions_confirmed":  done(f.QuestionsConfirmed),
		"competitors_3":        done(f.Competitors >= 3),
		"experiment_open":      done(f.OpenTasks > 0),
		"experiments_reviewed": experimentsReviewed,
		"report_this_week":     done(f.ReportOn != nil && !reportDate(*f.ReportOn, f.Now).Before(weekStart(f.Now))),
		"index_checked_7d":     {State: "no_data"},
	}
	if f.Search != nil {
		last := f.Search.LastIndexSuccess
		idx := done(last != nil && f.Now.Unix()-*last <= 7*86400)
		idx.At = last
		c["index_checked_7d"] = idx
	}
	return c
}

// sitemapOK: the latest audit has no sitemap finding at all, not just the
// two original codes, so a later rule (e.g. low-value URLs) also blocks it.
func sitemapOK(f Facts) bool {
	if !f.AuditDone {
		return false
	}
	for code := range f.Rules {
		if code == "NO_SITEMAP" || strings.HasPrefix(code, "SITEMAP_") {
			return false
		}
	}
	return true
}

// weekStart is Monday 00:00 of t's week, in t's own location.
func weekStart(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// reportDate reads on's calendar date and re-anchors it in now's location:
// reports are stamped with the server's local time.Now(), but a value round
// tripped through the database can come back tagged with a different zone
// that shares the same wall-clock numbers, which would shift the date if
// compared directly against weekStart(now).
func reportDate(on, now time.Time) time.Time {
	return time.Date(on.Year(), on.Month(), on.Day(), 0, 0, 0, 0, now.Location())
}

var noData = Signal{State: "no_data"}

func met(ok bool, v *float64) Signal {
	if ok {
		return Signal{State: "met", Value: v}
	}
	return Signal{State: "unmet", Value: v}
}

func val(v float64) *float64 { return &v }

func judgeSignals(f Facts) map[string]Signal {
	s := map[string]Signal{
		"newFast":      newFast(f),
		"imprCoverage": imprCoverageSignal(f.Search),
		"visibleShare": visibleShareSignal(f.Search),
		"gaThreshold":  gaThresholdSignal(f.Search),
		"accessPass":   layerSignal(f, "access"),
		"discoverPass": layerSignal(f, "discover"),
		"recognized":   recognizedSignal(f.RecognitionX, f.RecognitionN),
	}
	for id := range ManualSignals {
		s[id] = Signal{State: "manual"}
		if f.Confirmed[id] {
			s[id] = Signal{State: "met", Confirmed: true}
		}
	}
	return s
}

// newFast: median days from publish to first observed indexing over new
// pages old enough to judge (unindexed pages older than the threshold count
// as too slow), and at least half of them already have impressions. Value
// is the median in days.
func newFast(f Facts) Signal {
	if f.Search == nil {
		return noData
	}
	var delays []float64
	seen := 0
	for _, p := range f.Search.NewPages {
		if p.PublishedAt == nil {
			continue
		}
		switch {
		case p.FirstIndexedAt != nil:
			delays = append(delays, math.Max(0, float64(*p.FirstIndexedAt-*p.PublishedAt)/86400))
		case float64(f.Now.Unix()-*p.PublishedAt)/86400 > newPageDays:
			delays = append(delays, math.Inf(1))
		default:
			continue
		}
		if p.FirstImpressionAt != nil {
			seen++
		}
	}
	if len(delays) < 3 {
		return noData
	}
	sort.Float64s(delays)
	median := delays[len(delays)/2]
	var v *float64
	if !math.IsInf(median, 1) {
		v = val(median)
	}
	return met(median <= newPageDays && seen*2 >= len(delays), v)
}

func imprCoverageSignal(s *webstats.PlaybookSearch) Signal {
	if s == nil || s.Indexed == 0 {
		return noData
	}
	pct := 100 * float64(s.IndexedWithImpressions) / float64(s.Indexed)
	return met(pct >= imprCoverage, val(pct))
}

// visibleShareSignal: the latest week's share reaches the threshold, or the
// share rose in each of the last `weeks` weeks. Value is the latest share in %.
func visibleShareSignal(s *webstats.PlaybookSearch) Signal {
	if s == nil || len(s.VisibleShare) == 0 || s.VisibleShare[len(s.VisibleShare)-1] == nil {
		return noData
	}
	pct := 100 * *s.VisibleShare[len(s.VisibleShare)-1]
	if pct >= visibleShare {
		return met(true, val(pct))
	}
	if len(s.VisibleShare) < weeks+1 {
		return noData
	}
	tail := s.VisibleShare[len(s.VisibleShare)-weeks-1:]
	for i := 1; i < len(tail); i++ {
		if tail[i-1] == nil || tail[i] == nil {
			// The latest week already missed the threshold; a gap in the
			// weeks before it does not make that unmet result unknown.
			return met(false, val(pct))
		}
		if *tail[i] <= *tail[i-1] {
			return met(false, val(pct))
		}
	}
	return met(true, val(pct))
}

func gaThresholdSignal(s *webstats.PlaybookSearch) Signal {
	if s == nil || s.GAThresholded == nil {
		return noData
	}
	return met(!*s.GAThresholded, nil)
}

func layerSignal(f Facts, key string) Signal {
	if !f.AuditDone {
		return noData
	}
	return met(f.Layers[key] == "ok", nil)
}

// recognizedSignal uses the lower end of the 95% Wilson interval, so a
// small lucky sample does not pass. Value is that lower end in %.
func recognizedSignal(x, n int) Signal {
	if n < recognitionN {
		return noData
	}
	ci := metrics.Wilson(x, n)
	if ci == nil {
		return noData
	}
	lo := 100 * ci.Lo
	return met(lo >= recognitionLower, val(lo))
}

// observationDue: the window has ended and no result was recorded. Mirrors
// the readiness rule in plan.Evaluate (internal/service/plan/observation.go):
// technical observations check an audit state, not accumulated time-series
// data, so they are due as soon as the window itself has ended, with no
// extra wait; clicks/impressions wait 80h past the window's end for GSC to
// finalize the day, everything else waits 24h.
func observationDue(o model.Observation, now time.Time) bool {
	if len(o.Results) > 0 || o.FollowupThrough == "" {
		return false
	}
	end, err := time.Parse("2006-01-02", o.FollowupThrough)
	if err != nil {
		return false
	}
	if o.Metric == "technical" {
		return !now.Before(end)
	}
	wait := 24 * time.Hour
	if o.Metric == "clicks" || o.Metric == "impressions" {
		wait = 80 * time.Hour
	}
	return !now.Before(end.Add(wait))
}
