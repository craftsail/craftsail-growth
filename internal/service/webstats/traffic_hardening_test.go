// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestAppearanceUsesDiscoveryThenFilteredDates(t *testing.T) {
	calls := 0
	c := &Client{HTTP: &http.Client{Transport: roundTrip(func(req *http.Request) (int, string) {
		var b gscFactBody
		if err := json.NewDecoder(req.Body).Decode(&b); err != nil {
			t.Fatal(err)
		}
		calls++
		if calls == 1 {
			if !reflect.DeepEqual(b.Dimensions, []string{"searchAppearance"}) || len(b.DimensionFilterGroups) != 0 || b.AggregationType != "auto" {
				t.Fatalf("discovery %#v", b)
			}
			return 200, `{"responseAggregationType":"byPage","rows":[{"keys":["AMP_BLUE_LINK"]}]}`
		}
		if !reflect.DeepEqual(b.Dimensions, []string{"date"}) || len(b.DimensionFilterGroups) != 1 || b.DimensionFilterGroups[0].Filters[0].Expression != "AMP_BLUE_LINK" || b.AggregationType != "auto" {
			t.Fatalf("detail %#v", b)
		}
		return 200, `{"responseAggregationType":"byPage","rows":[{"keys":["2026-09-01"],"clicks":4,"impressions":20}]}`
	})}}
	rows, q, err := c.FetchGSCReport(context.Background(), "token", "sc-domain:example.com", "appearance", "web", "2026-09-01", "2026-09-01")
	if err != nil || calls != 2 || len(rows) != 1 || rows[0].SearchAppearance != "AMP_BLUE_LINK" || rows[0].Slice != "appearance" || rows[0].Day.Format("2006-01-02") != "2026-09-01" || !q.Known {
		t.Fatalf("%#v %#v %v", rows, q, err)
	}
}
func TestTrafficQuotaSharedAcrossServicesAndFamilies(t *testing.T) {
	db := testDB(t)
	a, b := New(db), New(db)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	b.Now = a.Now
	ca := a.trafficQuotaContext(context.Background(), "sc-domain:example.com", "123")
	cb := b.trafficQuotaContext(context.Background(), "sc-domain:example.com", "123")
	if err := reserveTraffic(ca, "gsc"); err != nil {
		t.Fatal(err)
	}
	backoffTraffic(ca, "gsc", "3600")
	if got := b.trafficRetryAt(context.Background(), "sc-domain:example.com", "123", errors.New("gsc HTTP 429 quota")); got != now.Add(time.Hour).Unix() {
		t.Fatalf("retry %d", got)
	}
	if got := b.trafficRetryAt(context.Background(), "sc-domain:example.com", "123", errors.Join(errors.New("gsc HTTP 429 quota"), errors.New("ga HTTP 401 unauthorized"))); got != 0 {
		t.Fatal("auth failure hidden by quota")
	}

	if class, _ := classifyErr(reserveTraffic(cb, "gsc")); class != "rate_limited" {
		t.Fatal("shared backoff ignored")
	}
	if err := reserveTraffic(cb, "ga"); err != nil {
		t.Fatal("other API blocked", err)
	}
	other := b.trafficQuotaContext(context.Background(), "sc-domain:other.com", "")
	if err := reserveTraffic(other, "gsc"); err != nil {
		t.Fatal("other property blocked", err)
	}
	now = now.Add(time.Hour + time.Second)
	if err := reserveTraffic(cb, "gsc"); err != nil {
		t.Fatal("backoff did not expire", err)
	}
}
func TestHistoryStartGapsAndLastSuccessSurviveFailure(t *testing.T) {
	db := testDB(t)
	s := New(db)
	ctx := context.Background()
	p, err := s.projects.Create(ctx, project.CreateInput{Name: "History", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	start := "2026-09-20"
	if _, err = s.projects.Update(ctx, p.Slug, project.UpdateInput{GoogleHistoryStart: &start}); err != nil {
		t.Fatal(err)
	}
	through, _ := time.Parse("2006-01-02", "2026-09-22")
	from := through.AddDate(0, -16, 0)
	fetch := func(_ context.Context, part dateChunk) (repo.SyncBatch, error) {
		if part.start.Before(through.AddDate(0, 0, -2)) {
			t.Fatal("history start ignored")
		}
		return repo.SyncBatch{}, nil
	}
	if err = s.syncReport(ctx, p.ID, "gsc", "sc-domain:a.com", "query", "web", from, through, 1, 1, fetch); err != nil {
		t.Fatal(err)
	}
	rows, err := s.syncProgress(ctx, p.ID, "gsc", "sc-domain:a.com")
	if err != nil || len(rows) != 1 || rows[0].TotalDays != 3 || rows[0].CoveredDays != 1 || len(rows[0].Gaps) != 1 || rows[0].Gaps[0].Through != "2026-09-21" || rows[0].LastSuccessAt == 0 {
		t.Fatalf("%#v %v", rows, err)
	}
	success := rows[0].LastSuccessAt
	_ = s.syncReport(ctx, p.ID, "gsc", "sc-domain:a.com", "query", "web", from, through, 1, 1, func(context.Context, dateChunk) (repo.SyncBatch, error) {
		return repo.SyncBatch{}, errors.New("network")
	})
	rows, _ = s.syncProgress(ctx, p.ID, "gsc", "sc-domain:a.com")
	if rows[0].LastSuccessAt != success || rows[0].State != "failed" || rows[0].CoveredDays != 1 {
		t.Fatalf("failure erased success %#v", rows)
	}
}
func TestOfficialQualityUnknownAndAggregationMismatch(t *testing.T) {
	for _, body := range []string{`{"rows":[]}`, `{"responseAggregationType":"byPage","rows":[]}`, `{"responseAggregationType":"byProperty","rows":[]}`} {
		c := &Client{HTTP: &http.Client{Transport: roundTrip(func(*http.Request) (int, string) { return 200, body })}}
		_, q, _, _, _, err := c.fetchGSCDateQuality(context.Background(), "t", "sc-domain:example.com", "2026-09-01", "2026-09-01")
		if body == `{"rows":[]}` && q.Known {
			t.Fatal("missing metadata promoted")
		}
		if body == `{"responseAggregationType":"byPage","rows":[]}` && err == nil {
			t.Fatal("mismatch accepted")
		}
		if body == `{"responseAggregationType":"byProperty","rows":[]}` && (!q.Known || err != nil) {
			t.Fatalf("%#v %v", q, err)
		}
	}
}
