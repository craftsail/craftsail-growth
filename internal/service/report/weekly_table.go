// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"fmt"
	"math"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

// alertMinBase is the last-week value under which an alert is never
// evaluated: small bases swing too much week over week to mean anything.
const alertMinBase = 20

// countCell renders a raw count: "—" when the week was not fully covered,
// never 0.
func countCell(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f", *v)
}

// ratio divides num/den, guarding the nil and zero-denominator cases that
// would otherwise render a misleading 0%.
func ratio(num, den *float64) *float64 {
	if num == nil || den == nil || *den == 0 {
		return nil
	}
	v := *num / *den
	return &v
}

func pctCell(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", *v*100)
}

func ratioCell(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", *v)
}

// changeCell is the week-over-week percent change of two already-computed
// cells (counts or ratios), using a real minus sign so it never reads as a
// hyphenated range.
func changeCell(this, last *float64) string {
	if this == nil || last == nil || *last == 0 {
		return "—"
	}
	pct := math.Round((*this - *last) / *last * 100)
	if pct == 0 {
		return "0%"
	}
	if pct < 0 {
		return fmt.Sprintf("−%.0f%%", -pct)
	}
	return fmt.Sprintf("+%.0f%%", pct)
}

// ppCell is the Change column for ratio rows shown as percentages (CTR,
// non-brand share, engagement rate): the plain difference in percentage
// points, since a relative percent change of a percentage is misleading.
func ppCell(this, last *float64, t func(string, ...string) string) string {
	if this == nil || last == nil {
		return "—"
	}
	diff := math.Round((*this-*last)*1000) / 10
	if diff == 0 {
		return t("pp", "v", "0.0")
	}
	sign := "+"
	if diff < 0 {
		sign = "−"
		diff = -diff
	}
	return t("pp", "v", sign+fmt.Sprintf("%.1f", diff))
}

