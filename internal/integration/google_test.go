// SPDX-License-Identifier: AGPL-3.0-or-later

package integration

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"gorm.io/gorm"
)

// Exercise a failure after DELETE, inside the real database transaction.
func rejectInsert(t *testing.T, db *gorm.DB, table string) {
	t.Helper()
	name := "reject_" + table
	sql := "CREATE TRIGGER " + name + " BEFORE INSERT ON " + table + " BEGIN SELECT RAISE(FAIL, 'injected failure'); END"
	if db.Dialector.Name() == "mysql" {
		sql = "CREATE TRIGGER " + name + " BEFORE INSERT ON " + table + " FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected failure'"
	}
	if err := db.Exec(sql).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Exec("DROP TRIGGER " + name).Error; err != nil {
			t.Error(err)
		}
	})
}

func testGoogleReplacement(t *testing.T, db *gorm.DB) {
	ctx := context.Background()
	r := &repo.Webstats{DB: db}
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	report := model.WebSyncReport{ProjectID: 10, Property: "properties/10", Source: "ga4", Report: "channel", Version: 3, Token: "first", From: day, Through: day}
	if err := r.BeginSyncReport(ctx, &report); err != nil {
		t.Fatal(err)
	}
	fact := model.GaFact{ProjectID: 10, Property: report.Property, Report: report.Report, Day: day, Source: "old", Sessions: 3}
	if err := r.ReplaceSyncBatch(ctx, report, day, day, repo.SyncBatch{GA: []model.GaFact{fact}, Quality: model.GoogleQuality{Known: true}}); err != nil {
		t.Fatal(err)
	}
	before, err := r.SyncDays(ctx, report.ID, day, day)
	if err != nil || len(before) != 1 {
		t.Fatalf("initial coverage: %v %v", before, err)
	}
	stale := report
	report.Token = "second"
	if err := r.BeginSyncReport(ctx, &report); err != nil {
		t.Fatal(err)
	}
	if err := r.ReplaceSyncBatch(ctx, stale, day, day, repo.SyncBatch{}); !errors.Is(err, repo.ErrSyncSuperseded) {
		t.Fatalf("stale write accepted: %v", err)
	}
	if err := r.FinishSyncReport(ctx, stale, "completed", ""); !errors.Is(err, repo.ErrSyncSuperseded) {
		t.Fatalf("stale finish accepted: %v", err)
	}
	// Nested cleanup removes the trigger before testing successful replacement.
	t.Run("rollback", func(t *testing.T) {
		rejectInsert(t, db, "ga_facts")
		fact.Source = "new"
		if err := r.ReplaceSyncBatch(ctx, report, day, day, repo.SyncBatch{GA: []model.GaFact{fact}}); err == nil {
			t.Fatal("expected injected failure")
		}
		var rows []model.GaFact
		if err := db.Where("project_id = ?", 10).Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		after, err := r.SyncDays(ctx, report.ID, day, day)
		if err != nil || !reflect.DeepEqual(before, after) || len(rows) != 1 || rows[0].Source != "old" {
			t.Fatalf("rollback lost facts/coverage: %v %v %v", rows, after, err)
		}
	})
	other := fact
	other.ProjectID = 11
	if err := r.UpsertGaFacts(ctx, []model.GaFact{other}); err != nil {
		t.Fatal(err)
	}
	if err := r.ReplaceSyncBatch(ctx, report, day, day, repo.SyncBatch{}); err != nil {
		t.Fatal(err)
	}
	var own, untouched int64
	if err := db.Model(&model.GaFact{}).Where("project_id = ?", 10).Count(&own).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.GaFact{}).Where("project_id = ?", 11).Count(&untouched).Error; err != nil {
		t.Fatal(err)
	}
	days, err := r.SyncDays(ctx, report.ID, day, day)
	if err != nil || own != 0 || untouched != 1 || len(days) != 1 {
		t.Fatalf("empty success/scope: %d %d %v %v", own, untouched, days, err)
	}
}

