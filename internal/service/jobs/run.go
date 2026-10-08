// SPDX-License-Identifier: AGPL-3.0-or-later

package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

const (
	PeriodAction = "serve"
	PeriodLabel  = "Run a full period"
)

var (
	ErrBusy          = errors.New("a job is already running for this project; wait for it or stop it")
	ErrUnknownAction = errors.New("unknown action")
	ErrNotFound      = errors.New("job not found")
	ErrNotRunning    = errors.New("job is not running")
)

type ActionFunc func(ctx context.Context, slug string, args map[string]any, log func(string)) error

type Spec struct {
	Resumable bool
	Label     string
	Desc      string
	Slow      bool
	Args      []string
	Run       ActionFunc
}

type Service struct {
	db       *gorm.DB
	projects *project.Service
	jobs     *repo.Jobs
	specs    map[string]Spec
	mu       sync.Mutex
	cancels  map[uint64]context.CancelFunc
	hooks    []func(*model.Job)
}

func New(db *gorm.DB) *Service {
	return &Service{
		db: db, projects: project.New(db), jobs: &repo.Jobs{DB: db},
		specs:   map[string]Spec{},
		cancels: map[uint64]context.CancelFunc{},
	}
}

// Register adds an action with only a run function; tests use it to plug
// in fake jobs.
func (s *Service) Register(action string, fn ActionFunc) {
	s.RegisterSpec(action, Spec{Label: action, Run: fn})
}

func (s *Service) RegisterSpec(action string, spec Spec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.specs[action] = spec
}

func (s *Service) Start(ctx context.Context, slug, action string, args map[string]any) (*model.Job, error) {
	s.mu.Lock()
	sp, ok := s.specs[action]
	s.mu.Unlock()
	if !ok || sp.Run == nil {
		return nil, fmt.Errorf("%w：%s", ErrUnknownAction, action)
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	if running, err := s.jobs.Running(ctx, p.ID); err != nil {
		return nil, err
	} else if running != nil {
		return nil, ErrBusy
	}
	now := time.Now().Unix()
	if args == nil {
		args = map[string]any{}
	}
	j := &model.Job{
		ProjectID: &p.ID, Action: action, Status: "running", Args: args, Resumable: sp.Resumable,
		Log: "$ craftsail-growth " + action + " --slug " + slug + "\n", StartedAt: &now,
	}
	if created, err := s.jobs.CreateExclusive(ctx, j); err != nil {
		return nil, err
	} else if !created {
		return nil, ErrBusy
	}
	jctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[j.ID] = cancel
	s.mu.Unlock()
	go s.exec(jctx, j.ID, slug, action, args, sp)
	return j, nil
}

func (s *Service) exec(ctx context.Context, id uint64, slug, action string, args map[string]any, sp Spec) {
	defer func() {
		s.mu.Lock()
		if c, ok := s.cancels[id]; ok {
			c()
			delete(s.cancels, id)
		}
		s.mu.Unlock()
	}()
	log := func(line string) {
		s.append(id, line)
	}
	log("=== " + or(sp.Label, action) + " started ===")
	// Resume metadata is server-owned, never trusted from a caller.
	current, getErr := s.jobs.ByID(ctx, id)
	if getErr != nil || current == nil || current.Status != "running" || ctx.Err() != nil {
		return
	}
	runArgs := make(map[string]any, len(args)+1)
	for k, v := range args {
		runArgs[k] = v
	}
	if current.StartedAt != nil {
		runArgs["_started_at"] = *current.StartedAt
	}

	err := sp.Run(ctx, slug, runArgs, log)
	j, _ := s.jobs.ByID(context.Background(), id)
	if j == nil || j.Status != "running" {
		return
	}
	now := time.Now().Unix()
	j.FinishedAt = &now
	if ctx.Err() != nil {
		j.Status = "stopped"
		j.Error = ctx.Err().Error()
		log("stopped")
	} else if deferred := new(Deferred); sp.Resumable && errors.As(err, &deferred) {
		next := time.Now().Add(deferred.After).Unix()
		j.Status = "queued"
		j.ResumeAt = &next
		j.FinishedAt = nil
		j.Error = ""
		j.Args = args
		log("batch saved; automatic continuation scheduled")
	} else if err != nil {
		j.Status = "failed"
		j.Error = err.Error()
		log("failed: " + err.Error())
	} else {
		j.Status = "done"
		log("done")
	}
	if saved, err := s.jobs.FinishRunning(context.Background(), j); err == nil && saved && j.Status != "queued" {
		s.finish(j)
	}
}

func (s *Service) OnFinish(fn func(*model.Job)) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.hooks = append(s.hooks, fn)
	s.mu.Unlock()
}

