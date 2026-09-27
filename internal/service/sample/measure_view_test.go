// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestMeasurePeriodRatioNotLastDay(t *testing.T) {
	day1 := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", Platform: "deepseek", SampledOn: day1, OK: true, Mentioned: true, QuestionText: "最好的跑鞋", SampleMode: "api"},
		{ID: 2, QID: "q1", Platform: "deepseek", SampledOn: day2, OK: true, Mentioned: false, QuestionText: "最好的跑鞋", SampleMode: "api"},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Now: day2})
	if view.Visibility == nil || math.Round(*view.Visibility) != 50 {
		t.Fatalf("visibility = %v, want 50", view.Visibility)
	}
	if len(view.VisibilitySeries) != 2 || view.VisibilitySeries[1].Value != 0 {
		t.Fatalf("series = %+v", view.VisibilitySeries)
	}
	if len(view.PromptCharts) != 1 || math.Round(view.PromptCharts[0].Visibility) != 50 {
		t.Fatalf("prompt = %+v", view.PromptCharts)
	}
}

func TestMeasureTagFilterMatchesElmo(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: true, Mentioned: false, QuestionText: "最好的工具"},
		{ID: 2, QID: "q2", SampledOn: day, OK: true, Mentioned: true, QuestionText: "Nike 怎么样"},
	}
	questions := []Question{{ID: "q1", Text: "最好的工具", Tags: []string{"price"}}, {ID: "q2", Text: "Nike 怎么样"}}
	all := buildMeasure(measureBuild{Brand: "Nike", Questions: questions, Rows: rows, Now: day})
	if all.Prompts != 2 {
		t.Fatalf("prompt list still shows both prompts, got %d", all.Prompts)
	}
	if all.VisibilityN != 1 {
		t.Fatalf("visibility must exclude the branded prompt, n=%d", all.VisibilityN)
	}
	if len(all.Tags) < 3 || all.Tags[0].ID != "branded" || all.Tags[1].ID != "unbranded" || all.Tags[2].ID != "price" {
		t.Fatalf("tags = %+v", all.Tags)
	}
	branded := buildMeasure(measureBuild{Brand: "Nike", Questions: questions, Rows: rows, Query: MeasureQuery{Tag: "branded"}, Now: day})
	if branded.Prompts != 1 || branded.PromptCharts[0].ID != "q2" {
		t.Fatalf("branded = %+v", branded.PromptCharts)
	}
	price := buildMeasure(measureBuild{Brand: "Nike", Questions: questions, Rows: rows, Query: MeasureQuery{Tag: "price"}, Now: day})
	if price.Prompts != 1 || price.PromptCharts[0].ID != "q1" {
		t.Fatalf("price = %+v", price.PromptCharts)
	}
}

func TestMeasureFanoutAddsUpAndSkipsUnavailable(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: true, QuestionText: "最好的跑鞋", WebQueries: []string{WebQueriesUnavailable}},
		{ID: 2, QID: "q1", SampledOn: day, OK: true, QuestionText: "最好的跑鞋", WebQueries: []string{"best running shoes 2026", "best running shoes 2026"}},
		{ID: 3, QID: "q1", SampledOn: day, OK: true, QuestionText: "最好的跑鞋"},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Now: day})
	if view.FanoutTotal != 3 || view.FanoutUnknown != 2 || view.FanoutKnown != 1 {
		t.Fatalf("fanout total=%d unknown=%d known=%d", view.FanoutTotal, view.FanoutUnknown, view.FanoutKnown)
	}
	if view.FanoutUnknown+view.FanoutKnown != view.FanoutTotal {
		t.Fatal("unknown + known != total")
	}
	if view.FanoutAvg != 1 {
		t.Fatalf("avg = %v", view.FanoutAvg)
	}
	for _, w := range view.Words {
		if w.Word == WebQueriesUnavailable {
			t.Fatalf("cloud contains unavailable: %+v", view.Words)
		}
	}
	found := false
	for _, w := range view.Added {
		if w.Word == "2026" || w.Word == "best" || w.Word == "running" || w.Word == "shoes" {
			found = true
		}
		if strings.Contains(w.Word, "最好的跑鞋") {
			t.Fatalf("added kept the whole question: %+v", view.Added)
		}
	}
	if !found {
		t.Fatalf("added = %+v, want a token absent from the question", view.Added)
	}
}

