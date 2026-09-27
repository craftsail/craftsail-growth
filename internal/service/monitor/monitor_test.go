// SPDX-License-Identifier: AGPL-3.0-or-later

package monitor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func monDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func dueProject(t *testing.T, db *gorm.DB) *model.Project {
	t.Helper()
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "周期项目", NoSite: true, Materials: "周期项目。"})
	if err != nil {
		t.Fatal(err)
	}
	every := 7
	past := time.Now().Add(-time.Hour)
	p.MonitorEveryDays = &every
	p.MonitorNextRun = &past
	if err := (&repo.Projects{DB: db}).Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	return p
}

func waitJob(t *testing.T, js *jobs.Service, id uint64) *model.Job {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		j, err := js.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if j.Status != "running" && j.Status != "queued" {
			return j
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("job did not finish")
	return nil
}

func TestTickDoesNotAdvanceWhilePeriodRuns(t *testing.T) {
	db := monDB(t)
	p := dueProject(t, db)
	past := *p.MonitorNextRun
	gate := make(chan struct{})
	js := jobs.New(db)
	js.Register(jobs.PeriodAction, func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		select {
		case <-gate:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	attach(db, js)
	Tick(db, js)

	got, err := project.New(db).Get(context.Background(), p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.MonitorNextRun == nil || got.MonitorNextRun.After(time.Now()) {
		t.Fatalf("next run advanced before finish: %v", got.MonitorNextRun)
	}
	close(gate)
	rows, _, err := js.Recent(context.Background(), p.Slug, 5)
	if err != nil || len(rows) == 0 {
		t.Fatal(err, rows)
	}
	done := waitJob(t, js, rows[0].ID)
	if done.Status != "done" {
		t.Fatalf("status %s", done.Status)
	}
	got, err = project.New(db).Get(context.Background(), p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.MonitorNextRun == nil || !got.MonitorNextRun.After(time.Now().Add(6*24*time.Hour)) {
		t.Fatalf("next run %v, past was %v", got.MonitorNextRun, past)
	}
}

func TestFailedPeriodStaysOnTheJob(t *testing.T) {
	db := monDB(t)
	p := dueProject(t, db)
	js := jobs.New(db)
	js.Register(jobs.PeriodAction, func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		return errors.New("抓取失败")
	})
	attach(db, js)
	Tick(db, js)
	rows, _, err := js.Recent(context.Background(), p.Slug, 5)
	if err != nil || len(rows) == 0 {
		t.Fatal(err, rows)
	}
	done := waitJob(t, js, rows[0].ID)
	if done.Status != "failed" || done.Error == "" {
		t.Fatalf("status %s error %q", done.Status, done.Error)
	}
	got, err := project.New(db).Get(context.Background(), p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.MonitorNextRun == nil || !got.MonitorNextRun.After(time.Now().Add(6*24*time.Hour)) {
		t.Fatalf("failed period should schedule the next try after it finishes, got %v", got.MonitorNextRun)
	}
}