func (s *Service) finish(j *model.Job) {
	s.mu.Lock()
	hooks := append([]func(*model.Job){}, s.hooks...)
	s.mu.Unlock()
	for _, h := range hooks {
		h(j)
	}
}

func (s *Service) Label(action string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sp, ok := s.specs[action]; ok && sp.Label != "" {
		return sp.Label
	}
	return action
}

func (s *Service) Wait(ctx context.Context, id uint64) error {
	for {
		j, err := s.Get(ctx, id)
		if err != nil {
			return err
		}
		if j.Status == "queued" {
			s.resumeDue(ctx, id)
		}
		switch j.Status {
		case "done":
			return nil
		case "failed", "stopped", "interrupted":
			if j.Error != "" {
				return errors.New(j.Error)
			}
			return fmt.Errorf("job %s", j.Status)
		}
		select {
		case <-ctx.Done():
			_ = s.Stop(context.Background(), id)
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (s *Service) append(id uint64, line string) {
	if line != "" && line[len(line)-1] != '\n' {
		line += "\n"
	}
	_ = s.jobs.AppendLog(context.Background(), id, line)
}

func (s *Service) Get(ctx context.Context, id uint64) (*model.Job, error) {
	j, err := s.jobs.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if j == nil {
		return nil, ErrNotFound
	}
	return j, nil
}

func (s *Service) Tail(ctx context.Context, id uint64, offset int) (string, int, *model.Job, error) {
	j, err := s.Get(ctx, id)
	if err != nil {
		return "", offset, nil, err
	}
	log := j.Log
	if offset < 0 {
		offset = 0
	}
	if offset > len(log) {
		offset = len(log)
	}
	chunk := log[offset:]
	return chunk, len(log), j, nil
}

func (s *Service) Stop(ctx context.Context, id uint64) error {
	s.mu.Lock()
	j, err := s.Get(ctx, id)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	if j.Status != "running" && j.Status != "queued" {
		s.mu.Unlock()
		return ErrNotRunning
	}
	changed, err := s.jobs.StopActive(ctx, id)
	if err == nil && changed {
		if cancel := s.cancels[id]; cancel != nil {
			cancel()
		}
	}
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if !changed {
		return ErrNotRunning
	}
	stopped, err := s.Get(ctx, id)
	if err == nil {
		s.finish(stopped)
	}
	return err
}

func (s *Service) Recent(ctx context.Context, slug string, limit int) ([]model.Job, *model.Job, error) {
	var pid uint64
	if slug != "" {
		p, err := s.projects.Get(ctx, slug)
		if err != nil {
			return nil, nil, err
		}
		pid = p.ID
	}
	rows, err := s.jobs.Recent(ctx, pid, limit)
	if err != nil {
		return nil, nil, err
	}
	var running *model.Job
	if pid > 0 {
		running, err = s.jobs.Running(ctx, pid)
		if err != nil {
			return nil, nil, err
		}
	}
	return rows, running, nil
}

func (s *Service) ReapOrphans(ctx context.Context) error {
	return s.jobs.InterruptRunning(ctx)
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