func testAnalysisIdentity(t *testing.T, db *gorm.DB) {
	ctx := context.Background()
	r := &repo.Webstats{DB: db}
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	var facts []model.GaFact
	for i, event := range []string{"sign_up", "Sign_Up", "signXup"} {
		facts = append(facts, model.GaFact{ProjectID: 20, Property: "properties/20", Report: "landing_event", Day: day, Landing: "/Pricing", Country: "Brazil", Device: "mobile", EventName: event, EventCount: float64(i + 1)})
	}
	if err := r.UpsertGaFacts(ctx, facts); err != nil {
		t.Fatal(err)
	}
	f := repo.GAFilter{ProjectID: 20, Property: "properties/20", Report: "landing_event", From: day, Through: day, PreviousFrom: day.AddDate(0, 0, -1), Country: "Brazil", Device: "mobile", Events: []string{"sign_up"}, Sort: "event_count", Direction: "desc", Limit: 1}
	rows, total, err := r.GAMetrics(ctx, f)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].EventName != "sign_up" || rows[0].EventCount != 1 {
		t.Fatalf("exact event filter: %v total=%d err=%v", rows, total, err)
	}
	var exported []repo.GAMetric
	if err := r.WalkGAMetrics(ctx, f, func(v repo.GAMetric) error { exported = append(exported, v); return nil }); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows, exported) {
		t.Fatal("event export differs from filtered view")
	}
	f.Events = nil
	f.Text = "sign_up"
	_, total, err = r.GAMetrics(ctx, f)
	if err != nil || total != 2 {
		t.Fatalf("literal underscore search: total=%d err=%v", total, err)
	}

	pages := []string{"https://example.com/Pricing", "https://example.com/pricing"}
	var gsc []model.GscFact
	for i, page := range pages {
		gsc = append(gsc, model.GscFact{ProjectID: 20, Property: "sc-domain:example.com", Slice: "page", SearchType: "web", Day: day, Page: page, Clicks: float64(i + 1), Impressions: 10})
	}
	if err := r.UpsertGscFacts(ctx, gsc); err != nil {
		t.Fatal(err)
	}
	sf := repo.SearchFilter{ProjectID: 20, Property: gsc[0].Property, Slice: "page", Group: "page", From: day, Through: day, PreviousFrom: day.AddDate(0, 0, -1), Limit: 1, Sort: "clicks", Direction: "desc"}
	search, total, err := r.SearchMetrics(ctx, sf)
	if err != nil || total != 2 || len(search) != 1 || search[0].Name != pages[1] {
		t.Fatalf("URL grouping/paging: %v %d %v", search, total, err)
	}
	var all []repo.SearchMetric
	if err := r.WalkSearchMetrics(ctx, sf, func(v repo.SearchMetric) error { all = append(all, v); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0] != search[0] {
		t.Fatalf("export clipped to page: %v", all)
	}
	sf.ExactPage = pages[0]
	search, total, err = r.SearchMetrics(ctx, sf)
	if err != nil || total != 1 || len(search) != 1 || search[0].Name != pages[0] {
		t.Fatalf("exact page filter: %v %d %v", search, total, err)
	}
	candidates, err := r.SearchPageURLs(ctx, 20, gsc[0].Property, day, day)
	if err != nil || len(candidates) != 2 {
		t.Fatalf("case-sensitive candidates: %v %v", candidates, err)
	}
}

