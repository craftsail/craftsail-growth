// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import "testing"

func TestCompareCitedCoversSubdomain(t *testing.T) {
	b := CompareCited(map[string]int{"www.maigoo.com": 3, "zhihu.com": 1})
	if len(b.Covered) == 0 {
		t.Fatal("maigoo should be covered")
	}
	found := false
	for _, c := range b.Covered {
		if c.Domain == "maigoo.com" && c.Yours == 3 {
			found = true
		}
	}
	if !found {
		t.Fatalf("%+v", b.Covered)
	}
	if b.CoverageRate <= 0 {
		t.Fatal("rate")
	}
}
