// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func firstDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestFirstCheckSavesAuditWithoutOptionalServices(t *testing.T) {
	for _, reachable := range []bool{true, false} {
		t.Run(fmt.Sprint(reachable), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !reachable {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if r.URL.Path == "/robots.txt" {
					fmt.Fprint(w, "User-agent: *\nAllow: /\n")
					return
				}
				if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/page-") {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, "<html><head><title>Tools</title></head><body><h1>Tools</h1><p>Tools for independent teams.</p>")
				for i := 0; i < 10; i++ {
					fmt.Fprintf(w, `<a href="/page-%d">Page</a>`, i)
				}
				fmt.Fprint(w, "</body></html>")
			}))
			defer server.Close()
			db := firstDB(t)
			ctx := context.Background()
			p, err := project.New(db).Create(ctx, project.CreateInput{Name: "First", URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			s := New(db)
			s.crawl.Delay = 0
			s.crawl.HTTP.Sleep = func() {}
			// Any accidental call to an optional phase would panic.
			s.bootstrap = nil
			s.sample = nil
			s.web = nil
			s.report = nil
			s.verify = nil
			var stages []string
			s.Log = func(line string) { stages = append(stages, line) }
			err = s.FirstCheck(ctx, p.Slug, 1000)
			if reachable && err != nil {
				t.Fatal(err)
			}
			if !reachable && (err == nil || !strings.Contains(err.Error(), "no page returned 200")) {
				t.Fatalf("missing reachability failure %v", err)
			}
			audit, err := s.audit.Latest(ctx, p.Slug)
			if err != nil || audit == nil || audit.PageCount < 1 || audit.PageCount > 5 {
				t.Fatalf("audit %#v %v", audit, err)
			}
			if len(stages) < 2 || stages[0] != "=== 1/2 crawl ===" || stages[1] != "=== 2/2 audit ===" {
				t.Fatalf("%v", stages)
			}
			if !reachable && len(audit.Findings) == 0 {
				t.Fatal("failed access evidence was lost")
			}
			fresh, _ := s.projects.Get(ctx, p.Slug)
			if fresh.Bootstrap != nil {
				t.Fatal("first check altered brand draft")
			}
		})
	}
}

func TestFirstCheckNoSiteDoesNotInventAudit(t *testing.T) {
	db := firstDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Offline", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db)
	s.crawl = nil
	s.audit = nil
	if err := s.FirstCheck(ctx, p.Slug, 5); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.Audit{}).Count(&count)
	if count != 0 {
		t.Fatal("invented website audit")
	}
}
