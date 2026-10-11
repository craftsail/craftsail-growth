// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/plan"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

var ErrInvalid = errors.New("unknown product stage or signal")

// Service reports where a project stands in the playbooks. It only reads
// data other features store, apart from the product stage and the manual
// signal confirmations.
type Service struct {
	db       *gorm.DB
	projects *project.Service
	audits   *repo.Audits
	sample   *sample.Service
	plan     *plan.Service
	reports  *repo.Reports
	web      *webstats.Service
	rows     *repo.Playbook
	// Now overrides time.Now in tests.
	Now func() time.Time
}

func New(db *gorm.DB) *Service {
	return &Service{
		db: db, projects: project.New(db), audits: &repo.Audits{DB: db}, sample: sample.New(db, nil),
		plan: plan.New(db), reports: &repo.Reports{DB: db}, web: webstats.New(db), rows: &repo.Playbook{DB: db},
	}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) Status(ctx context.Context, slug string) (*Status, error) {
	f, err := s.facts(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := Judge(*f)
	return &out, nil
}

func (s *Service) SetStage(ctx context.Context, slug, stage string) error {
	switch stage {
	case "", "s0", "s1", "s2":
	default:
		return ErrInvalid
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	return s.rows.SetStage(ctx, p.ID, stage)
}

func (s *Service) Confirm(ctx context.Context, slug, signal string, user uint64, on bool) error {
	if !ManualSignals[signal] {
		return ErrInvalid
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return err
	}
	return s.rows.Confirm(ctx, p.ID, signal, user, on)
}

func (s *Service) facts(ctx context.Context, slug string) (*Facts, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	prog, err := s.projects.Progress(ctx, slug)
	if err != nil {
		return nil, err
	}
	f := &Facts{
		Now: s.now(), NoSite: p.NoSite, ProductStage: prog.ProductStage, QuestionsConfirmed: prog.QuestionsConfirmed,
		GoogleSelected: strings.TrimSpace(p.GscSite) != "", BrandFilled: brandFilled(p.Brand),
		Layers: map[string]string{}, Rules: map[string]bool{},
	}
	a, _, err := s.audits.Latest(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	f.FirstCheck = a != nil && (a.NoSite || a.PageCount > 0)
	if a != nil && !a.NoSite && a.PageCount > 0 {
		f.AuditDone = true
		for _, l := range a.Layers {
			key, _ := l["key"].(string)
			status, _ := l["status"].(string)
			if b, _ := l["blocked_by"].(string); b != "" {
				status = "blocked"
			}
			f.Layers[key] = status
		}
		issues, err := s.audits.LatestIssues(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		for _, i := range issues {
			f.Rules[i.Code] = true
		}
	}
	runs, err := s.sample.Runs(ctx, slug, 50)
	if err != nil {
		return nil, err
	}
	for _, r := range runs {
		if r.Status == "completed" && r.Succeeded > 0 {
			f.EngineAnswered = true
			break
		}
	}
	comps, err := (&repo.Competitors{DB: s.db}).List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for _, c := range comps {
		if model.CompetitorConfirmed(c) {
			f.Competitors++
		}
	}
	month, err := s.sample.Measure(ctx, slug, sample.MeasureQuery{Range: "30d"})
	if err != nil {
		return nil, err
	}
	if month.Recognition != nil {
		f.RecognitionN = month.RecognitionN
		f.RecognitionX = int(math.Round(*month.Recognition / 100 * float64(month.RecognitionN)))
	}
	if r, err := s.reports.Latest(ctx, p.ID); err != nil {
		return nil, err
	} else if r != nil {
		on := r.ReportOn
		f.ReportOn = &on
	}
	tasks, err := (&repo.Tasks{DB: s.db}).List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.Status == model.TaskOpen || t.Status == model.TaskDoing {
			f.OpenTasks++
		}
	}
	obs, err := s.plan.Observations(ctx, slug, "")
	if err != nil {
		return nil, err
	}
	f.Observations = len(obs)
	for _, o := range obs {
		if observationDue(o, f.Now) {
			f.DueObservations++
		}
	}
	if !p.NoSite {
		if f.Search, err = s.web.PlaybookSearch(ctx, p); err != nil {
			return nil, err
		}
	}
	if f.Confirmed, err = s.rows.Confirmations(ctx, p.ID); err != nil {
		return nil, err
	}
	return f, nil
}

func brandFilled(b model.Brand) bool {
	if model.IsPendingFact(b.Definition) || model.IsPendingFact(b.TargetUsers) {
		return false
	}
	for _, a := range b.Aliases {
		if !model.IsPendingFact(a) {
			return true
		}
	}
	return false
}
