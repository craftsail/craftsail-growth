// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

type Service struct {
	projects        *project.Service
	questions       *repo.Questions
	competitors     *repo.Competitors
	samples         *repo.Samples
	metrics         *repo.Metrics
	Ask             func(platform, question string) AskResult
	Sleep           func(time.Duration)
	BypassAvailable bool
}

func New(db *gorm.DB, asker *Asker) *Service {
	if asker == nil {
		asker = NewAsker()
	}
	return &Service{
		projects:    project.New(db),
		questions:   &repo.Questions{DB: db},
		competitors: &repo.Competitors{DB: db},
		samples:     &repo.Samples{DB: db},
		metrics:     &repo.Metrics{DB: db},
		Ask:         asker.Ask,
		Sleep:       func(d time.Duration) { time.Sleep(d) },
	}
}

type RunInput struct {
	Platforms []string
	Repeat    int
	Limit     int
	// FillTo is the number of OK API rounds a buyer question should already have today.
	// Missing rounds are sampled; a full day is skipped.
	FillTo int
	// Trigger records why the run started: manual | schedule. Retries set "retry".
	Trigger string
	JobID   *uint64
	// RetryRun re-asks only the failed engine x prompt x round calls of that run.
	RetryRun uint64
}

type RunResult struct {
	Slug      string               `json:"slug"`
	Count     int                  `json:"count"`
	Skipped   []string             `json:"skipped"`
	Ran       []string             `json:"ran"`
	Errors    map[string]string    `json:"errors"`
	Platforms map[string]PlatStats `json:"platforms"`
	RunID     uint64               `json:"run_id"`
}

func (s *Service) cfgOf(ctx context.Context, p *model.Project) (Cfg, []Question, error) {
	qs, err := s.questions.List(ctx, p.ID)
	if err != nil {
		return Cfg{}, nil, err
	}
	comps, err := s.competitors.List(ctx, p.ID)
	if err != nil {
		return Cfg{}, nil, err
	}
	var cq []Comp
	for _, c := range comps {
		cq = append(cq, Comp{Name: c.Name, Site: c.Site, Aliases: c.Aliases})
	}
	cfg := Cfg{BrandName: p.Name, Aliases: p.Brand.Aliases, Site: p.Site, Competitors: cq}
	var qq []Question
	for _, q := range qs {
		qq = append(qq, Question{Language: q.Language, Record: q, ID: q.QID, Group: q.GroupName, Market: q.Market, Text: q.Text, Tags: q.Tags, Enabled: q.Enabled, Off: !q.Enabled})
	}
	return cfg, qq, nil
}

