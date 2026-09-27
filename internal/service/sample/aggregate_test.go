// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import "testing"

func makeRow(platform, qid string, rnd int, mode string, question string, mentioned, probe bool) Row {
	rank := 0
	if mentioned {
		rank = 1
	}
	return Row{
		Platform: platform, QuestionID: qid, Round: rnd, SampleMode: mode,
		Question: question, Market: "cn", OK: true, BrandInQuestion: probe,
		Analysis: Analysis{BrandMentioned: mentioned, BrandRank: rank, AnswerChars: 10},
	}
}

func TestDedupSameDayKeepsLast(t *testing.T) {
	first := makeRow("deepseek", "Q1", 1, "api", "有什么好用的工具？", true, false)
	last := makeRow("deepseek", "Q1", 1, "api", "有什么好用的工具？", false, false)
	rows := DedupRows([]Row{first, last})
	if len(rows) != 1 || rows[0].Analysis.BrandMentioned {
		t.Fatalf("%+v", rows)
	}
	agg := Aggregate(DedupRows([]Row{first, last}), testCfg())
	m := agg["deepseek"]
	if m.Samples != 1 || m.MentionRate == nil || *m.MentionRate != 0 {
		t.Fatalf("%+v", m)
	}
}

func TestDedupKeyDistinguishesRoundAndMode(t *testing.T) {
	rows := []Row{
		makeRow("deepseek", "Q1", 1, "api", "q", true, false),
		makeRow("deepseek", "Q1", 2, "api", "q", true, false),
		makeRow("deepseek", "Q1", 1, "manual", "q", true, false),
	}
	if len(DedupRows(rows)) != 3 {
		t.Fatal(len(DedupRows(rows)))
	}
}

func TestProbeOnlyMentionRateNone(t *testing.T) {
	rows := []Row{
		makeRow("deepseek", "Q1", 1, "api", "AIGCLINK定制家是什么", true, true),
		makeRow("deepseek", "Q2", 1, "api", "AIGCLINK定制家官网是哪个", true, true),
	}
	m := Aggregate(rows, testCfg())["deepseek"]
	if m.MentionRate != nil {
		t.Fatalf("mention_rate should be nil, got %v", *m.MentionRate)
	}
	if m.Samples != 0 || m.Probe.Samples != 2 || m.Probe.RecognizedRate == nil || *m.Probe.RecognizedRate != 1 {
		t.Fatalf("%+v", m)
	}
}

func TestMixedPlatformStillSplits(t *testing.T) {
	rows := []Row{
		makeRow("deepseek", "Q1", 1, "api", "AIGCLINK定制家是什么", true, true),
		makeRow("deepseek", "Q2", 1, "api", "有什么好用的工具？", true, false),
		makeRow("deepseek", "Q3", 1, "api", "有什么好用的工具？", false, false),
	}
	m := Aggregate(rows, testCfg())["deepseek"]
	if m.Samples != 2 || m.MentionRate == nil || *m.MentionRate != 0.5 {
		t.Fatalf("%+v", m)
	}
	if m.Probe.Samples != 1 {
		t.Fatalf("probe %+v", m.Probe)
	}
}

func TestAggregateDoesNotMixSampleModes(t *testing.T) {
	rows := []Row{
		makeRow("deepseek", "Q1", 1, "api", "有什么好用的工具？", true, false),
		makeRow("deepseek", "Q2", 1, "manual", "有什么好用的工具？", false, false),
	}
	rows[0].Analysis.CompetitorsMentioned = []string{"竞品A"}
	m := Aggregate(rows, testCfg())["deepseek"]
	if m.MentionRate != nil || m.ShareOfVoice != nil {
		t.Fatalf("mixed modes must not publish one rate: %+v", m)
	}
	api := m.ByMode["api"]
	manual := m.ByMode["manual"]
	if api.MentionRate == nil || *api.MentionRate != 1 || api.ShareOfVoice == nil || *api.ShareOfVoice != 0.5 {
		t.Fatalf("api %+v", api)
	}
	if manual.MentionRate == nil || *manual.MentionRate != 0 || manual.ShareOfVoice != nil {
		t.Fatalf("manual %+v", manual)
	}
}

func TestNoSiteCiteRateNil(t *testing.T) {
	cfg := Cfg{BrandName: "商品", Site: "", Competitors: nil}
	rows := []Row{makeRow("deepseek", "q001", 1, "api", "有哪些好用的绿茶", false, false)}
	rows[0].Analysis.CitedDomains = []string{"x.com"}
	m := Aggregate(rows, cfg)["deepseek"]
	if m.OwnDomainCiteRate != nil {
		t.Fatal("no-site cite rate must be nil")
	}
	if m.MentionRate == nil || *m.MentionRate != 0 {
		t.Fatalf("mention %+v", m.MentionRate)
	}
}

func TestDedupKeepsDifferentDays(t *testing.T) {
	a := makeRow("deepseek", "Q1", 1, "api", "有什么好用的工具？", true, false)
	b := makeRow("deepseek", "Q1", 1, "api", "有什么好用的工具？", false, false)
	a.Day, b.Day = "2026-09-01", "2026-09-02"
	if got := len(DedupRows([]Row{a, b})); got != 2 {
		t.Fatalf("rows from different days must both count over a window, got %d", got)
	}
}

func TestAggregateCarriesInterval(t *testing.T) {
	rows := []Row{
		makeRow("deepseek", "Q1", 1, "api", "有什么好用的工具？", true, false),
		makeRow("deepseek", "Q2", 1, "api", "推荐一个工具", false, false),
	}
	st := Aggregate(rows, testCfg())["deepseek"]
	if st.MentionCI == nil || st.MentionN != 2 {
		t.Fatalf("mention ci=%v n=%d", st.MentionCI, st.MentionN)
	}
}