func TestMeasureShareMatchesLeaderboard(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: true, Mentioned: true, CompetitorsMentioned: []string{"Adidas"}, QuestionText: "跑鞋品牌"},
		{ID: 2, QID: "q2", SampledOn: day, OK: true, CompetitorsMentioned: []string{"Adidas"}, QuestionText: "另一题"},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Competitors: []string{"Adidas"}, Rows: rows, Now: day})
	if view.Share == nil {
		t.Fatal("missing share")
	}
	var sum float64
	var you *MeasureLeader
	for i := range view.Leaders {
		sum += view.Leaders[i].Share
		if view.Leaders[i].IsBrand {
			you = &view.Leaders[i]
		}
	}
	if you == nil || you.Name != "Nike" || math.Abs(you.Share-*view.Share) > 0.001 {
		t.Fatalf("you = %+v share %v", you, view.Share)
	}
	if math.Abs(sum-100) > 0.001 {
		t.Fatalf("shares sum %v", sum)
	}
	if you.Prompts != 1 || view.Prompts != 2 {
		t.Fatalf("coverage prompts=%d you=%d", view.Prompts, you.Prompts)
	}
}

func TestMeasureOpportunityUsesSameVisibility(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: true, Mentioned: false, CompetitorsMentioned: []string{"Altra"}, QuestionText: "宽鞋头跑鞋"},
		{ID: 2, QID: "q1", SampledOn: day, OK: true, Mentioned: true, CompetitorsMentioned: []string{"Altra"}, QuestionText: "宽鞋头跑鞋"},
		{ID: 3, QID: "q1", SampledOn: day, OK: true, Mentioned: false, CompetitorsMentioned: []string{"Altra"}, QuestionText: "宽鞋头跑鞋"},
		{ID: 4, QID: "q1", SampledOn: day, OK: true, Mentioned: false, CompetitorsMentioned: []string{"Altra"}, QuestionText: "宽鞋头跑鞋"},
	}
	cites := []model.SampleCitation{
		{SampleID: 1, QID: "q1", URL: "https://altra.example/guide", Domain: "altra.example", Category: "competitor", SampledOn: day},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Cites: cites, Now: day})
	if len(view.Opportunities) != 1 {
		t.Fatalf("ops = %+v", view.Opportunities)
	}
	op := view.Opportunities[0]
	if len(op.Prompts) == 0 || len(op.Rivals) == 0 || op.Prompts[0].QID != "q1" {
		t.Fatalf("evidence = %+v", op)
	}
	if !strings.Contains(op.Why, "25%") || !strings.Contains(op.Why, "100%") || !strings.Contains(op.Why, "Altra") {
		t.Fatalf("why = %s", op.Why)
	}
	if math.Round(view.PromptCharts[0].Visibility) != 25 {
		t.Fatalf("card visibility = %v", view.PromptCharts[0].Visibility)
	}
}

func TestMeasureCitationShareFromRows(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{{ID: 1, QID: "q1", SampledOn: day, OK: true, QuestionText: "题"}}
	cites := []model.SampleCitation{
		{SampleID: 1, QID: "q1", URL: "https://nike.com/a", Domain: "nike.com", Category: "brand", SampledOn: day},
		{SampleID: 1, QID: "q1", URL: "https://zhihu.com/a", Domain: "zhihu.com", Category: "social", PageType: "forum", SampledOn: day},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Site: "https://nike.com", Rows: rows, Cites: cites, Now: day})
	if view.Citations != 2 || view.UniqueDomains != 2 || view.CitationShare == nil || *view.CitationShare != 50 {
		t.Fatalf("cites=%d domains=%d share=%v", view.Citations, view.UniqueDomains, view.CitationShare)
	}
	if len(view.CategorySeries) != 1 || len(view.PageTypeSeries) != 1 {
		t.Fatal("missing stack")
	}
	if len(view.Gaps) != 0 {
		t.Fatalf("own cite should not be a gap: %+v", view.Gaps)
	}
}

func TestMeasureEmptyChartReasonUsesToday(t *testing.T) {
	day := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: false, Error: "HTTP 401", QuestionText: "买家题"},
		{ID: 2, QID: "q1", SampledOn: day, OK: true, QuestionText: "买家题"},
	}
	view := buildMeasure(measureBuild{Brand: "acmecli", Rows: rows, Now: day})
	if view.Runs != 1 || view.Visibility == nil {
		t.Fatalf("runs=%d vis=%v", view.Runs, view.Visibility)
	}
	if strings.Contains(view.FailReason, "401") {
		t.Fatalf("successful runs should not keep the old key error as the chart reason: %s", view.FailReason)
	}
}