// writeWeeklyTable renders the book's weekly table for the project's SEO
// stage: the new-site table (three week columns, a current index snapshot)
// or the growth table (this/last week, change, and an alert line). A
// project with no site (seoStage == "") gets no table at all.
func writeWeeklyTable(b *strings.Builder, w *webstats.WeeklySearch, seoStage, lang string) {
	if seoStage == "" {
		return
	}
	t := func(k string, vars ...string) string { return reportText(lang, "weeklyReport."+k, vars...) }
	line := func(value string) { fmt.Fprintf(b, "- %s\n", value) }
	// colHeader appends the week's own dates to a column header, e.g.
	// "This week (2026-09-28 – 2026-10-04)"; a week with no dates (the
	// zero value in tests) keeps the bare header.
	colHeader := func(key string, wk webstats.Week) string {
		if wk.From == "" {
			return t(key)
		}
		return t("colRange", "col", t(key), "from", wk.From, "through", wk.Through)
	}

	if seoStage != "seoGrow" {
		fmt.Fprintf(b, "\n## %s\n\n", t("tableNew"))
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n|---|---:|---:|---:|\n", t("colMetric"), colHeader("colThis", w.Weeks[0]), colHeader("colLast", w.Weeks[1]), colHeader("colFour", w.Weeks[2]))
		row := func(metric string, cells ...string) {
			fmt.Fprintf(b, "| %s | %s |\n", metric, strings.Join(cells, " | "))
		}
		indexedCell := "—" // nothing inspected yet is unknown, not "0 of 0"
		if w.Inspected > 0 {
			indexedCell = t("rowIndexedValue", "indexed", fmt.Sprint(w.Indexed), "inspected", fmt.Sprint(w.Inspected))
		}
		row(t("rowIndexed"), indexedCell, "—", "—")
		sevenDay := "—"
		if w.PublishedMature != 0 {
			sevenDay = fmt.Sprintf("%d / %d", w.IndexedWithinWeek, w.PublishedMature)
		}
		row(t("rowSevenDay"), sevenDay, "—", "—")
		row(t("rowNewPages"), countCell(w.Weeks[0].NewPages), countCell(w.Weeks[1].NewPages), countCell(w.Weeks[2].NewPages))
		row(t("rowPagesImpr"), countCell(w.Weeks[0].PagesWithImpressions), countCell(w.Weeks[1].PagesWithImpressions), countCell(w.Weeks[2].PagesWithImpressions))
		row(t("rowImpressions"), countCell(w.Weeks[0].Impressions), countCell(w.Weeks[1].Impressions), countCell(w.Weeks[2].Impressions))
		row(t("rowClicks"), countCell(w.Weeks[0].Clicks), countCell(w.Weeks[1].Clicks), countCell(w.Weeks[2].Clicks))
		row(t("rowOrganic"), countCell(w.Weeks[0].OrganicSessions), countCell(w.Weeks[1].OrganicSessions), countCell(w.Weeks[2].OrganicSessions))
		row(t("rowKeyEvents"), countCell(w.Weeks[0].OrganicKeyEvents), countCell(w.Weeks[1].OrganicKeyEvents), countCell(w.Weeks[2].OrganicKeyEvents))
		b.WriteString("\n")
		line(t("snapshot"))
		line(t("smallNumbers"))
		line(t("tableSourceNew"))
	} else {
		fmt.Fprintf(b, "\n## %s\n\n", t("tableGrow"))
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n|---|---:|---:|---:|---|\n", t("colMetric"), colHeader("colThis", w.Weeks[0]), colHeader("colLast", w.Weeks[1]), t("colChange"), t("colAlert"))
		row := func(metric string, cells ...string) {
			fmt.Fprintf(b, "| %s | %s |\n", metric, strings.Join(cells, " | "))
		}
		// alertCell shows "—" below alertMinBase, where swings are noise, and
		// whenever the rule did not fire; it names the rule only when hit.
		alertCell := func(this, last *float64) string {
			if this == nil || last == nil || *last < alertMinBase {
				return "—"
			}
			pct := (*this - *last) / *last * 100
			if pct > -20 {
				return "—"
			}
			return t("alertWeekDrop") + " · " + t("alertHit")
		}

		nbClicks0, nbClicks1 := w.Weeks[0].NonBrandClicks, w.Weeks[1].NonBrandClicks
		row(t("rowNbClicks"), countCell(nbClicks0), countCell(nbClicks1), changeCell(nbClicks0, nbClicks1), alertCell(nbClicks0, nbClicks1))

		nbImpr0, nbImpr1 := w.Weeks[0].NonBrandImpressions, w.Weeks[1].NonBrandImpressions
		row(t("rowNbImpr"), countCell(nbImpr0), countCell(nbImpr1), changeCell(nbImpr0, nbImpr1), "—")

		ctr0, ctr1 := ratio(nbClicks0, nbImpr0), ratio(nbClicks1, nbImpr1)
		row(t("rowNbCtr"), pctCell(ctr0), pctCell(ctr1), ppCell(ctr0, ctr1, t), "—")

		share0, share1 := ratio(nbClicks0, w.Weeks[0].VisibleClicks), ratio(nbClicks1, w.Weeks[1].VisibleClicks)
		row(t("rowNbShare"), pctCell(share0), pctCell(share1), ppCell(share0, share1, t), "—")

		organic0, organic1 := w.Weeks[0].OrganicSessions, w.Weeks[1].OrganicSessions
		row(t("rowOrganic"), countCell(organic0), countCell(organic1), changeCell(organic0, organic1), alertCell(organic0, organic1))

		cts0, cts1 := ratio(organic0, w.Weeks[0].Clicks), ratio(organic1, w.Weeks[1].Clicks)
		row(t("rowClickToSession"), ratioCell(cts0), ratioCell(cts1), changeCell(cts0, cts1), "—")

		eng0, eng1 := ratio(w.Weeks[0].OrganicEngaged, organic0), ratio(w.Weeks[1].OrganicEngaged, organic1)
		row(t("rowEngagement"), pctCell(eng0), pctCell(eng1), ppCell(eng0, eng1, t), "—")

		eps0, eps1 := ratio(w.Weeks[0].OrganicKeyEvents, organic0), ratio(w.Weeks[1].OrganicKeyEvents, organic1)
		row(t("rowEventsPerSession"), ratioCell(eps0), ratioCell(eps1), changeCell(eps0, eps1), "—")

		ai0, ai1 := w.Weeks[0].AISessions, w.Weeks[1].AISessions
		row(t("rowAISessions"), countCell(ai0), countCell(ai1), changeCell(ai0, ai1), "—")

		b.WriteString("\n")
		line(t("tableSourceGrow"))
	}
	line(t("dayNote"))
	line(t("notTracked"))
}
