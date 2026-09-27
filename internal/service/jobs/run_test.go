// SPDX-License-Identifier: AGPL-3.0-or-later

package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func jobDB(t *testing.T) *gorm.DB {
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

func TestRejectsSecondRunningJob(t *testing.T) {
	db := jobDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{Name: "甲工", NoSite: true, Materials: "甲工是测试品牌。"})
	if err != nil {
		t.Fatal(err)
	}
	gate := make(chan struct{})
	svc := New(db)
	svc.Register("slow", func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		log("working")
		select {
		case <-gate:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	j1, err := svc.Start(context.Background(), p.Slug, "slow", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Start(context.Background(), p.Slug, "slow", nil)
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("want ErrBusy, got %v", err)
	}
	close(gate)
	waitDone(t, svc, j1.ID)
	if _, err := svc.Start(context.Background(), p.Slug, "slow", nil); err != nil {
		t.Fatal(err)
	}
}

func TestTailReturnsIncrementalLog(t *testing.T) {
	db := jobDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{Name: "乙工", NoSite: true, Materials: "乙工。"})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	svc.Register("loggy", func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		log("hello")
		log("world")
		return nil
	})
	j, err := svc.Start(context.Background(), p.Slug, "loggy", nil)
	if err != nil {
		t.Fatal(err)
	}
	waitDone(t, svc, j.ID)
	chunk, off, job, err := svc.Tail(context.Background(), j.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "done" {
		t.Fatalf("status %s", job.Status)
	}
	if !strings.Contains(chunk, "hello") || !strings.Contains(chunk, "world") {
		t.Fatalf("log %q", chunk)
	}
	more, off2, _, err := svc.Tail(context.Background(), j.ID, off)
	if err != nil {
		t.Fatal(err)
	}
	if more != "" || off2 != off {
		t.Fatalf("expected empty tail, got %q off %d", more, off2)
	}
}

func TestStopCancelsRunningJob(t *testing.T) {
	db := jobDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{Name: "丙工", NoSite: true, Materials: "丙工。"})
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	svc := New(db)
	svc.Register("block", func(ctx context.Context, slug string, args map[string]any, log func(string)) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	j, err := svc.Start(context.Background(), p.Slug, "block", nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not start")
	}
	if err := svc.Stop(context.Background(), j.ID); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, svc, j.ID, "stopped")
}

func waitDone(t *testing.T, svc *Service, id uint64) {
	t.Helper()
	waitStatus(t, svc, id, "done")
}

func waitStatus(t *testing.T, svc *Service, id uint64, want string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		j, err := svc.Get(context.Background(), id)
		if err == nil && j != nil && j.Status == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	j, _ := svc.Get(context.Background(), id)
	st := ""
	if j != nil {
		st = j.Status
	}
	t.Fatalf("job %d status %s want %s", id, st, want)
}
