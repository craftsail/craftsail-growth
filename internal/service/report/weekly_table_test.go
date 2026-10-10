// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

func TestWeeklyTableLayouts(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	w := &webstats.WeeklySearch{Indexed: 12, Inspected: 20, PublishedMature: 4, IndexedWithinWeek: 3}
	w.Weeks[0] = webstats.Week{From: "2026-09-28", Through: "2026-10-04", Clicks: f(70), Impressions: f(700), PagesWithImpressions: f(9), NonBrandClicks: f(14), NonBrandImpressions: f(140), VisibleClicks: f(35), OrganicSessions: f(50), OrganicEngaged: f(25), OrganicKeyEvents: f(5)}
	w.Weeks[1] = webstats.Week{From: "2026-09-21", Through: "2026-09-27", Clicks: f(60), Impressions: f(600), NonBrandClicks: f(20), NonBrandImpressions: f(150), VisibleClicks: f(30), OrganicSessions: f(70)}
	w.Weeks[2] = webstats.Week{From: "2026-08-31", Through: "2026-09-06"}
	var b strings.Builder
	writeWeeklyTable(&b, w, "seoNew", "en")
	out := b.String()
	for _, want := range []string{
		"Weekly table: new site", "| Indexed URLs (inspected) | 12 of 20 |", "| Impressions | 700 | 600 | — |", "3 / 4", "Pacific",
		"This week (2026-09-28 – 2026-10-04)", "Last week (2026-09-21 – 2026-09-27)", "4 weeks ago (2026-08-31 – 2026-09-06)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("new-site table missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "{") {
		t.Fatalf("unfilled placeholder:\n%s", out)
	}
	b.Reset()
	writeWeeklyTable(&b, w, "seoGrow", "en")
	out = b.String()
	for _, want := range []string{
		"Weekly table: growth", "This week (2026-09-28 – 2026-10-04)", "Last week (2026-09-21 – 2026-09-27)",
		"| Non-brand clicks (visible queries) | 14 | 20 | −30% | week over week below −20% · below the alert line |",
		"| Non-brand CTR | 10.0% | 13.3% | −3.3 pp | — |",
		"| Non-brand share of visible clicks | 40.0% | 66.7% | −26.7 pp | — |",
		"| Organic search sessions (GA4) | 50 | 70 | −29% | week over week below −20% · below the alert line |",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("growth table missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "{") {
		t.Fatalf("unfilled placeholder:\n%s", out)
	}
}

// TestWeeklyTableAlertMinBase checks that a week-over-week drop never fires
// the alert when last week's value is below alertMinBase: small bases swing
// too much to mean anything, even when the percentage looks dramatic.
func TestWeeklyTableAlertMinBase(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	w := &webstats.WeeklySearch{}
	// A 50% drop (10 -> 5) would hit the -20% rule, but last week's base of
	// 10 is below alertMinBase (20), so the alert must stay silent.
	w.Weeks[0] = webstats.Week{NonBrandClicks: f(5), NonBrandImpressions: f(50), VisibleClicks: f(5)}
	w.Weeks[1] = webstats.Week{NonBrandClicks: f(10), NonBrandImpressions: f(100), VisibleClicks: f(10)}
	var b strings.Builder
	writeWeeklyTable(&b, w, "seoGrow", "en")
	out := b.String()
	if strings.Contains(out, "below the alert line") {
		t.Fatalf("alert fired below alertMinBase:\n%s", out)
	}
	if !strings.Contains(out, "| Non-brand clicks (visible queries) | 5 | 10 | −50% | — |") {
		t.Fatalf("expected a silent alert cell below alertMinBase:\n%s", out)
	}

	// Above the base, a qualifying drop does fire.
	b.Reset()
	w.Weeks[0].NonBrandClicks, w.Weeks[1].NonBrandClicks = f(10), f(20)
	writeWeeklyTable(&b, w, "seoGrow", "en")
	out = b.String()
	if !strings.Contains(out, "| Non-brand clicks (visible queries) | 10 | 20 | −50% | week over week below −20% · below the alert line |") {
		t.Fatalf("expected the alert to fire at or above alertMinBase:\n%s", out)
	}
}

// TestChangeAndPPCellsAvoidNegativeZero checks that a rounded-to-zero
// change never prints as "−0%" or "−0.0 pp".
func TestChangeAndPPCellsAvoidNegativeZero(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	this, last := f(99.6), f(100) // -0.4%, rounds to 0
	if got := changeCell(this, last); got != "0%" {
		t.Fatalf("changeCell(%v, %v) = %q, want %q", *this, *last, got, "0%")
	}
	tr := func(k string, vars ...string) string { return reportText("en", "weeklyReport."+k, vars...) }
	this2, last2 := f(0.1002), f(0.1) // -0.02 pp, rounds to 0.0
	if got := ppCell(this2, last2, tr); got != "0.0 pp" {
		t.Fatalf("ppCell(%v, %v) = %q, want %q", *this2, *last2, got, "0.0 pp")
	}
}

func TestWeeklyTableNoSite(t *testing.T) {
	var b strings.Builder
	writeWeeklyTable(&b, &webstats.WeeklySearch{}, "", "en")
	if b.Len() != 0 {
		t.Fatalf("no-site project should render no table: %q", b.String())
	}
}
