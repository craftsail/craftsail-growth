// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestBudgetContinuationFillsHistoryWithoutRefreshingTail(t *testing.T) {
	s := New(testDB(t))
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	started := time.Now().Add(-time.Second)
	seen := map[string]int{}
	for i := 0; i < 7; i++ {
		ctx, cancel, _ := withSyncBudget(context.Background(), BatchOptions{RefreshBefore: started, MaxBatches: 1})
		err := s.syncReport(ctx, 1, "ga4", "123", "landing", "", day.AddDate(0, 0, -6), day, 1, 7, func(_ context.Context, p dateChunk) (repo.SyncBatch, error) {
			seen[p.start.Format("2006-01-02")]++
			return repo.SyncBatch{}, nil
		})
		cancel()
		if i < 6 && !errors.Is(err, ErrSyncBudget) || i == 6 && err != nil {
			t.Fatalf("turn %d: %v", i, err)
		}
	}
	if len(seen) != 7 {
		t.Fatalf("history stalled: %v", seen)
	}
	for _, n := range seen {
		if n != 1 {
			t.Fatalf("tail fetched repeatedly: %v", seen)
		}
	}
	rows, _ := s.rows.SyncReports(context.Background(), 1, "ga4", "123")
	if len(rows) != 1 || rows[0].State != "completed" {
		t.Fatalf("state %#v", rows)
	}
}
func TestPaginationBudgetRetainsPreviousSnapshot(t *testing.T) {
	s := New(testDB(t))
	day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	old := model.GaFact{ProjectID: 1, Property: "123", Report: "landing", Day: day, Landing: "/old", Sessions: 9}
	if err := s.rows.UpsertGaFacts(context.Background(), []model.GaFact{old}); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.Client = &Client{GALimit: 1, HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (int, string) {
		calls++
		return 200, `{"rowCount":2,"rows":[{"dimensionValues":[{"value":"20260920"},{"value":"/new"}],"metricValues":[{"value":"1"},{"value":"1"},{"value":"0"},{"value":"5"}]}]}`
	})}}
	ctx, cancel, _ := withSyncBudget(context.Background(), BatchOptions{MaxRequests: 1})
	defer cancel()
	err := s.syncReport(ctx, 1, "ga4", "123", "landing", "", day, day, 7, 1, func(ctx context.Context, p dateChunk) (repo.SyncBatch, error) {
		rows, q, err := s.Client.FetchGAReport(ctx, "token", "123", "landing", "2026-09-20", "2026-09-20")
		return repo.SyncBatch{GA: rows, Quality: q}, err
	})
	if !errors.Is(err, ErrSyncBudget) || calls != 1 {
		t.Fatalf("limit %d %v", calls, err)
	}
	var facts []model.GaFact
	s.rows.DB.Find(&facts)
	if len(facts) != 1 || facts[0].Landing != "/old" {
		t.Fatalf("lost old data %#v", facts)
	}
	reports, _ := s.rows.SyncReports(context.Background(), 1, "ga4", "123")
	days, _ := s.rows.SyncDays(context.Background(), reports[0].ID, day, day)
	if len(days) != 0 || reports[0].State != "backfilling" || reports[0].ErrorClass != "budget" {
		t.Fatalf("coverage %#v %#v", reports, days)
	}
}
func TestBudgetDeadlineAndParentCancellationDiffer(t *testing.T) {
	for _, parentCancel := range []bool{false, true} {
		t.Run(fmt.Sprint(parentCancel), func(t *testing.T) {
			s := New(testDB(t))
			parent, cancelParent := context.WithCancel(context.Background())
			defer cancelParent()
			duration := 30 * time.Millisecond
			if parentCancel {
				duration = time.Second
			}
			ctx, cancel, _ := withSyncBudget(parent, BatchOptions{MaxDuration: duration})
			defer cancel()
			day := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
			err := s.syncReport(ctx, 1, "ga4", "123", "landing", "", day, day, 1, 1, func(ctx context.Context, _ dateChunk) (repo.SyncBatch, error) {
				if parentCancel {
					cancelParent()
				}
				<-ctx.Done()
				return repo.SyncBatch{}, ctx.Err()
			})
			want := ErrSyncBudget
			state := "backfilling"
			if parentCancel {
				want = context.Canceled
				state = "paused"
			}
			if !errors.Is(err, want) {
				t.Fatalf("error %v want %v", err, want)
			}
			reports, _ := s.rows.SyncReports(context.Background(), 1, "ga4", "123")
			if len(reports) != 1 || reports[0].State != state {
				t.Fatalf("state %#v", reports)
			}
		})
	}
	real := errors.New("quota or storage failure")
	if err := withoutBudgetError(errors.Join(real, ErrSyncBudget)); !errors.Is(err, real) {
		t.Fatal("budget hid real failure", err)
	}
}
func TestRunBatchReportsPendingAndKeepsMakingProgress(t *testing.T) {
	t.Setenv("GOOGLE_SA_JSON", "test-account")
	t.Setenv("GOOGLE_REFRESH_TOKEN", "")
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Batch", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	if err := s.projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	// A cached test token keeps all network access inside the fake transport.
	s.Client.store("test-account", "fixture-token", time.Now().Add(time.Hour))
	calls := 0
	s.Client.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
		calls++
		if req.Method == "GET" {
			return 200, `{"timeZone":"UTC"}`
		}
		return 200, `{"rowCount":0,"rows":[],"metadata":{"timeZone":"UTC"}}`
	})}
	start := time.Now().Add(-time.Second)
	for i := 0; i < 2; i++ {
		result, err := s.RunBatch(ctx, p.Slug, BatchOptions{RefreshBefore: start, MaxRequests: 2, MaxBatches: 1})
		if err != nil || result == nil || !result.Pending || result.Requests > 2 || result.Batches > 1 {
			t.Fatalf("turn %d %#v %v", i, result, err)
		}
	}
	reports, _ := s.rows.SyncReports(ctx, p.ID, "ga4", "123")
	var daily model.WebSyncReport
	for _, r := range reports {
		if r.Report == "daily" {
			daily = r
		}
	}
	days, err := s.rows.SyncDays(ctx, daily.ID, daily.From, daily.Through)
	if err != nil || len(days) != 56 {
		t.Fatalf("expected two distinct 28-day batches: %d %v", len(days), err)
	}
	if calls > 4 {
		t.Fatalf("request budget %d", calls)
	}
}

