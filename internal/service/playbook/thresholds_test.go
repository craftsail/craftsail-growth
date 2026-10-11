// SPDX-License-Identifier: AGPL-3.0-or-later

package playbook

import (
	"os"
	"regexp"
	"strconv"
	"testing"
)

// The help center prints the thresholds from the TS catalog; the judges
// use the Go constants. They must be the same numbers.
func TestThresholdsMatchCatalog(t *testing.T) {
	src, err := os.ReadFile("../../../web/src/features/playbook/catalog.ts")
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)export const THRESHOLDS = \{(.*?)\} as const`).FindSubmatch(src)
	if block == nil {
		t.Fatal("THRESHOLDS not found in catalog.ts")
	}
	ts := map[string]int{}
	for _, m := range regexp.MustCompile(`(\w+):\s*(\d+)`).FindAllSubmatch(block[1], -1) {
		v, _ := strconv.Atoi(string(m[2]))
		ts[string(m[1])] = v
	}
	want := map[string]int{
		"newPageDays": newPageDays, "imprCoverage": imprCoverage, "visibleShare": visibleShare, "weeks": weeks,
		"recognitionN": recognitionN, "recognitionLower": recognitionLower, "seoPassSignals": seoPassSignals,
	}
	for k, v := range want {
		if ts[k] != v {
			t.Errorf("%s: catalog %d, Go %d", k, ts[k], v)
		}
	}
}