func testConcurrentQuota(t *testing.T, db *gorm.DB) {
	ctx := context.Background()
	const now int64 = 1800000000
	const workers = 24
	start := make(chan struct{})
	outcomes := make(chan int64, workers)
	failures := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			r := &repo.Webstats{DB: db}
			retry, err := r.ReserveGoogleRequest(ctx, "sc-domain:quota.com", "traffic/gsc", now, now, now+86400, 100, 5)
			if err != nil {
				failures <- err
				return
			}
			outcomes <- retry
		}()
	}
	close(start)
	wg.Wait()
	close(outcomes)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	allowed := 0
	for retry := range outcomes {
		if retry == 0 {
			allowed++
		} else if retry != now+60 {
			t.Errorf("unexpected retry %d", retry)
		}
	}
	if allowed != 5 {
		t.Fatalf("concurrent requests exceeded/lost quota: %d", allowed)
	}
	r := &repo.Webstats{DB: db}
	row, err := r.GoogleQuota(ctx, "sc-domain:quota.com", "traffic/gsc")
	if err != nil || row == nil || row.DailyUsed != 5 || row.MinuteUsed != 5 {
		t.Fatalf("persisted quota: %v %v", row, err)
	}
	if err := r.BackoffGoogle(ctx, "sc-domain:quota.com", "traffic/gsc", now+900); err != nil {
		t.Fatal(err)
	}
	if err := r.BackoffGoogle(ctx, "sc-domain:quota.com", "traffic/gsc", now+300); err != nil {
		t.Fatal(err)
	}
	retry, err := r.ReserveGoogleRequest(ctx, "sc-domain:quota.com", "traffic/gsc", now+60, now, now+86400, 100, 5)
	if err != nil || retry != now+900 {
		t.Fatalf("shorter backoff overwrote longer: %d %v", retry, err)
	}
	retry, err = r.ReserveGoogleRequest(ctx, "sc-domain:quota.com", "traffic/ga", now+60, now, now+86400, 100, 5)
	if err != nil || retry != 0 {
		t.Fatalf("unrelated quota family blocked: %d %v", retry, err)
	}
	retry, err = r.ReserveGoogleRequest(ctx, "sc-domain:quota.com", "traffic/gsc", now+86400, now+86400, now+172800, 100, 5)
	if err != nil || retry != 0 {
		t.Fatalf("quota did not reset: %d %v", retry, err)
	}
}

func testIndexHistory(t *testing.T, db *gorm.DB) {
	r := &repo.Webstats{DB: db}
	ctx := context.Background()
	now := time.Unix(1800000000, 0)
	a := model.IndexURL{ProjectID: 1, Property: "sc-domain:a.com", URL: "https://a.com/p", FromCrawl: true, FirstSeenAt: now.Unix(), LastSeenAt: now.Unix()}
	b := a
	b.Property = "https://a.com/"
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{a, b}); err != nil {
		t.Fatal(err)
	}
	due, _ := r.DueIndexURLs(ctx, 1, a.Property, now.Unix(), 30)
	if len(due) != 1 {
		t.Fatal(due)
	}
	a = due[0]
	result := &model.GscIndex{Verdict: "PASS", GoogleCanonical: a.URL, UserCanonical: a.URL, Raw: `{"provider":"evidence"}`}
	if err := r.RecordInspection(ctx, a, result, "", now); err != nil {
		t.Fatal(err)
	}
	if err := r.RecordInspection(ctx, a, nil, "gsc HTTP 500", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	// Re-discovery preserves inspection, timestamps and both discovery sources.
	rediscovered := a
	rediscovered.FromCrawl = false
	rediscovered.FromSearch = true
	rediscovered.FirstSeenAt = now.Add(time.Hour).Unix()
	if err := r.DiscoverIndexURLs(ctx, []model.IndexURL{rediscovered}); err != nil {
		t.Fatal(err)
	}
	inventory, err := r.IndexInventory(ctx, 1, a.Property, "", "error", 1, 50, now.Unix())
	if err != nil || len(inventory.Items) != 1 {
		t.Fatalf("%#v %v", inventory, err)
	}
	got := inventory.Items[0]
	if !got.FromCrawl || !got.FromSearch || got.FirstSeenAt != now.Unix() || got.Verdict != "PASS" || got.FirstIndexedAt == nil || *got.FirstIndexedAt != now.Unix() || got.LastSuccessAt != now.Unix() || got.Latest.GoogleCanonical != a.URL {
		t.Fatalf("%#v", got)
	}
	hist, err := r.IndexHistory(ctx, 1, a.Property, a.URL, 1, 1)
	if err != nil || hist.Total != 2 || len(hist.Items) != 1 || hist.Items[0].Error == "" || hist.Items[0].Result != nil {
		t.Fatalf("%#v %v", hist, err)
	}
	other, _ := r.IndexHistory(ctx, 1, b.Property, a.URL, 1, 50)
	if other.Total != 0 {
		t.Fatal("property leaked")
	}
	other, _ = r.IndexHistory(ctx, 2, a.Property, a.URL, 1, 50)
	if other.Total != 0 {
		t.Fatal("project leaked")
	}
	latest, _ := r.ListIndex(ctx, 1, a.Property)
	if len(latest) != 1 || latest[0].Verdict != "PASS" || latest[0].Raw == "" {
		t.Fatal(latest)
	}
	// A failed history write rolls back the latest-result overwrite as well.
	rejectInsert(t, db, "index_inspections")
	if err := r.RecordInspection(ctx, a, &model.GscIndex{Verdict: "FAIL"}, "", now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected rollback")
	}
	latest, _ = r.ListIndex(ctx, 1, a.Property)
	if latest[0].Verdict != "PASS" {
		t.Fatal("partial write")
	}
	hist, _ = r.IndexHistory(ctx, 1, a.Property, a.URL, 1, 50)
	if hist.Total != 2 {
		t.Fatal("partial history")
	}
}

