// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/chartmath"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

type Service struct {
	projects *project.Service
	rows     *repo.Webstats
	Client   *Client
	// Date-only totals. Nil uses Client. Query rows must not be passed here.
	FetchGSCDate func(ctx context.Context, token, site, start, end string) ([]model.GscFact, string, int, bool, error)
	FetchGADate  func(ctx context.Context, token, property, start, end string) ([]model.GaDaily, int, bool, error)
	// Now overrides time.Now in tests. Nil uses time.Now.
	Now func() time.Time
}

func New(db *gorm.DB) *Service {
	return &Service{
		projects: project.New(db),
		rows:     &repo.Webstats{DB: db},
		Client:   &Client{},
	}
}

type RunResult struct {
	Pending   bool   `json:"pending"`
	Requests  int    `json:"requests"`
	Batches   int    `json:"batches"`
	GscRows   int    `json:"gsc_rows"`
	GaRows    int    `json:"ga_rows"`
	From      string `json:"from"`
	To        string `json:"to"`
	IndexNote string `json:"index_note,omitempty"`
}

// Run pulls official date-only totals and the fact slices for the project's
// Search Console and GA4 properties. Query rows are drill-down only.
func (s *Service) Run(ctx context.Context, slug string) (*RunResult, error) {
	return s.RunBatch(ctx, slug, BatchOptions{})
}

