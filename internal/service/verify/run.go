// SPDX-License-Identifier: AGPL-3.0-or-later

package verify

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/audit"
	"github.com/craftsail/craftsail-growth/internal/service/crawl"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
)

type Service struct {
	db       *gorm.DB
	projects *project.Service
	audit    *audit.Service
	sample   *sample.Service
	tasks    *repo.Tasks
	verifies *repo.Verifies
}

func New(db *gorm.DB) *Service {
	return &Service{
		db: db, projects: project.New(db), audit: audit.New(db), sample: sample.New(db, sample.NewAsker()),
		tasks: &repo.Tasks{DB: db}, verifies: &repo.Verifies{DB: db},
	}
}

type Report struct {
	Changed int                  `json:"changed"`
	Pass    int                  `json:"pass"`
	Fail    int                  `json:"fail"`
	Manual  int                  `json:"manual"`
	Results []model.VerifyResult `json:"results"`
}

func (s *Service) Run(ctx context.Context, slug string, recrawl bool) (*Report, error) {
	if recrawl {
		cs := crawl.New(s.db, httputil.New())
		cs.Delay = 200 * time.Millisecond
		if _, err := cs.Run(ctx, slug, 0); err != nil {
			return nil, err
		}
		if _, err := s.audit.Run(ctx, slug); err != nil {
			return nil, err
		}
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	tasks, err := s.tasks.List(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return &Report{}, nil
	}
	in, err := s.inputs(ctx, slug, p.ID)
	if err != nil {
		return nil, err
	}
	changed := 0
	var rows []model.VerifyResult
	pass, fail, manual := 0, 0, 0
	now := time.Now()
	for i := range tasks {
		t := &tasks[i]
		o := CheckWith(*t, in)
		prev := t.Status
		if o.Progress != nil {
			o.Progress["at"] = now.Format(time.RFC3339)
			if t.ProgressFirst == nil {
				cp := map[string]any{}
				for k, v := range o.Progress {
					cp[k] = v
				}
				t.ProgressFirst = cp
			}
			t.Progress = o.Progress
		}
		verdict := "pending"
		switch {
		case o.OK != nil && *o.OK:
			verdict = "pass"
			pass++
		case o.OK != nil:
			verdict = "fail"
			fail++
		default:
			manual++
		}
		if next := NextStatus(t.Status, o.OK); next != t.Status {
			t.Status = next
			changed++
			if next == model.TaskVerified {
				at := now.Unix()
				t.VerifiedAt = &at
				t.ClosedAt = &at
			}
		}
		res := verdict
		t.Evidence = append(t.Evidence, now.Format(time.RFC3339)+" "+res+" "+o.Note)
		if len(t.Evidence) > 6 {
			t.Evidence = t.Evidence[len(t.Evidence)-6:]
		}
		if err := s.tasks.Save(ctx, t); err != nil {
			return nil, err
		}
		rows = append(rows, model.VerifyResult{
			TaskCode: t.Code, Title: t.Title, Priority: t.Priority, Verdict: verdict,
			Note: o.Note, Was: prev, Now: t.Status, Progress: o.Progress,
		})
	}
	rep := &model.VerifyReport{ProjectID: p.ID, RunAt: now.Unix(), Changed: changed,
		Summary: map[string]any{"pass": pass, "fail": fail, "manual": manual}}
	if err := s.verifies.Save(ctx, rep, rows); err != nil {
		return nil, err
	}
	return &Report{Changed: changed, Pass: pass, Fail: fail, Manual: manual, Results: rows}, nil
}

func (s *Service) Latest(ctx context.Context, slug string) (*model.VerifyReport, []model.VerifyResult, error) {
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	return s.verifies.Latest(ctx, p.ID)
}

// inputs gathers audit findings and current metrics once for all tasks.
func (s *Service) inputs(ctx context.Context, slug string, projectID uint64) (Inputs, error) {
	in := Inputs{Visibility: map[string][2]int{}, Prompt: map[string]map[string][2]int{}}
	if rep, _ := s.audit.Latest(ctx, slug); rep != nil {
		in.Audit = rep.Audit
		in.Pages = rep.Pages
	}
	issues, err := (&repo.Audits{DB: s.db}).LatestIssues(ctx, projectID)
	if err != nil {
		return in, err
	}
	in.Issues = issues
	in.HaveIssues = in.Audit.ID != 0
	if m, _ := s.sample.PeriodMetrics(ctx, slug); m != nil {
		in.Metrics = m.Payload
	}
	for _, access := range []string{"api", "web"} {
		view, err := s.sample.Measure(ctx, slug, sample.MeasureQuery{Range: "30d", Access: access})
		if err != nil || view == nil || view.Access != access {
			continue
		}
		in.Visibility[access] = [2]int{view.VisibilityX, view.VisibilityN}
		per := map[string][2]int{}
		for _, pc := range view.PromptCharts {
			per[pc.ID] = [2]int{pc.X, pc.N}
		}
		in.Prompt[access] = per
	}
	return in, nil
}
