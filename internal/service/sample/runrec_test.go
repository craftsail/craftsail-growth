// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestRunOutcome(t *testing.T) {
	cases := []struct {
		ok, fail int
		want     string
	}{{5, 0, "succeeded"}, {3, 2, "partial"}, {0, 4, "failed"}, {0, 0, "failed"}}
	for _, c := range cases {
		if got := runOutcome(c.ok, c.fail); got != c.want {
			t.Fatalf("%d/%d -> %s, want %s", c.ok, c.fail, got, c.want)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	if got := estimateTokens(10, 3, 2); got != 10*3*2*960 {
		t.Fatalf("got %d", got)
	}
}

func TestRunRecordsPartialAndRetriesFailuresOnly(t *testing.T) {
	db := sampleDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{
		Name: "甲工", NoSite: true, Materials: strings.Repeat("这是面向团队的诊断工具材料。", 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.New(db).Run(ctx, p.Slug, true); err != nil {
		t.Fatal(err)
	}
	svc := New(db, NewAsker())
	svc.BypassAvailable = true
	svc.Sleep = func(time.Duration) {}
	var calls atomic.Int32
	svc.Ask = func(platform, question string) AskResult {
		calls.Add(1)
		if platform == "kimi" {
			return AskResult{OK: false, Error: "HTTP 503"}
		}
		return AskResult{OK: true, Answer: "甲工不错"}
	}
	res, err := svc.Run(ctx, p.Slug, RunInput{Platforms: []string{"deepseek", "kimi"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	var run model.SampleRun
	if err := db.First(&run, res.RunID).Error; err != nil {
		t.Fatal(err)
	}
	if run.Planned != 4 || run.Succeeded != 2 || run.Failed != 2 || run.Outcome != "partial" || run.Status != "completed" || run.EstTokens == 0 {
		t.Fatalf("run = %+v", run)
	}
	var tagged int64
	db.Model(&model.Sample{}).Where("run_id = ?", run.ID).Count(&tagged)
	if tagged != 4 {
		t.Fatalf("samples tagged with run = %d", tagged)
	}

	calls.Store(0)
	svc.Ask = func(platform, question string) AskResult {
		calls.Add(1)
		return AskResult{OK: true, Answer: "甲工不错"}
	}
	res2, err := svc.Run(ctx, p.Slug, RunInput{RetryRun: run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("retry must only re-ask the 2 failed pairs, asked %d", calls.Load())
	}
	var retry model.SampleRun
	db.First(&retry, res2.RunID)
	if retry.Trigger != "retry" || retry.ParentID == nil || *retry.ParentID != run.ID || retry.Outcome != "succeeded" {
		t.Fatalf("retry run = %+v", retry)
	}
}

func TestApplyOverrideMarksManual(t *testing.T) {
	yes := true
	sm := model.Sample{Mentioned: false, Negative: false, NeedsReview: true}
	applyOverride(&sm, OverrideInput{Mentioned: &yes, Negative: &yes})
	if !sm.Mentioned || !sm.Negative || !sm.ManualOverride || sm.NeedsReview {
		t.Fatalf("%+v", sm)
	}
}

func TestRunAsksEnginesInParallelAndSavesProgress(t *testing.T) {
	db := sampleDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{
		Name: "甲工", NoSite: true, Materials: strings.Repeat("这是面向团队的诊断工具材料。", 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.New(db).Run(ctx, p.Slug, true); err != nil {
		t.Fatal(err)
	}
	svc := New(db, NewAsker())
	svc.BypassAvailable = true
	// Both engines must be inside Ask at the same time to pass the barrier;
	// a sequential run would time out here.
	var arrived atomic.Int32
	both := make(chan struct{})
	var progress atomic.Int64
	svc.Ask = func(platform, question string) AskResult {
		if arrived.Add(1) == 2 {
			close(both)
		}
		select {
		case <-both:
		case <-time.After(3 * time.Second):
			return AskResult{Error: "engines were not asked in parallel"}
		}
		if arrived.Load() > 2 {
			var r model.SampleRun
			db.Where("project_id = ?", p.ID).Order("id desc").First(&r)
			if int64(r.Succeeded) > progress.Load() {
				progress.Store(int64(r.Succeeded))
			}
		}
		return AskResult{OK: true, Answer: "甲工不错"}
	}
	res, err := svc.Run(ctx, p.Slug, RunInput{Platforms: []string{"deepseek", "kimi"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	var run model.SampleRun
	db.First(&run, res.RunID)
	if run.Succeeded != 4 || run.Failed != 0 {
		t.Fatalf("run %d ok / %d failed, want 4 / 0", run.Succeeded, run.Failed)
	}
	if progress.Load() == 0 {
		t.Fatal("no progress was saved while the run was going")
	}
	var n int64
	db.Model(&model.Sample{}).Where("run_id = ?", res.RunID).Count(&n)
	if n != 4 {
		t.Fatalf("%d samples saved, want 4", n)
	}
}