func (s *Service) Run(ctx context.Context, slug string, in RunInput) (*RunResult, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	cfg, qs, err := s.cfgOf(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(qs) == 0 {
		return nil, fmt.Errorf("the prompt library is empty; run bootstrap first")
	}
	libraryRows := make([]model.Question, len(qs))
	for i, q := range qs {
		libraryRows[i] = q.Record
	}
	library, err := s.questions.Snapshot(ctx, p, libraryRows)
	if err != nil {
		return nil, err
	}
	for i := range qs {
		qs[i].Revision = library.Revision
	}
	repeat := in.Repeat
	if repeat <= 0 {
		repeat = 1
	}
	plats := in.Platforms
	if len(plats) == 0 {
		plats = model.DefaultPlatforms()
	}
	var runnable, skipped []string
	for _, code := range plats {
		if _, ok := Lookup(code); !ok {
			continue
		}
		pr, _ := Lookup(code)
		if pr.Manual {
			skipped = append(skipped, code)
			continue
		}
		if s.Ask == nil {
			skipped = append(skipped, code)
			continue
		}
		if s.BypassAvailable || Available(code) {
			runnable = append(runnable, code)
		} else {
			skipped = append(skipped, code)
		}
	}
	if len(runnable) == 0 {
		return &RunResult{Slug: slug, Skipped: skipped, Errors: map[string]string{}, Platforms: map[string]PlatStats{}}, nil
	}

	day := dateOnly(time.Now())
	calls, err := s.planCalls(ctx, p, qs, runnable, in, repeat, day)
	if err != nil {
		return nil, err
	}
	trigger := in.Trigger
	if trigger == "" {
		trigger = "manual"
	}
	var parent *uint64
	if in.RetryRun > 0 {
		trigger = "retry"
		id := in.RetryRun
		parent = &id
	}
	now := time.Now().Unix()
	run := &model.SampleRun{
		PromptRevision: library.Revision, SamplingLanguage: p.SamplingLanguage, TargetRegion: p.TargetRegion,
		ProjectID: p.ID, JobID: in.JobID, ParentID: parent, Trigger: trigger, Status: "running",
		Planned: len(calls), EstTokens: estimateTokens(len(calls), 1, 1), StartedAt: now, CreatedAt: now,
	}
	if err := s.samples.DB.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}

	// One worker per engine: engines answer in parallel, while calls to the
	// same engine stay sequential so its rate limit is respected. Each answer
	// is saved and counted as it arrives, so progress shows while the run is
	// going and finished answers survive a restart.
	var stored []model.Sample
	var rows []Row
	errs := map[string]string{}
	var mu sync.Mutex
	byPlat := map[string][]call{}
	var order []string
	for _, c := range calls {
		if _, ok := byPlat[c.plat]; !ok {
			order = append(order, c.plat)
		}
		byPlat[c.plat] = append(byPlat[c.plat], c)
	}
	var wg sync.WaitGroup
	for _, plat := range order {
		wg.Add(1)
		go func(plat string, list []call) {
			defer wg.Done()
			pr, _ := Lookup(plat)
			for _, c := range list {
				if ctx.Err() != nil {
					return
				}
				q, rnd := c.q, c.round
				res := s.Ask(plat, q.Text)
				an := Analysis{}
				if res.OK {
					an = AnalyzeAnswer(res.Answer, cfg, res.Citations)
				}
				probe := BrandInQuestion(q.Text, cfg)
				rank := an.BrandRank
				rp := &rank
				if rank == 0 {
					rp = nil
				}
				searched := res.Searched || len(res.Citations) > 0 || len(res.WebQueries) > 0
				runID := run.ID
				sm := model.Sample{
					ProjectID: p.ID, SampledOn: day, Platform: plat, PlatformName: pr.Name,
					PromptRevision: library.Revision, SamplingLanguage: q.Language, TargetRegion: p.TargetRegion,
					QID: q.ID, Round: rnd, RunID: &runID, SampleMode: "api", QuestionText: q.Text,
					Answer: res.Answer, Cited: toCited(res.Citations), Mentioned: an.BrandMentioned,
					Rank: rp, CompetitorsMentioned: an.CompetitorsMentioned,
					Negative: len(an.NegativeCues) > 0, OK: res.OK, BrandInQuestion: probe,
					NeedsReview: an.NeedsReview, NegativeCues: an.NegativeCues,
					CitedDomains: an.CitedDomains, OwnDomainCited: an.OwnDomainCited,
					WebQueries: ReportedWebQueries(searched, append(res.WebQueries, QueriesFromPayload(res.Raw)...)),
					Candidates: an.Candidates, Error: res.Error, Market: MarketOf(plat),
					Env: "api", Raw: map[string]any{"model": res.Model, "searched": searched, "question_tags": q.Tags, "strategy_version": "api-v1", "prompt_revision": library.Revision, "sampling_language": q.Language, "target_region": p.TargetRegion},
				}
				row := Row{
					PromptRevision: library.Revision, Platform: plat, QuestionID: q.ID, Round: rnd, SampleMode: "api",
					Question: q.Text, Market: MarketOf(plat), Day: day.Format("2006-01-02"), OK: res.OK, BrandInQuestion: probe, Analysis: an,
				}
				_ = s.samples.Upsert(ctx, []model.Sample{sm})
				mu.Lock()
				if res.OK {
					run.Succeeded++
				} else {
					run.Failed++
					errs[plat] = res.Error
				}
				stored = append(stored, sm)
				rows = append(rows, row)
				_ = s.samples.DB.Model(&model.SampleRun{}).Where("id = ?", run.ID).
					Updates(map[string]any{"succeeded": run.Succeeded, "failed": run.Failed}).Error
				mu.Unlock()
				if s.Sleep != nil {
					s.Sleep(400 * time.Millisecond)
				}
			}
		}(plat, byPlat[plat])
	}
	wg.Wait()
	cancelled := ctx.Err() != nil
	finished := time.Now().Unix()
	run.FinishedAt = &finished
	run.Status = "completed"
	if cancelled {
		run.Status = "cancelled"
	}
	run.Outcome = runOutcome(run.Succeeded, run.Failed)
	defer func() { _ = s.samples.DB.Save(run).Error }()
	if err := s.samples.Upsert(ctx, stored); err != nil {
		return nil, err
	}
	if err := s.persistCitations(ctx, p.ID, day, cfg); err != nil {
		return nil, err
	}
	okRows := []Row{}
	for _, r := range rows {
		if r.OK {
			okRows = append(okRows, r)
		}
	}
	agg := s.dayStats(ctx, p.ID, day, cfg)
	if len(agg) == 0 {
		agg = Aggregate(DedupRows(okRows), cfg)
	}
	s.writeDay(ctx, p, day, agg, len(rows), len(qs))
	_ = s.confirmCompetitors(ctx, p.ID, okRows)
	return &RunResult{Slug: slug, Count: len(rows), Skipped: skipped, Ran: runnable, Errors: errs, Platforms: agg, RunID: run.ID}, nil
}

