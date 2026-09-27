// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import "testing"

func TestCrawlViewCountsOrphansAndCritical(t *testing.T) {
	pages := []CrawlPage{
		{URL: "https://acme.com/", Status: 200, Score: 87, Words: 1028, H1: 1, ImagesMissingAlt: 0, InternalLinks: 12, Millis: 664},
		{URL: "https://acme.com/old", Status: 404, Score: 0, Words: 0, H1: 0, InternalLinks: 0, Millis: 65},
	}
	view := CrawlView(pages, 72)
	if view.Pages != 2 || view.Critical != 1 || view.Health != 72 {
		t.Fatalf("%+v", view)
	}
	if view.Orphans != 1 {
		t.Fatalf("orphans %d", view.Orphans)
	}
	if view.ContentAvg != 44 {
		t.Fatalf("avg %d", view.ContentAvg)
	}
}
