// SPDX-License-Identifier: AGPL-3.0-or-later

package monitor

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
)

func Tick(db *gorm.DB, js *jobs.Service) {
	if db == nil || js == nil {
		return
	}
	ctx := context.Background()
	ps, err := (&repo.Projects{DB: db}).List(ctx)
	if err != nil {
		log.Println("monitor list:", err)
		return
	}
	today := time.Now()
	for i := range ps {
		p := &ps[i]
		if p.MonitorEveryDays == nil || *p.MonitorEveryDays <= 0 {
			continue
		}
		if p.MonitorNextRun != nil && p.MonitorNextRun.After(today) {
			continue
		}
		_, err := js.Start(ctx, p.Slug, jobs.PeriodAction, map[string]any{"monitor": true})
		if err != nil && !errors.Is(err, jobs.ErrBusy) {
			log.Println("monitor start:", err)
		}
	}
}

func attach(db *gorm.DB, js *jobs.Service) {
	if js == nil {
		return
	}
	js.OnFinish(func(j *model.Job) {
		if !monitorJob(j) {
			return
		}
		if j.Status != "done" && j.Status != "failed" && j.Status != "stopped" {
			return
		}
		advance(db, j)
		// A full scheduled period leaves indexing to its own resumable job, so
		// traffic backfill and inspection quota waits do not rerun the period.
		if j.Status == "done" {
			p, err := (&repo.Projects{DB: db}).ByID(context.Background(), *j.ProjectID)
			if err == nil && p != nil && ((!p.NoSite && p.Site != "") || p.GscSite != "") {
				if _, err := js.Start(context.Background(), p.Slug, "indexing", map[string]any{"monitor": true}); err != nil && !errors.Is(err, jobs.ErrBusy) {
					log.Println("monitor indexing:", err)
				}
			}
		}
	})
}

func monitorJob(j *model.Job) bool {
	if j == nil || j.Action != jobs.PeriodAction || j.ProjectID == nil {
		return false
	}
	switch v := j.Args["monitor"].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	default:
		return false
	}
}

func advance(db *gorm.DB, j *model.Job) {
	ctx := context.Background()
	var p model.Project
	if err := db.WithContext(ctx).First(&p, *j.ProjectID).Error; err != nil {
		return
	}
	if p.MonitorEveryDays == nil || *p.MonitorEveryDays <= 0 {
		return
	}
	base := time.Now()
	if j.FinishedAt != nil && *j.FinishedAt > 0 {
		base = time.Unix(*j.FinishedAt, 0)
	}
	next := base.Add(time.Duration(*p.MonitorEveryDays) * 24 * time.Hour)
	p.MonitorNextRun = &next
	_ = (&repo.Projects{DB: db}).Save(ctx, &p)
}

func Start(db *gorm.DB, js *jobs.Service) {
	attach(db, js)
	go func() {
		t := time.NewTicker(30 * time.Minute)
		defer t.Stop()
		for range t.C {
			Tick(db, js)
		}
	}()
}
