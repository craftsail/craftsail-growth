// SPDX-License-Identifier: AGPL-3.0-or-later

package plan

import (
	"context"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
	"time"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestReleaseFreezesWindowsAndKeepsTaskState(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	s := New(db)
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Observation", URL: "https://observe.test/"})
	if err != nil {
		t.Fatal(err)
	}
	task := model.Task{ProjectID: p.ID, Code: "S-001", Status: model.TaskDone, Acceptance: map[string]any{"type": "manual"}}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	prop := p.Site + "/"
	target := p.Site + "/a"
	from := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	through := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	store := &repo.Webstats{DB: db}
	report := model.WebSyncReport{ProjectID: p.ID, Source: "gsc", Property: prop, Report: "page", SearchType: "web", Version: 2, Token: "release", From: from, Through: through}
	if err := store.BeginSyncReport(ctx, &report); err != nil {
		t.Fatal(err)
	}
	batch := repo.SyncBatch{Quality: model.GoogleQuality{Known: true, Aggregations: []string{"byPage"}}, GSC: []model.GscFact{{ProjectID: p.ID, Property: prop, Slice: "page", SearchType: "web", Day: from, Page: target, KeyHash: "before", Clicks: 100, Impressions: 2000}, {ProjectID: p.ID, Property: prop, Slice: "page", SearchType: "web", Day: through, Page: target, KeyHash: "after", Clicks: 200, Impressions: 2000}, {ProjectID: p.ID, Property: prop, Slice: "page", SearchType: "web", Day: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Page: target, KeyHash: "release-day", Clicks: 9999, Impressions: 99999}}}
	if err := store.ReplaceSyncBatch(ctx, report, from, through, batch); err != nil {
		t.Fatal(err)
	}
	release := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC).Unix()
	o, err := s.Release(ctx, p.Slug, task.Code, model.Observation{Hypothesis: "A clearer answer improves clicks", Metric: "clicks", URLs: []string{target}, ReleasedAt: release})
	if err != nil {
		t.Fatal(err)
	}
	if o.Baseline.Value != 100 || o.BaselineFrom != "2026-06-03" || o.FollowupFrom != "2026-07-02" || o.FollowupThrough != "2026-07-29" {
		t.Fatalf("windows %+v", o)
	}
	early, err := s.Evaluate(ctx, p.Slug, task.Code, o.ID, true, "")
	if err != nil || early.Conclusion != "pending" || early.ID != 0 {
		t.Fatalf("early %+v %v", early, err)
	}
	if err := db.Model(&model.GscFact{}).Where("key_hash = ?", "before").Update("clicks", 999).Error; err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	result, err := s.Evaluate(ctx, p.Slug, task.Code, o.ID, true, "Reviewed guardrails")
	if err != nil || result.Conclusion != "improved" || result.Followup.Value != 200 {
		t.Fatalf("result %+v %v", result, err)
	}
	saved, _ := s.tasks.ByCode(ctx, p.ID, task.Code)
	if saved.Status != model.TaskDone {
		t.Fatal("observation changed task", saved.Status)
	}
	if err := db.Model(&model.GscFact{}).Where("key_hash = ?", "after").Update("clicks", 10).Error; err != nil {
		t.Fatal(err)
	}
	second, err := s.Evaluate(ctx, p.Slug, task.Code, o.ID, true, "Correction")
	if err != nil || second.Conclusion != "declined" {
		t.Fatalf("second %+v %v", second, err)
	}
	list, err := s.Observations(ctx, p.Slug, task.Code)
	if err != nil || len(list) != 1 || len(list[0].Results) != 2 || list[0].Baseline.Value != 100 || list[0].Results[1].Followup.Value != 200 {
		t.Fatalf("history %+v %v", list, err)
	}
	if _, err := s.Evaluate(ctx, p.Slug, "wrong", o.ID, true, ""); !errors.Is(err, ErrObservationMissing) {
		t.Fatal("wrong task", err)
	}
	bad := model.Observation{Metric: "clicks", Hypothesis: "x", URLs: []string{target}, ControlURLs: []string{target}, ReleasedAt: release}
	if _, err := s.Release(ctx, p.Slug, task.Code, bad); !errors.Is(err, ErrObservation) {
		t.Fatal("overlapping control", err)
	}
}
func TestAIComparisonRequiresFixedModelsPromptsAndMix(t *testing.T) {
	rows := []model.Sample{}
	for i := 0; i < 40; i++ {
		rows = append(rows, model.Sample{OK: true, QID: "Q1", QuestionText: "Which tool?", Platform: "test", SampleMode: "api", Mentioned: i < 5, Raw: map[string]any{"model": "m1", "searched": false}})
	}
	base := aiMeasurement(rows, []string{"Q1"})
	if !base.Valid || base.Count != 40 {
		t.Fatal(base)
	}
	for i := range rows {
		rows[i].Mentioned = i < 35
	}
	after := aiMeasurement(rows, []string{"Q1"})
	o := model.Observation{Metric: "visibility", Baseline: base}
	if verdict, _ := conclude(o, after); verdict != "improved" {
		t.Fatal(verdict)
	}
	rows[0].Raw = map[string]any{"model": "m2", "searched": false}
	if changed := aiMeasurement(rows, []string{"Q1"}); changed.Valid || changed.Reason != "sample_changed" {
		t.Fatal(changed)
	}
	rows[0].Raw = nil
	if unknown := aiMeasurement(rows, []string{"Q1"}); unknown.Valid {
		t.Fatal(unknown)
	}
	o.Baseline.Count = 20
	if verdict, _ := conclude(o, after); verdict != "insufficient" {
		t.Fatal(verdict)
	}
}
func TestTechnicalObservationRejectsOldAudit(t *testing.T) {
	o := model.Observation{Metric: "technical", ReleasedAt: 500}
	for _, m := range []model.Measurement{{Valid: true, AuditAt: 499}, {Valid: false, AuditAt: 600}} {
		if v, _ := conclude(o, m); v != "pending" {
			t.Fatal(v)
		}
	}
	if v, _ := conclude(o, model.Measurement{Valid: true, AuditAt: 600}); v != "improved" {
		t.Fatal(v)
	}
}