// planCalls lists the engine x prompt x round calls for a run. A retry lists
// only the failed calls of the given run that can still be asked.
func (s *Service) planCalls(ctx context.Context, p *model.Project, qs []Question, runnable []string, in RunInput, repeat int, day time.Time) ([]call, error) {
	var calls []call
	if in.RetryRun > 0 {
		byID := map[string]Question{}
		for _, q := range qs {
			byID[q.ID] = q
		}
		can := map[string]bool{}
		for _, plat := range runnable {
			can[plat] = true
		}
		var failed []model.Sample
		if err := s.samples.DB.WithContext(ctx).
			Where("project_id = ? AND run_id = ? AND ok = ?", p.ID, in.RetryRun, false).
			Order("platform, qid, round").Find(&failed).Error; err != nil {
			return nil, err
		}
		for _, sm := range failed {
			if q, ok := byID[sm.QID]; ok && can[sm.Platform] {
				if sm.QuestionText != q.Text || (sm.PromptRevision != "" && sm.PromptRevision != q.Revision) {
					return nil, fmt.Errorf("prompt library changed; start a new run instead of retrying an old version")
				}
				calls = append(calls, call{plat: sm.Platform, q: q, round: sm.Round})
			}
		}
		return calls, nil
	}
	for _, plat := range runnable {
		qset := QuestionsFor(qs)
		var nonempty []Question
		for _, q := range qset {
			if strings.TrimSpace(q.Text) != "" {
				nonempty = append(nonempty, q)
			}
		}
		qset = nonempty
		if in.Limit > 0 && len(qset) > in.Limit {
			qset = qset[:in.Limit]
		}
		for _, q := range qset {
			rounds := make([]int, 0, repeat)
			for rnd := 1; rnd <= repeat; rnd++ {
				rounds = append(rounds, rnd)
			}
			if in.FillTo > 0 && BuyerGroups[q.Group] {
				have, err := s.samples.CountRounds(ctx, p.ID, day, plat, q.ID, "api", q.Revision)
				if err != nil {
					return nil, err
				}
				rounds = NextRounds(have, in.FillTo)
			}
			for _, rnd := range rounds {
				calls = append(calls, call{plat: plat, q: q, round: rnd})
			}
		}
	}
	return calls, nil
}

