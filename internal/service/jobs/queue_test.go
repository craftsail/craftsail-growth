// SPDX-License-Identifier: AGPL-3.0-or-later

package jobs

import (
	"context"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeferredJobResumesSameIDAfterRestart(t *testing.T) {
	db := jobDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Resume", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db)
	var calls, finished atomic.Int32
	var started atomic.Int64
	spec := Spec{Resumable: true, Run: func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		stamp, ok := args["_started_at"].(int64)
		if !ok || stamp == 1 {
			t.Error("caller controlled start timestamp")
		}
		log("batch progress")
		if calls.Add(1) == 1 {
			started.Store(stamp)
			return &Deferred{After: time.Hour}
		}
		if stamp != started.Load() {
			t.Error("refresh boundary changed across continuation")
		}
		return nil
	}}
	s.RegisterSpec("webstats", spec)
	s.OnFinish(func(*model.Job) { finished.Add(1) })
	j, err := s.Start(ctx, p.Slug, "webstats", map[string]any{"_started_at": 1})
	if err != nil {
		t.Fatal(err)
	}
	waitStatus(t, s, j.ID, "queued")
	queued, _ := s.Get(ctx, j.ID)
	if queued.ResumeAt == nil || queued.FinishedAt != nil || finished.Load() != 0 {
		t.Fatalf("premature finish %#v", queued)
	}
	if _, err := s.Start(ctx, p.Slug, "webstats", nil); !errors.Is(err, ErrBusy) {
		t.Fatal("queued job should reserve project", err)
	}
	s.ResumeDue(ctx)
	if calls.Load() != 1 {
		t.Fatal("resumed before due time")
	}
	restarted := New(db)
	restarted.RegisterSpec("webstats", spec)
	restarted.OnFinish(func(*model.Job) { finished.Add(1) })
	other := New(db)
	other.RegisterSpec("webstats", spec)
	other.OnFinish(func(*model.Job) { finished.Add(1) })
	if err := restarted.ReapOrphans(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Job{}).Where("id = ?", j.ID).Update("resume_at", time.Now().Add(-time.Second).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, worker := range []*Service{restarted, other} {
		wg.Add(1)
		go func(s *Service) { defer wg.Done(); s.ResumeDue(ctx) }(worker)
	}
	wg.Wait()
	waitDone(t, restarted, j.ID)
	if calls.Load() != 2 {
		t.Fatalf("duplicate claim %d", calls.Load())
	}
	// Finish hook runs just after the durable terminal transition.
	deadline := time.Now().Add(time.Second)
	for finished.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if finished.Load() != 1 {
		t.Fatalf("finish hooks %d", finished.Load())
	}
	done, _ := restarted.Get(ctx, j.ID)
	if done.ResumeAt != nil {
		t.Fatal("resume timestamp retained")
	}
	var count int64
	db.Model(&model.Job{}).Count(&count)
	if count != 1 {
		t.Fatal("continuation created another job")
	}
}
func TestStopQueuedAndStopDuringDeferral(t *testing.T) {
	for _, queued := range []bool{true, false} {
		t.Run(map[bool]string{true: "queued", false: "running"}[queued], func(t *testing.T) {
			db := jobDB(t)
			ctx := context.Background()
			p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Stop", NoSite: true})
			if err != nil {
				t.Fatal(err)
			}
			s := New(db)
			started := make(chan struct{})
			release := make(chan struct{})
			var calls atomic.Int32
			s.RegisterSpec("webstats", Spec{Resumable: true, Run: func(context.Context, string, map[string]any, func(string)) error {
				calls.Add(1)
				close(started)
				if !queued {
					<-release
				}
				return &Deferred{After: time.Hour}
			}})
			j, err := s.Start(ctx, p.Slug, "webstats", nil)
			if err != nil {
				t.Fatal(err)
			}
			<-started
			if queued {
				waitStatus(t, s, j.ID, "queued")
			}
			if err := s.Stop(ctx, j.ID); err != nil {
				t.Fatal(err)
			}
			if !queued {
				close(release)
			}
			s.ResumeDue(ctx)
			waitStatus(t, s, j.ID, "stopped")
			restored := New(db)
			restored.RegisterSpec("webstats", s.specs["webstats"])
			if err := restored.ReapOrphans(ctx); err != nil {
				t.Fatal(err)
			}
			restored.ResumeDue(ctx)
			time.Sleep(25 * time.Millisecond)
			stopped, _ := s.Get(ctx, j.ID)
			if stopped.ResumeAt != nil || calls.Load() != 1 || stopped.Status != "stopped" {
				t.Fatalf("stopped job restarted %#v calls %d", stopped, calls.Load())
			}
		})
	}
}
func TestOnlyResumableOrphansAreQueued(t *testing.T) {
	s := New(jobDB(t))
	ctx := context.Background()
	now := time.Now().Unix()
	rows := []model.Job{{Action: "webstats", Status: "running", Resumable: true, StartedAt: &now}, {Action: "crawl", Status: "running", StartedAt: &now}, {Action: "webstats", Status: "stopped", Resumable: true, StartedAt: &now}}
	for i := range rows {
		if err := s.db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := s.ReapOrphans(ctx); err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"queued", "interrupted", "stopped"} {
		j, _ := s.Get(ctx, rows[i].ID)
		if j.Status != want {
			t.Fatalf("orphan %#v", j)
		}
	}
}
func TestConcurrentStartsAcrossServices(t *testing.T) {
	db := jobDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Exclusive", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	gate := make(chan struct{})
	defer close(gate)
	var successes, busy atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		s := New(db)
		s.Register("slow", func(context.Context, string, map[string]any, func(string)) error { <-gate; return nil })
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Start(ctx, p.Slug, "slow", nil)
			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, ErrBusy) {
				busy.Add(1)
			} else {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || busy.Load() != 5 {
		t.Fatalf("starts %d busy %d", successes.Load(), busy.Load())
	}
}