func (s *Service) run(ctx context.Context, slug string) (*RunResult, error) {
	if !UserConnected() && strings.TrimSpace(os.Getenv("GOOGLE_SA_JSON")) == "" {
		return nil, fmt.Errorf("Google is not connected")
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	explicitGSC := strings.TrimSpace(p.GscSite)
	gaProp := strings.TrimSpace(p.GA4Property)
	projectSite := strings.TrimSpace(p.Site)
	if p.NoSite {
		projectSite = ""
	}
	wantGSC := projectSite != "" || explicitGSC != ""
	if !wantGSC && gaProp == "" {
		return nil, fmt.Errorf("set the website or a GA4 property first")
	}

	now := s.now()
	startDay, endDay := window(now)
	live := (wantGSC && s.FetchGSCDate == nil) || (gaProp != "" && s.FetchGADate == nil)
	token, err := s.accessTokenContext(ctx, live)
	if err != nil {
		return nil, err
	}
	gscSite := explicitGSC
	if wantGSC && live {
		granted, listErr := s.client().FetchSitesContext(ctx, token)
		if listErr != nil {
			granted = nil
		}
		gscSite = ResolveGSCSite(granted, projectSite, explicitGSC)
	}

	var notes []string
	var failures []error
	if live && token != "test" {
		s.client().OnPage = func(report, request, body string) {
			source := "gsc"
			if strings.HasPrefix(report, "ga4/") || report == "channel" || report == "landing" || report == "session" || report == "page" || report == "event" || report == "hour" {
				source = "ga4"
			}
			s.archivePage(ctx, p.ID, source, report, request, body, now)
		}
	}
	if official := s.SyncOfficial(ctx, p.ID, token, gscSite, gaProp, now); official != "" {
		notes = append(notes, official)
		failures = append(failures, errors.New(official))
	}
	if live && token != "test" {
		note, syncErr := s.SyncFacts(ctx, p.ID, token, gscSite, gaProp, now)
		if note != "" {
			notes = append(notes, note)
		}
		syncErr = withoutBudgetError(syncErr)
		if syncErr != nil {
			notes = append(notes, syncErr.Error())
			failures = append(failures, syncErr)
		}
	}
	pending, pendingErr := s.pendingSync(context.WithoutCancel(ctx), p.ID, gscSite, gaProp)
	if pendingErr != nil {
		failures = append(failures, pendingErr)
	}
	if live && token != "test" && gscSite != "" && ctx.Err() == nil && len(failures) == 0 && !pending {
		if note, err := s.pullIndex(ctx, token, p.ID, gscSite, now); err != nil {
			failures = append(failures, err)
		} else if note != "" {
			notes = append(notes, note)
		}
	}

	if days, err := strconv.Atoi(strings.TrimSpace(os.Getenv("GOOGLE_RAW_RETENTION_DAYS"))); err == nil && days > 0 {
		_, _ = s.rows.PruneRaw(ctx, days)
	}

	res := &RunResult{Pending: pending, From: startDay.Format("2006-01-02"), To: endDay.Format("2006-01-02"), IndexNote: strings.Join(notes, "; ")}
	if prop := s.officialProperty(ctx, p, "gsc"); prop != "" {
		if facts, err := s.rows.ListQueryPage(context.WithoutCancel(ctx), p.ID, prop, startDay, endDay); err == nil {
			res.GscRows = len(facts)
		}
	}
	if prop := s.officialProperty(ctx, p, "ga4"); prop != "" {
		if facts, err := s.rows.ListGaSessionFacts(context.WithoutCancel(ctx), p.ID, prop, startDay, endDay); err == nil {
			res.GaRows = len(facts)
		}
	}
	return res, errors.Join(failures...)
}

// Snapshot is stored rows plus the monitoring report. Windows and Official come from date-only totals.
type Snapshot struct {
	Observation *SearchObservation `json:"observation"`
	Sync        []SyncProgress     `json:"sync"`
	Insight     map[string]any     `json:"insight"`
	Sitemaps    []model.GscSitemap `json:"sitemaps"`
	Index       []model.GscIndex   `json:"index"`
	Sources     []SourceView       `json:"sources"`
	Windows     []model.WebWindow  `json:"windows"`
	Official    []OfficialRow      `json:"official"`
	Deltas      []MetricDelta      `json:"deltas"`
}

type MetricDelta struct {
	Source string  `json:"source"`
	Metric string  `json:"metric"`
	Kind   string  `json:"kind"`
	Dir    string  `json:"dir"`
	Value  float64 `json:"value"`
}

type OfficialRow struct {
	Day         string   `json:"day"`
	Source      string   `json:"source"`
	Clicks      float64  `json:"clicks"`
	Impressions float64  `json:"impressions"`
	Position    float64  `json:"position,omitempty"`
	Sessions    float64  `json:"sessions"`
	Engaged     *float64 `json:"engaged,omitempty"`
}

func (s *Service) Snapshot(ctx context.Context, slug string) (*Snapshot, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	sitemaps, err := s.rows.ListSitemaps(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	indexed, err := s.rows.ListIndex(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	insight, err := s.Insight(ctx, p)
	if err != nil {
		return nil, err
	}
	observation, searchDaily, err := s.searchObservation(ctx, p)
	if err != nil {
		return nil, err
	}
	sources, windows, official := s.monitorReport(ctx, p)
	var progress []SyncProgress
	for _, source := range []string{"gsc", "ga4"} {
		property := s.officialProperty(ctx, p, source)
		if property == "" {
			continue
		}
		rows, err := s.syncProgress(ctx, p.ID, source, property)
		if err != nil {
			return nil, err
		}
		progress = append(progress, rows...)
	}
	return &Snapshot{
		Observation: observation, Sync: progress, Insight: insight, Sitemaps: sitemaps, Index: indexed,
		Sources: sources, Windows: windows, Official: official, Deltas: snapshotDeltas(windows, observation, searchDaily),
	}, nil
}

// Snapshot KPIs follow the same coverage gate as the search diagnostics.
func snapshotDeltas(windows []model.WebWindow, observation *SearchObservation, rows []model.GscDaily) []MetricDelta {
	var eligible []model.WebWindow
	for _, win := range windows {
		if win.Source == "gsc" {
			if observation == nil || observation.Coverage.State != "covered" || observation.PreviousCoverage.State != "covered" || win.Property != observation.Property || win.WindowDays != 28 || win.FinalizedThrough.Format("2006-01-02") != observation.Coverage.Through {
				continue
			}
			from, _ := time.Parse("2006-01-02", observation.PreviousCoverage.From)
			to, _ := time.Parse("2006-01-02", observation.Coverage.Through)
			if sumOfficial(rows, from, to).rows != 56 {
				continue
			}
		}
		eligible = append(eligible, win)
	}
	return windowDeltas(eligible)
}

func windowDeltas(windows []model.WebWindow) []MetricDelta {
	out := make([]MetricDelta, 0, len(windows)*3)
	for _, win := range windows {
		if win.Source == "gsc" {
			out = append(out, metricDelta("gsc", "clicks", chartmath.CountDelta(win.Clicks, win.PreviousClicks)))
			out = append(out, metricDelta("gsc", "impressions", chartmath.CountDelta(win.Impressions, win.PreviousImpressions)))
			continue
		}
		out = append(out, metricDelta("ga4", "sessions", chartmath.CountDelta(win.Sessions, win.PreviousSessions)))
	}
	return out
}

func metricDelta(source, metric string, d chartmath.Delta) MetricDelta {
	return MetricDelta{Source: source, Metric: metric, Kind: d.Kind, Dir: d.Dir, Value: d.Value}
}

func (s *Service) monitorReport(ctx context.Context, p *model.Project) ([]SourceView, []model.WebWindow, []OfficialRow) {
	connected := UserConnected() || strings.TrimSpace(os.Getenv("GOOGLE_SA_JSON")) != ""
	sources := make([]SourceView, 0, 2)
	windows := make([]model.WebWindow, 0, 2)
	official := make([]OfficialRow, 0)
	for _, source := range []string{"gsc", "ga4"} {
		active, _ := s.rows.ActiveProperty(ctx, p.ID, source)
		property := active
		if property == "" {
			property = configuredProperty(p, source)
		}
		imp, _ := s.rows.GetImport(ctx, p.ID, source)
		in := SourceInput{Connected: connected, Property: property}
		if source == "gsc" && property != "" {
			in.Calendar = "Pacific time"
		}
		if source == "ga4" && property != "" {
			in.Calendar = "property time zone not recorded"
			if tz := s.propertyTimezone(ctx, p.ID, source, property); tz != "" {
				in.Calendar = tz
			}
		}
		if imp != nil {
			in.State = imp.State
			in.PausedReason = imp.PausedReason
			in.Through = imp.FinalizedThrough
			if imp.Property != "" {
				in.Property = imp.Property
				property = imp.Property
			}
		}
		in.Backfilling = s.officialBackfilling(ctx, p.ID, source, in.Through)
		view := PresentSource(in)
		view.Source = source
		sources = append(sources, view)
		if property == "" {
			continue
		}
		win, _ := s.rows.LatestWindow(ctx, p.ID, source, property)
		if win != nil {
			windows = append(windows, *win)
		}
		from, to := time.Time{}, time.Time{}
		if imp != nil && imp.FinalizedThrough != nil {
			to = dateOnly(*imp.FinalizedThrough)
			from = to.AddDate(0, 0, -(officialWindowDays - 1))
		}
		if source == "gsc" && !to.IsZero() {
			rows, err := s.rows.ListGscDaily(ctx, p.ID, property, from, to)
			if err != nil {
				continue
			}
			for _, row := range rows {
				official = append(official, OfficialRow{
					Day: row.Day.UTC().Format("2006-01-02"), Source: "gsc",
					Clicks: row.Clicks, Impressions: row.Impressions, Position: row.Position,
				})
			}
		}
		if source == "ga4" && !to.IsZero() {
			rows, err := s.rows.ListGaDaily(ctx, p.ID, property, from, to)
			if err != nil {
				continue
			}
			for _, row := range rows {
				official = append(official, OfficialRow{
					Day: row.Day.UTC().Format("2006-01-02"), Source: "ga4",
					Sessions: row.Sessions, Engaged: row.Engaged,
				})
			}
		}
	}
	return sources, windows, official
}

func (s *Service) officialBackfilling(ctx context.Context, projectID uint64, source string, through *time.Time) bool {
	property, err := s.rows.ActiveProperty(ctx, projectID, source)
	if err != nil || property == "" {
		return false
	}
	rows, err := s.rows.SyncReports(ctx, projectID, source, property)
	if err != nil {
		return false
	}
	for _, row := range rows {
		if row.Report == "daily" && row.Version == currentSyncVersion {
			return row.State == "backfilling"
		}
	}
	return false
}

func configuredProperty(p *model.Project, source string) string {
	if source == "ga4" {
		if key, ok := GAPropertyKey(p.GA4Property); ok {
			return key
		}
		return ""
	}
	if key, ok := GSCPropertyKey(p.GscSite); ok {
		return key
	}
	if !p.NoSite {
		if key, ok := GSCPropertyKey(p.Site); ok {
			return key
		}
	}
	return ""
}

func (s *Service) propertyTimezone(ctx context.Context, projectID uint64, source, key string) string {
	var row model.WebProperty
	err := s.rows.DB.WithContext(ctx).
		Where("project_id = ? AND source = ? AND property_key = ?", projectID, source, key).
		First(&row).Error
	if err != nil {
		return ""
	}
	return row.Timezone
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) client() *Client {
	if s.Client == nil {
		s.Client = &Client{}
	}
	return s.Client
}

// accessToken uses "test" when every configured side is hooked; otherwise it exchanges GOOGLE_SA_JSON.
func (s *Service) accessToken(needGoogle bool) (string, error) {
	return s.accessTokenContext(context.Background(), needGoogle)
}
func (s *Service) accessTokenContext(ctx context.Context, needGoogle bool) (string, error) {
	if !needGoogle {
		return "test", nil
	}
	if UserConnected() {
		return s.client().UserAccessTokenContext(ctx)
	}
	return s.client().AccessTokenContext(ctx, os.Getenv("GOOGLE_SA_JSON"))
}

func (s *Service) pullIndex(ctx context.Context, token string, projectID uint64, site string, now time.Time) (string, error) {
	maps, err := s.client().FetchSitemapsContext(ctx, token, site)
	if err != nil {
		return "", syncBudgetError(ctx, err)
	}
	for i := range maps {
		maps[i].ProjectID = projectID
		maps[i].FetchedAt = now.Unix()
	}
	if err := s.rows.UpsertSitemaps(ctx, maps); err != nil {
		return "", err
	}
	targets, err := s.rows.IndexTargets(ctx, projectID, indexInspectCap)
	if err != nil {
		return "", err
	}
	existing, err := s.rows.ListIndex(ctx, projectID)
	if err != nil {
		return "", err
	}
	fresh := map[string]int64{}
	for _, row := range existing {
		fresh[row.URL] = row.FetchedAt
	}
	indexed := make([]model.GscIndex, 0, len(targets))
	var skipped int
	var first string
	for _, page := range targets {
		if at, ok := fresh[page]; ok && now.Unix()-at < int64((7*24*time.Hour).Seconds()) {
			continue
		}
		if !urlInProperty(site, page) {
			skipped++
			continue
		}
		row, err := s.client().InspectURLContext(ctx, token, site, page)
		if err != nil {
			if errors.Is(syncBudgetError(ctx, err), ErrSyncBudget) || ctx.Err() != nil {
				return "", syncBudgetError(ctx, err)
			}
			skipped++
			if first == "" {
				first = err.Error()
			}
			continue
		}
		row.ProjectID = projectID
		row.FetchedAt = now.Unix()
		if err := s.rows.UpsertIndex(ctx, []model.GscIndex{row}); err != nil {
			return "", syncBudgetError(ctx, err)
		}
		if budget := budgetFrom(ctx); budget != nil {
			budget.mu.Lock()
			budget.committed++
			budget.mu.Unlock()
		}
		indexed = append(indexed, row)
	}

	if len(indexed) == 0 && first != "" {
		return "index inspection failed: " + first, nil
	}
	if skipped > 0 && len(indexed) > 0 {
		return fmt.Sprintf("skipped %d URLs outside this Search Console property", skipped), nil
	}
	return "", nil
}

// window ends 3 days ago (GSC lag). start is 27 days before end.
func window(now time.Time) (start, end time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end = today.AddDate(0, 0, -3)
	start = end.AddDate(0, 0, -27)
	return start, end
}