func (s *Service) confirmCompetitors(ctx context.Context, projectID uint64, rows []Row) error {
	seen := map[string]bool{}
	for _, r := range rows {
		for _, n := range r.Analysis.CompetitorsMentioned {
			seen[n] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	comps, err := s.competitors.List(ctx, projectID)
	if err != nil {
		return err
	}
	changed := false
	t := true
	for i := range comps {
		if comps[i].Confirmed != nil && !*comps[i].Confirmed && seen[comps[i].Name] {
			comps[i].Confirmed = &t
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.competitors.Replace(ctx, projectID, comps)
}

func (s *Service) List(ctx context.Context, slug, platform, qid string) ([]model.Sample, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	rows, err := s.samples.List(ctx, p.ID, time.Time{}, platform, qid, 300)
	for i := range rows {
		rows[i].PlatformName = LabelOf(rows[i].Platform) // current name, not the stored one
	}
	return rows, err
}

// PeriodMetrics computes the same window the dashboard shows by default and
// returns it in the payload shape existing readers expect.
func (s *Service) PeriodMetrics(ctx context.Context, slug string) (*model.Metric, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	cfg, _, err := s.cfgOf(ctx, p)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	since := now.AddDate(0, 0, -metrics.DefaultWindowDays)
	var stored []model.Sample
	if err := s.samples.DB.WithContext(ctx).
		Where("project_id = ? AND sampled_on >= ? AND ok = ?", p.ID, since.Format("2006-01-02"), true).
		Find(&stored).Error; err != nil {
		return nil, err
	}
	if len(stored) == 0 {
		return nil, nil
	}
	rows := make([]Row, 0, len(stored))
	for _, sm := range stored {
		rank := 0
		if sm.Rank != nil {
			rank = *sm.Rank
		}
		// BrandInQuestion is left false on purpose: modeStats recomputes it
		// from the current brand name and aliases.
		rows = append(rows, Row{
			PromptRevision: sm.PromptRevision, Platform: sm.Platform, QuestionID: sm.QID, Round: sm.Round, SampleMode: sm.SampleMode,
			Question: sm.QuestionText, Market: sm.Market, Day: sm.SampledOn.Format("2006-01-02"), OK: true,
			Analysis: Analysis{BrandMentioned: sm.Mentioned, BrandRank: rank,
				CompetitorsMentioned: sm.CompetitorsMentioned, CitedDomains: sm.CitedDomains, OwnDomainCited: sm.OwnDomainCited},
		})
	}
	payload, err := periodPayload(rows, cfg, metrics.DefaultWindowDays)
	if err != nil {
		return nil, err
	}
	return &model.Metric{ProjectID: p.ID, MetricOn: now, Market: p.Market, Payload: payload}, nil
}

// periodPayload aggregates rows and round-trips through JSON so readers that
// type-assert map[string]any keep working.
func periodPayload(rows []Row, cfg Cfg, windowDays int) (map[string]any, error) {
	agg := Aggregate(DedupRows(rows), cfg)
	raw, err := json.Marshal(map[string]any{"platforms": agg, "sample_count": len(rows), "window_days": windowDays})
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) ImportRecords(ctx context.Context, slug string, recs []SheetRec) (*RunResult, error) {
	return s.importRecs(ctx, slug, recs)
}

func (s *Service) ImportMarkdown(ctx context.Context, slug, text string) (*RunResult, error) {
	return s.importRecs(ctx, slug, ParseSheet(text))
}

func (s *Service) importRecs(ctx context.Context, slug string, parsed []SheetRec) (*RunResult, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	cfg, qs, err := s.cfgOf(ctx, p)
	if err != nil {
		return nil, err
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("no answers found; check that the ```answer blocks are filled in")
	}
	records := make([]model.Question, len(qs))
	for i, q := range qs {
		records[i] = q.Record
	}
	library, err := s.questions.Snapshot(ctx, p, records)
	if err != nil {
		return nil, err
	}
	qlangs := map[string]string{}
	qmap := map[string]string{}
	for _, q := range qs {
		qmap[q.ID] = q.Text
		qlangs[q.ID] = q.Language
	}
	day := dateOnly(time.Now())
	var stored []model.Sample
	var rows []Row
	for _, rec := range parsed {
		qtext := rec.Question
		if t := qmap[rec.QID]; t != "" && qtext == "" {
			qtext = t
		}
		language, region, revision := rec.Language, rec.Region, ""
		if language != "" && language != "en" && language != "zh" && language != "pt" {
			return nil, project.ErrInvalidLanguage
		}
		if len(region) > 64 || len(rec.Model) > 128 {
			return nil, fmt.Errorf("sampling metadata too long")
		}
		if qtext == qmap[rec.QID] && qtext != "" {
			if language == "" {
				language = qlangs[rec.QID]
			}
			if region == "" {
				region = p.TargetRegion
			}
			if language == qlangs[rec.QID] && region == p.TargetRegion {
				revision = library.Revision
			}
		}
		cited := rec.Citations
		if len(cited) == 0 {
			cited = citationsFromAnswer(rec.Answer)
		}
		an := AnalyzeAnswer(rec.Answer, cfg, cited)
		rank := an.BrandRank
		rp := &rank
		if rank == 0 {
			rp = nil
		}
		stored = append(stored, model.Sample{
			ProjectID: p.ID, SampledOn: day, Platform: rec.Platform, PlatformName: LabelOf(rec.Platform),
			PromptRevision: revision, SamplingLanguage: language, TargetRegion: region, QID: rec.QID, Round: 1, SampleMode: "manual", QuestionText: qtext, Answer: rec.Answer,
			Mentioned: an.BrandMentioned, Rank: rp, CompetitorsMentioned: an.CompetitorsMentioned,
			OK: true, BrandInQuestion: BrandInQuestion(qtext, cfg), NeedsReview: an.NeedsReview,
			NegativeCues: an.NegativeCues, CitedDomains: an.CitedDomains, OwnDomainCited: an.OwnDomainCited,
			Cited: toCited(cited), WebQueries: ReportedWebQueries(len(rec.WebQueries) > 0, rec.WebQueries),
			Candidates: an.Candidates, Market: MarketOf(rec.Platform), Env: envOf(rec.SessionMode),
			Raw: map[string]any{"session_mode": rec.SessionMode, "model": rec.Model, "strategy_version": "manual-v1", "prompt_revision": revision, "sampling_language": language, "target_region": region},
		})
		rows = append(rows, Row{
			PromptRevision: revision, Platform: rec.Platform, QuestionID: rec.QID, Round: 1, SampleMode: "manual",
			Question: qtext, Market: MarketOf(rec.Platform), Day: day.Format("2006-01-02"), OK: true,
			BrandInQuestion: BrandInQuestion(qtext, cfg), Analysis: an,
		})
	}
	now := time.Now().Unix()
	run := &model.SampleRun{ProjectID: p.ID, Trigger: "import", Status: "completed", Outcome: runOutcome(len(stored), 0),
		Planned: len(stored), Succeeded: len(stored), StartedAt: now, FinishedAt: &now, CreatedAt: now}
	if err := s.samples.DB.WithContext(ctx).Create(run).Error; err != nil {
		return nil, err
	}
	for i := range stored {
		id := run.ID
		stored[i].RunID = &id
	}
	if err := s.samples.Upsert(ctx, stored); err != nil {
		return nil, err
	}
	if err := s.persistCitations(ctx, p.ID, day, cfg); err != nil {
		return nil, err
	}
	agg := s.dayStats(ctx, p.ID, day, cfg)
	if len(agg) == 0 {
		agg = Aggregate(DedupRows(rows), cfg)
	}
	s.writeDay(ctx, p, day, agg, len(rows), len(qs))
	_ = s.confirmCompetitors(ctx, p.ID, rows)
	return &RunResult{Slug: slug, Count: len(rows), Platforms: agg}, nil
}

func (s *Service) dayStats(ctx context.Context, projectID uint64, day time.Time, cfg Cfg) map[string]PlatStats {
	stored, err := s.samples.List(ctx, projectID, day, "", "", 2000)
	if err != nil {
		return nil
	}
	var rows []Row
	for _, sm := range stored {
		if !sm.OK {
			continue
		}
		rank := 0
		if sm.Rank != nil {
			rank = *sm.Rank
		}
		rows = append(rows, Row{
			PromptRevision: sm.PromptRevision, Platform: sm.Platform, QuestionID: sm.QID, Round: sm.Round, SampleMode: sm.SampleMode,
			Question: sm.QuestionText, Market: sm.Market, Day: sm.SampledOn.Format("2006-01-02"), OK: true, BrandInQuestion: sm.BrandInQuestion,
			Analysis: Analysis{
				BrandMentioned: sm.Mentioned, BrandRank: rank,
				CompetitorsMentioned: sm.CompetitorsMentioned, CitedDomains: sm.CitedDomains,
				OwnDomainCited: sm.OwnDomainCited,
			},
		})
	}
	return Aggregate(DedupRows(rows), cfg)
}

func (s *Service) writeDay(ctx context.Context, p *model.Project, day time.Time, agg map[string]PlatStats, samples, questions int) {
	if len(agg) == 0 {
		return
	}
	payload := map[string]any{"platforms": agg, "sample_count": samples, "question_count": questions}
	if snap, err := s.citationSnap(ctx, p.ID, day); err == nil {
		payload["citations"] = snap
	}
	if qs := s.querySnap(ctx, p.ID, day); qs.Unavailable > 0 || len(qs.Queries) > 0 {
		payload["web_queries"] = qs
	}
	_ = s.metrics.Upsert(ctx, &model.Metric{ProjectID: p.ID, MetricOn: day, Market: p.Market, Payload: payload})
}

func (s *Service) Sheet(ctx context.Context, slug, intent string, limit int) (string, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return "", err
	}
	_, qs, err := s.cfgOf(ctx, p)
	if err != nil {
		return "", err
	}
	return RenderSheet(p.Name, model.DefaultPlatforms(), qs, intent, limit, Available), nil
}

func toCited(cs []Citation) []model.Citation {
	var out []model.Citation
	for _, c := range cs {
		out = append(out, model.Citation{URL: c.URL, Title: c.Title})
	}
	return out
}

// Runs lists the latest sampling runs, newest first.
func (s *Service) Runs(ctx context.Context, slug string, limit int) ([]model.SampleRun, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	var out []model.SampleRun
	err = s.samples.DB.WithContext(ctx).Where("project_id = ?", p.ID).Order("id desc").Limit(limit).Find(&out).Error
	return out, err
}

// OverrideInput is a person's correction of how an answer was read.
type OverrideInput struct {
	Mentioned *bool `json:"mentioned"`
	Negative  *bool `json:"negative"`
}

func applyOverride(sm *model.Sample, in OverrideInput) {
	if in.Mentioned != nil {
		sm.Mentioned = *in.Mentioned
	}
	if in.Negative != nil {
		sm.Negative = *in.Negative
	}
	sm.ManualOverride = true
	sm.NeedsReview = false
}

// Override stores a manual correction. Metrics are computed on read, so the
// dashboard reflects it immediately, and later re-runs do not overwrite it.
func (s *Service) Override(ctx context.Context, slug string, id uint64, in OverrideInput) (*model.Sample, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	var sm model.Sample
	if err := s.samples.DB.WithContext(ctx).Where("id = ? AND project_id = ?", id, p.ID).First(&sm).Error; err != nil {
		return nil, err
	}
	applyOverride(&sm, in)
	if err := s.samples.DB.WithContext(ctx).Save(&sm).Error; err != nil {
		return nil, err
	}
	return &sm, nil
}
