// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
	"github.com/craftsail/craftsail-growth/internal/service/sample"
	"github.com/craftsail/craftsail-growth/internal/service/webstats"
)

//go:embed catalog_generated.json
var catalogJSON []byte
var catalogs = func() map[string]map[string]any {
	var v map[string]map[string]any
	if err := json.Unmarshal(catalogJSON, &v); err != nil {
		panic(err)
	}
	return v
}()

func reportText(lang, key string, vars ...string) string {
	var v any = catalogs[lang]
	for _, part := range strings.Split(key, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return key
		}
		v = m[part]
	}
	text, ok := v.(string)
	if !ok {
		return key
	}
	for i := 0; i+1 < len(vars); i += 2 {
		text = strings.ReplaceAll(text, "{"+vars[i]+"}", vars[i+1])
	}
	return text
}
func markdownText(v string) string {
	return strings.NewReplacer("\\", "\\\\", "|", "\\|", "\n", " ", "\r", " ", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;").Replace(v)
}
func reportItemTitle(it opportunity.Item, lang string) string {
	t := func(k string, v ...string) string { return reportText(lang, k, v...) }
	if it.Source == "audit" && lang != "en" {
		if value := t("rules." + it.Kind + ".title"); !strings.HasPrefix(value, "rules.") {
			return value
		}
	}
	if it.Source == "search" {
		q, _ := it.Detail["query"].(string)
		if q == "" && len(it.URLs) > 0 {
			q = it.URLs[0]
		}
		if v := t("plan.titles."+it.Kind, "q", q); !strings.HasPrefix(v, "plan.") {
			return v
		}
	}
	if it.Source == "metric" && it.Kind == "visibility_down" {
		return t("plan.titles.visibility_down")
	}
	if it.Source == "citation" {
		prompt, _ := it.Detail["prompt"].(string)
		keys := map[string]string{"social": "citation_social", "outreach": "citation_outreach", "existing-content": "citation_refresh", "creation": "citation_creation"}
		if k := keys[it.Kind]; k != "" && prompt != "" {
			return t("plan.titles."+k, "q", prompt)
		}
	}
	return it.Title
}
func (s *Service) weeklyMarkdown(ctx context.Context, p *model.Project, lang string) (string, [][2]string, error) {
	t := func(key string, vars ...string) string { return reportText(lang, "weeklyReport."+key, vars...) }
	var b strings.Builder
	cards := [][2]string{}
	now := time.Now()
	heading := func(key string) { fmt.Fprintf(&b, "\n## %s\n\n", t(key)) }
	line := func(value string) { fmt.Fprintf(&b, "- %s\n", value) }
	fmt.Fprintf(&b, "# %s · %s · %s\n\n%s\n", markdownText(p.Name), t("title"), now.Format("2006-01-02"), t("weekly"))
	if p.DemoScenario != "" {
		fmt.Fprintf(&b, "\n%s\n", reportText(lang, "demoGuide.banner"))
	}
	status, statusErr := s.playbook.Status(ctx, p.Slug)
	web := webstats.New(s.db)
	board, boardErr := web.SearchBoard(ctx, p.Slug)
	var seoStage string
	if statusErr == nil {
		seoStage = status.SeoStage
		seo := t("noSite")
		if seoStage != "" {
			seo = t(seoStage)
		}
		product := t("stageUnset")
		if status.ProductStage != "" {
			product = strings.ToUpper(status.ProductStage)
		}
		target := "/p/" + url.PathEscape(p.Slug) + "/help#start"
		link := fmt.Sprintf("[%s](%s)", reportText(lang, "nav.help"), target)
		line(t("stagesLine", "seo", seo, "ai", t(status.AiStage), "product", product, "link", link))
	} else if board != nil && board.Observation != nil && board.Observation.Mode == "established" {
		// playbook.Status failed; fall back to a coarse stage from the search
		// board alone so the weekly table still renders. The stage line
		// itself is skipped since the rest of status is unavailable.
		seoStage = "seoGrow"
	} else {
		seoStage = "seoNew"
	}
	heading("health")
	coverage := func(source string, c webstats.GrainCoverage) {
		line(t("coverage", "source", source, "from", c.From, "through", c.Through, "covered", fmt.Sprint(c.CoveredDays), "total", fmt.Sprint(c.TotalDays)))
	}
	if boardErr != nil {
		line(t("unavailable"))
	} else if board != nil && board.Observation != nil {
		coverage("GSC", board.Observation.Coverage)
		coverage("GSC page", board.PageCoverage)
		coverage("GSC query", board.QueryCoverage)
	}
	ga, gaErr := web.ExploreGA(ctx, p.Slug, "channel", webstats.ExploreInput{PageSize: 10})
	if gaErr != nil {
		line("GA4: " + t("unavailable"))
	} else if ga != nil {
		coverage("GA4", ga.Coverage)
		yes := "no"
		if ga.Comparable {
			yes = "yes"
		}
		line(t("quality", "value", t(yes)))
	}
	fmt.Fprintf(&b, "\n### %s\n\n", t("imports"))
	line(t("importSnapshot"))
	wr := &repo.Webstats{DB: s.db}
	for _, source := range []string{"gsc", "ga4"} {
		imp, e := wr.GetImport(ctx, p.ID, source)
		if e != nil {
			return "", nil, e
		}
		if imp == nil || imp.Property == "" {
			continue
		}
		win, e := wr.LatestWindow(ctx, p.ID, source, imp.Property)
		if e != nil {
			return "", nil, e
		}
		if win == nil {
			continue
		}
		through := "—"
		if imp.FinalizedThrough != nil {
			through = imp.FinalizedThrough.Format("2006-01-02")
		}
		calendar := "pacific"
		if source == "ga4" {
			calendar = "propertyTime"
		}
		line(t("finalized", "source", source, "property", markdownText(imp.Property), "through", through, "calendar", t(calendar)))
		if source == "gsc" {
			line(t("daily", "impressions", insightNum(win.Impressions), "clicks", insightNum(win.Clicks)))
		}
	}
	line(t("official"))
	if !p.NoSite {
		if weekly, err := web.WeeklySearch(ctx, p, time.Time{}); err != nil {
			line(t("unavailable"))
		} else {
			writeWeeklyTable(&b, weekly, seoStage, lang)
		}
	}
	observations, err := (&repo.Observations{DB: s.db}).List(ctx, p.ID, "")
	if err != nil {
		return "", nil, err
	}
	heading("shipped")
	shippedSince := now.AddDate(0, 0, -7).Unix()
	written := false
	shipped := map[uint64]bool{}
	for _, o := range observations {
		if shipped[o.TaskID] {
			continue
		}
		shipped[o.TaskID] = true
		if o.ReleasedAt < shippedSince {
			continue
		}
		line(markdownText(o.TaskCode + " · " + o.Hypothesis))
		written = true
	}
	if !written {
		line(t("none"))
	}
	heading("google")
	if insight, e := web.Insight(ctx, p); e != nil {
		line(t("unavailable"))
	} else if insight != nil {
		line(t("window", "from", insightStr(insight["from"]), "through", insightStr(insight["to"]), "timezone", t("propertyTime")))
		line(t("gscTotals", "impressions", insightNum(insight["gsc_impressions"]), "clicks", insightNum(insight["gsc_clicks"])))
		line(t("gaTotals", "sessions", insightNum(insight["ga_sessions"]), "events", insightNum(insight["ga_key_events"]), "ai", insightNum(insight["ai_sessions"])))
	}
	line(t("official"))

	heading("ai")
	line(t("aiWindow"))
	haveAI := false
	var aiChanges []string
	for _, access := range []string{"api", "web"} {
		cur, e := s.sample.Measure(ctx, p.Slug, sample.MeasureQuery{Range: "30d", Access: access})
		if e != nil {
			line(t("unavailable"))
			continue
		}
		if cur == nil || cur.Access != access || cur.VisibilityN+cur.RecognitionN == 0 {
			continue
		}
		haveAI = true
		if cur.MixedVersions {
			line(reportText(lang, "sampling.mixed"))
		}
		fmt.Fprintf(&b, "\n### %s\n\n", strings.ToUpper(access))
		if cur.Visibility != nil && cur.VisibilityCI != nil {
			text := t("visibility", "value", fmt.Sprintf("%.0f", *cur.Visibility), "x", fmt.Sprint(cur.VisibilityX), "n", fmt.Sprint(cur.VisibilityN), "lo", fmt.Sprintf("%.0f", cur.VisibilityCI.Lo), "hi", fmt.Sprintf("%.0f", cur.VisibilityCI.Hi))
			if cur.LowSample {
				text += " " + t("small")
			}
			line(text)
			cards = append(cards, [2]string{t("ai") + " (" + access + ")", fmt.Sprintf("%.0f%%", *cur.Visibility)})
		}
		if prev, e := s.sample.MeasureWindow(ctx, p.Slug, access, 60, 30); e == nil && prev != nil && prev.VisibilityN > 0 && cur.VisibilityN > 0 {
			aiChanges = append(aiChanges, t("aiChange", "access", access, "beforeX", fmt.Sprint(prev.VisibilityX), "beforeN", fmt.Sprint(prev.VisibilityN), "afterX", fmt.Sprint(cur.VisibilityX), "afterN", fmt.Sprint(cur.VisibilityN)))
		}
		if cur.Recognition != nil {
			line(t("recognition", "value", fmt.Sprintf("%.0f", *cur.Recognition), "n", fmt.Sprint(cur.RecognitionN)))
		}
	}
	if !haveAI {
		line(t("none"))
	}
	heading("engines")
	if cur, e := s.sample.Measure(ctx, p.Slug, sample.MeasureQuery{Range: "30d"}); e == nil && cur != nil && len(cur.Engines) > 0 {
		for _, engine := range cur.Engines {
			for _, access := range []string{"api", "web"} {
				v, e := s.sample.Measure(ctx, p.Slug, sample.MeasureQuery{Range: "30d", Platform: engine.ID, Access: access})
				if e != nil {
					return "", nil, e
				}
				if v != nil && v.Access == access && v.VisibilityN > 0 {
					line(fmt.Sprintf("%s (%s): %d/%d", markdownText(engine.Label), access, v.VisibilityX, v.VisibilityN))
				}
			}
		}
	} else {
		line(t("none"))
	}
	heading("changes")
	for _, change := range aiChanges {
		line(change)
	}
	if board != nil && board.Period.Comparable && board.Period.ClicksDelta != nil {
		line(t("delta", "before", fmt.Sprintf("%.0f", board.Period.PreviousClicks), "after", fmt.Sprintf("%.0f", board.Period.Clicks), "delta", fmt.Sprintf("%.1f", *board.Period.ClicksDelta)))
	} else {
		line(t("none"))
	}
	if ga != nil && ga.Coverage.CoveredDays > 0 {
		fmt.Fprintf(&b, "\n### GA4\n\n%s\n\n| %s | %s | %s | %s |\n|---|---:|---:|---:|\n", t("window", "from", ga.Coverage.From, "through", ga.Coverage.Through, "timezone", ga.Timezone), t("channel"), t("sessions"), t("events"), t("perSession"))
		for _, r := range ga.Items {
			if r.CurrentRows == 0 {
				continue
			}
			ratio := "—"
			if r.Sessions > 0 {
				ratio = fmt.Sprintf("%.2f", r.KeyEvents/r.Sessions)
			}
			fmt.Fprintf(&b, "| %s / %s / %s | %.0f | %.0f | %s |\n", markdownText(r.Channel), markdownText(r.Source), markdownText(r.Medium), r.Sessions, r.KeyEvents, ratio)
		}
	}
	items, err := s.opportunity.List(ctx, p.Slug, opportunity.ListFilter{})
	if err != nil {
		return "", nil, err
	}
	titles := map[string]string{}
	for _, it := range items {
		if it.TaskCode != "" {
			titles[it.TaskCode] = reportItemTitle(it, lang)
		}
	}
	heading("actions")
	tasks, err := s.tasks.List(ctx, p.ID)
	if err != nil {
		return "", nil, err
	}
	written = false
	for _, task := range tasks {
		if task.UpdatedAt < now.AddDate(0, 0, -7).Unix() {
			continue
		}
		title := titles[task.Code]
		if title == "" {
			title = task.Title
		}
		line(markdownText(task.Code+" · "+title) + " · " + t(task.Status))
		written = true
	}
	if !written {
		line(t("none"))
	}
	heading("observations")
	written = false
	latest := map[uint64]bool{}
	for _, o := range observations {
		if latest[o.TaskID] {
			continue
		}
		latest[o.TaskID] = true
		if o.Metric != "technical" && o.FollowupThrough > now.Format("2006-01-02") {
			continue
		}
		state := "pending"
		if len(o.Results) > 0 {
			state = o.Results[0].Conclusion
		}
		line(markdownText(o.TaskCode+" · "+o.Hypothesis) + " · " + reportText(lang, "observation."+state) + " · " + t("due", "date", o.FollowupThrough))
		written = true
	}
	if !written {
		line(t("none"))
	}
	heading("next")
	written = false
	for _, it := range items {
		if !it.Recommended {
			continue
		}
		target := "/p/" + url.PathEscape(p.Slug) + "/opportunities?key=" + url.QueryEscape(it.Key)
		if it.Status != "" {
			target += "&tab=progress"
		}
		line(fmt.Sprintf("%s · [%s](%s)", it.Priority, markdownText(reportItemTitle(it, lang)), target))
		written = true
	}
	if !written {
		line(t("none"))
	}
	heading("rulesTitle")
	line(t("rules"))
	return b.String(), cards, nil
}