func TestRunBatchStopsWhenRequestBudgetCannotMakeProgress(t *testing.T) {
	t.Setenv("GOOGLE_SA_JSON", "test-account")
	t.Setenv("GOOGLE_REFRESH_TOKEN", "")
	s := New(testDB(t))
	ctx := context.Background()
	p, err := s.projects.Create(ctx, project.CreateInput{Name: "No progress", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	p.GA4Property = "123"
	if err := s.projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	s.Client.store("test-account", "fixture-token", time.Now().Add(time.Hour))
	s.Client.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
		if req.Method != "GET" {
			t.Fatal("request exceeded budget")
		}
		return 200, `{"timeZone":"UTC"}`
	})}
	result, err := s.RunBatch(ctx, p.Slug, BatchOptions{MaxRequests: 1})
	if result == nil || !result.Pending || err == nil || !strings.Contains(err.Error(), "Automatic continuation stopped") {
		t.Fatalf("unbounded continuation %#v %v", result, err)
	}
}

func TestBudgetKeepsGAReportsAheadOfGSCSecondaryHistory(t *testing.T) {
	s := New(testDB(t))
	ctx, cancel, _ := withSyncBudget(context.Background(), BatchOptions{MaxBatches: 2})
	defer cancel()
	var dims [][]gaName
	s.Client.HTTP = &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
		if req.URL.Host != "analyticsdata.googleapis.com" {
			t.Fatal("GSC displaced first GA batch", req.URL)
		}
		var body struct{ Dimensions, Metrics []gaName }
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(req.URL.Path, ":checkCompatibility") {
			var dimensions, metrics []any
			for _, v := range body.Dimensions {
				dimensions = append(dimensions, map[string]any{"dimensionMetadata": map[string]string{"apiName": v.Name}, "compatibility": "COMPATIBLE"})
			}
			for _, v := range body.Metrics {
				metrics = append(metrics, map[string]any{"metricMetadata": map[string]string{"apiName": v.Name}, "compatibility": "COMPATIBLE"})
			}
			raw, _ := json.Marshal(map[string]any{"dimensionCompatibilities": dimensions, "metricCompatibilities": metrics})
			return 200, string(raw)
		}
		dims = append(dims, body.Dimensions)
		return 200, `{"rows":[],"metadata":{"timeZone":"UTC"}}`
	})}
	_, err := s.SyncFacts(ctx, 1, "token", "sc-domain:a.com", "123", time.Now())
	if !errors.Is(err, ErrSyncBudget) || len(dims) != 2 || len(dims[0]) != 4 {
		t.Fatalf("priority %#v %v", dims, err)
	}
}
