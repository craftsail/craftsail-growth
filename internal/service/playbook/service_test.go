// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestStatusFromStoredData(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	s := New(db)
	s.Now = func() time.Time { return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC) }
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Quill", URL: "https://quill.test/"})
	if err != nil {
		t.Fatal(err)
	}

	st, err := s.Status(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if st.SeoStage != "seoNew" || st.AiStage != "aiReach" || st.Checks["first_check"].State != "todo" {
		t.Fatalf("new project: %+v", st)
	}

	audit := model.Audit{ProjectID: p.ID, PageCount: 3, Layers: []map[string]any{
		{"key": "access", "status": "ok"}, {"key": "discover", "status": "ok"}, {"key": "understand", "status": "warn"},
	}}
	if err := db.Create(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.AuditIssue{ProjectID: p.ID, AuditID: audit.ID, Code: "NO_LLMS_TXT"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.SetStage(ctx, p.Slug, "s0"); err != nil {
		t.Fatal(err)
	}
	if err := s.Confirm(ctx, p.Slug, "crawlUp", 1, true); err != nil {
		t.Fatal(err)
	}
	st, err = s.Status(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if st.ProductStage != "s0" || st.AiStage != "aiKnown" || st.Checks["first_check"].State != "done" ||
		st.Checks["llms_published"].State != "todo" || st.Checks["audit_access_ok"].State != "done" ||
		st.Signals["crawlUp"].State != "met" || st.SeoMet != 1 {
		t.Fatalf("after audit: %+v", st)
	}

	if err := s.SetStage(ctx, p.Slug, "s9"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad stage: %v", err)
	}
	if err := s.Confirm(ctx, p.Slug, "newFast", 1, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("automatic signal cannot be confirmed: %v", err)
	}
}
