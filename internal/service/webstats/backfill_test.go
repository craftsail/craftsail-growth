// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"testing"
	"time"
)

func TestSyncChunks(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	all := chunks("gsc", time.Time{}, now)
	if len(all) == 0 || all[0].start.Format("2006-01-02") != "2025-05-22" || all[0].end.Format("2006-01-02") != "2025-05-28" {
		t.Fatalf("first %#v", all)
	}
	if all[len(all)-1].end.After(time.Date(2026, 9, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("last %#v", all[len(all)-1])
	}
	inc := chunks("gsc", time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC), now)
	if len(inc) != 1 || inc[0].start.Format("2006-01-02") != "2026-09-10" || inc[0].end.Format("2006-01-02") != "2026-09-19" {
		t.Fatalf("inc %#v", inc)
	}
}