func testReleaseAndReport(t *testing.T, db *gorm.DB) {
	ctx := context.Background()
	task := model.Task{ProjectID: 30, Code: "S01", Status: model.TaskDoing}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	r := &repo.Observations{DB: db}
	baseline := model.Measurement{Valid: true, Value: 5, Exposure: 100, Signature: "frozen", Strata: map[string][2]int{"br/mobile": {5, 100}}}
	release := model.Observation{ProjectID: 30, TaskID: task.ID, TaskCode: task.Code, URLs: []string{"https://example.com/Pricing"}, Baseline: baseline, ReleasedAt: 1800000000, BaselineFrom: "2026-09-01", BaselineThrough: "2026-09-28", FollowupFrom: "2026-10-01", FollowupThrough: "2026-10-28"}
	if err := r.Create(ctx, &release); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"incomplete", "observed"} {
		result := model.ObservationResult{ProjectID: 30, ObservationID: release.ID, Reason: reason, Followup: model.Measurement{Value: 7}}
		if err := r.Append(ctx, &result); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := r.List(ctx, 30, task.Code)
	if err != nil || len(loaded) != 1 {
		t.Fatalf("release read: %v %v", loaded, err)
	}
	got := loaded[0]
	if !reflect.DeepEqual(got.Baseline, baseline) || !reflect.DeepEqual(got.URLs, release.URLs) || got.FollowupThrough != release.FollowupThrough || len(got.Results) != 2 || got.Results[0].Reason != "observed" || got.Results[1].Reason != "incomplete" {
		t.Fatalf("snapshot/history roundtrip: %+v", got)
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.Status != model.TaskDoing || task.ReleasedAt == nil || *task.ReleasedAt != release.ReleasedAt {
		t.Fatalf("release changed task status: %+v", task)
	}
	hidden, err := r.List(ctx, 31, task.Code)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("release scope: %v %v", hidden, err)
	}

	reports := &repo.Reports{DB: db}
	east := time.FixedZone("UTC+8", 8*60*60)
	for i, lang := range []string{"zh", "pt"} {
		report := model.Report{ProjectID: 30, ReportOn: time.Date(2026, 10, 1, i, 0, 0, 0, east), Language: lang, Markdown: lang}
		if err := reports.Upsert(ctx, &report); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&model.Report{}).Where("project_id = ?", 30).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	report, err := reports.Latest(ctx, 30)
	if err != nil || count != 1 || report == nil || report.ReportOn.Format("2006-01-02") != "2026-10-01" || report.Language != "pt" || report.Markdown != "pt" {
		t.Fatalf("calendar upsert: %v count=%d err=%v", report, count, err)
	}
}