func TestMeasureFailedRunsStayOutOfVisibility(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: false, Error: "HTTP 401", QuestionText: "最好的工具"},
		{ID: 2, QID: "q1", SampledOn: day, OK: false, Error: "HTTP 401", QuestionText: "最好的工具"},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Now: day})
	if view.Runs != 0 || view.Visibility != nil || view.Failed != 2 || !strings.Contains(view.FailReason, "401") {
		t.Fatalf("runs=%d vis=%v failed=%d reason=%s", view.Runs, view.Visibility, view.Failed, view.FailReason)
	}
	if len(view.PromptCharts) != 1 || view.PromptCharts[0].Runs != 0 || view.PromptCharts[0].Failed != 2 {
		t.Fatalf("chart = %+v", view.PromptCharts)
	}
}

func TestMeasurePromptRunsStayTogether(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", Platform: "deepseek", SampledOn: day, OK: true, SampleMode: "api", Mentioned: true, CompetitorsMentioned: []string{"Adidas"}, WebQueries: []string{WebQueriesUnavailable}, Answer: "Nike", CreatedAt: 100, QuestionText: "跑鞋"},
		{ID: 2, QID: "q1", Platform: "chatgpt", SampledOn: day, OK: false, SampleMode: "manual", Error: "验证码", CreatedAt: 90, QuestionText: "跑鞋"},
	}
	view := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Query: MeasureQuery{QID: "q1"}, Now: day})
	if view.Prompt == nil || view.Prompt.Runs != 2 || len(view.RunsDetail) != 2 {
		t.Fatalf("prompt runs = %+v %d", view.Prompt, len(view.RunsDetail))
	}
	if view.RunsDetail[0].Access != "API" || !view.RunsDetail[0].Unknown || len(view.RunsDetail[0].Queries) != 0 {
		t.Fatalf("run = %+v", view.RunsDetail[0])
	}
	if !view.RunsDetail[0].Chips[0].Self || view.RunsDetail[0].Chips[1].Name != "Adidas" {
		t.Fatalf("chips = %+v", view.RunsDetail[0].Chips)
	}
	if view.RunsDetail[1].OK || view.RunsDetail[1].Access != "Manual" {
		t.Fatalf("failed = %+v", view.RunsDetail[1])
	}
	if view.Prompt.Next != "Done today 1/3" {
		t.Fatalf("next = %s", view.Prompt.Next)
	}
}

func TestMeasureSplitsAccessAndReportsInterval(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", Platform: "deepseek", SampledOn: day, OK: true, Mentioned: true, QuestionText: "最好的工具", SampleMode: "api"},
		{ID: 2, QID: "q1", Platform: "deepseek", SampledOn: day, OK: true, Mentioned: true, QuestionText: "最好的工具", SampleMode: "api", Round: 2},
		{ID: 3, QID: "q1", Platform: "doubao", SampledOn: day, OK: true, Mentioned: false, QuestionText: "最好的工具", SampleMode: "manual"},
	}
	v := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Now: day})
	if v.Access != "api" || v.VisibilityN != 2 || v.VisibilityX != 2 || v.Visibility == nil || *v.Visibility != 100 {
		t.Fatalf("api view access=%s n=%d vis=%v", v.Access, v.VisibilityN, v.Visibility)
	}
	if v.VisibilityCI == nil || v.VisibilityCI.Hi != 100 {
		t.Fatalf("ci should be on the 0-100 scale, got %+v", v.VisibilityCI)
	}
	if len(v.Accesses) != 2 {
		t.Fatalf("accesses = %v", v.Accesses)
	}
	if len(v.PromptCharts) != 1 || v.PromptCharts[0].X != 2 || v.PromptCharts[0].N != 2 {
		t.Fatalf("prompt x/n = %+v", v.PromptCharts)
	}
	w := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Now: day, Query: MeasureQuery{Access: "web"}})
	if w.VisibilityN != 1 || w.Visibility == nil || *w.Visibility != 0 {
		t.Fatalf("web view n=%d vis=%v", w.VisibilityN, w.Visibility)
	}
}

func TestMeasureShareIgnoresBrandedPrompts(t *testing.T) {
	day := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ID: 1, QID: "q1", SampledOn: day, OK: true, Mentioned: false, QuestionText: "最好的工具", CompetitorsMentioned: []string{"Adidas"}},
		{ID: 2, QID: "q2", SampledOn: day, OK: true, Mentioned: true, QuestionText: "Nike 怎么样"},
	}
	v := buildMeasure(measureBuild{Brand: "Nike", Rows: rows, Now: day})
	if v.Share == nil || *v.Share != 0 {
		t.Fatalf("a branded prompt naming the brand must not add share of voice, got %v", v.Share)
	}
	if v.Recognition == nil || *v.Recognition != 100 || v.RecognitionN != 1 {
		t.Fatalf("recognition = %v n=%d", v.Recognition, v.RecognitionN)
	}
}
