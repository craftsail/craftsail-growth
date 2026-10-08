// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"testing"
)

func TestSharedQuotaResetBackoffAndRestart(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	const now int64 = 1800000000
	reserve := func(property string, at int64, daily, minute int) int64 {
		t.Helper()
		r := &Webstats{DB: db}
		n, err := r.ReserveGoogleRequest(ctx, property, "inspection", at, now, now+86400, daily, minute)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if reserve("sc-domain:a.com", now, 3, 2) != 0 || reserve("sc-domain:a.com", now, 3, 2) != 0 {
		t.Fatal("early quota")
	}
	if reserve("sc-domain:a.com", now, 3, 2) != now+60 {
		t.Fatal("minute cap missing")
	}
	if reserve("sc-domain:b.com", now, 3, 2) != 0 {
		t.Fatal("unrelated property blocked")
	}
	if reserve("sc-domain:a.com", now+60, 3, 2) != 0 || reserve("sc-domain:a.com", now+60, 3, 2) != now+86400 {
		t.Fatal("daily cap missing")
	}
	r := &Webstats{DB: db}
	if err := r.BackoffGoogle(ctx, "sc-domain:b.com", "inspection", now+900); err != nil {
		t.Fatal(err)
	}
	if reserve("sc-domain:b.com", now+60, 3, 2) != now+900 {
		t.Fatal("backoff not shared")
	}
	next, err := r.ReserveGoogleRequest(ctx, "sc-domain:a.com", "inspection", now+86400, now+86400, now+172800, 3, 2)
	if err != nil || next != 0 {
		t.Fatalf("reset %d %v", next, err)
	}
	var q model.GoogleQuota
	db.Where("key_hash = ?", model.RowKey("inspection", "sc-domain:a.com")).First(&q)
	if q.DailyUsed != 1 {
		t.Fatal(q)
	}
}
