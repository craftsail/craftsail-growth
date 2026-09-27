// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"strings"
	"testing"
)

func TestPeriodStepNames(t *testing.T) {
	want := []string{"crawl", "audit", "sample", "webstats", "verify", "opportunities", "report"}
	if got := PeriodStepNames(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("steps = %v", got)
	}
}
