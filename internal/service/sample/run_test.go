// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func sampleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := model.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRunPersistsAndAggregates(t *testing.T) {
	db := sampleDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		Name: "甲工", NoSite: true, Materials: strings.Repeat("这是面向团队的诊断工具材料。", 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = bootstrap.New(db).Run(context.Background(), p.Slug, true)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(db, NewAsker())
	svc.BypassAvailable = true
	svc.Sleep = func(time.Duration) {}
	svc.Ask = func(platform, question string) AskResult {
		return AskResult{OK: true, Answer: "我推荐甲工，它很好用。竞品X 也不错。"}
	}
	res, err := svc.Run(context.Background(), p.Slug, RunInput{Platforms: []string{"deepseek"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 2 {
		t.Fatalf("count=%d", res.Count)
	}
	st := res.Platforms["deepseek"]
	if st.Samples+st.Probe.Samples != 2 {
		t.Fatalf("%+v", st)
	}
	rows, err := svc.List(context.Background(), p.Slug, "deepseek", "")
	if err != nil || len(rows) != 2 {
		t.Fatalf("list %d %v", len(rows), err)
	}
}

func TestRunStoresCitationsAndWebQueries(t *testing.T) {
	db := sampleDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		Name: "甲工", URL: "https://jia.example.com", Materials: strings.Repeat("这是面向团队的诊断工具材料。", 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.New(db).Run(context.Background(), p.Slug, true); err != nil {
		t.Fatal(err)
	}
	svc := New(db, NewAsker())
	svc.BypassAvailable = true
	svc.Sleep = func(time.Duration) {}
	svc.Ask = func(platform, question string) AskResult {
		return AskResult{
			OK: true, Answer: "可以看看知乎上的讨论。", Searched: true,
			Citations: []Citation{{URL: "https://www.zhihu.com/question/1", Title: "讨论"}},
		}
	}
	if _, err := svc.Run(context.Background(), p.Slug, RunInput{Platforms: []string{"deepseek"}, Limit: 1}); err != nil {
		t.Fatal(err)
	}
	var cites []model.SampleCitation
	if err := db.Find(&cites).Error; err != nil {
		t.Fatal(err)
	}
	if len(cites) != 1 || cites[0].Domain != "zhihu.com" || cites[0].Category != "social" || cites[0].CitationIndex != 0 {
		t.Fatalf("%+v", cites)
	}
	var samples []model.Sample
	if err := db.Find(&samples).Error; err != nil {
		t.Fatal(err)
	}
	if len(samples) == 0 || len(samples[0].WebQueries) != 1 || samples[0].WebQueries[0] != WebQueriesUnavailable {
		t.Fatalf("%+v", samples)
	}
}

func TestFillToSkipsQuestionsAlreadyRunToday(t *testing.T) {
	db := sampleDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		Name: "甲工", NoSite: true, Materials: strings.Repeat("这是面向团队的诊断工具材料。", 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrap.New(db).Run(context.Background(), p.Slug, true); err != nil {
		t.Fatal(err)
	}
	svc := New(db, NewAsker())
	svc.BypassAvailable = true
	svc.Sleep = func(time.Duration) {}
	calls := 0
	svc.Ask = func(platform, question string) AskResult {
		calls++
		return AskResult{OK: true, Answer: "甲工适合团队。"}
	}
	if _, err := svc.Run(context.Background(), p.Slug, RunInput{Platforms: []string{"deepseek"}, Limit: 1, FillTo: 3}); err != nil {
		t.Fatal(err)
	}
	first := calls
	if first == 0 {
		t.Fatal("first pass should sample")
	}
	if _, err := svc.Run(context.Background(), p.Slug, RunInput{Platforms: []string{"deepseek"}, Limit: 1, FillTo: 3}); err != nil {
		t.Fatal(err)
	}
	if calls != first {
		t.Fatalf("second pass asked again: %d -> %d", first, calls)
	}
	m, err := svc.PeriodMetrics(context.Background(), p.Slug)
	if err != nil || m == nil {
		t.Fatal(err)
	}
	raw, _ := m.Payload["platforms"].(map[string]any)
	if raw["deepseek"] == nil {
		t.Fatalf("empty rerun wiped metrics: %+v", m.Payload)
	}
}

func TestBackfillCitationsFromStoredJSON(t *testing.T) {
	db := sampleDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		Name: "甲工", URL: "https://jia.example.com", Materials: strings.Repeat("这是面向团队的诊断工具材料。", 8),
	})
	if err != nil {
		t.Fatal(err)
	}
	sm := model.Sample{
		ProjectID: p.ID, SampledOn: dateOnly(time.Now()), Platform: "deepseek", QID: "q001", Round: 1,
		SampleMode: "api", QuestionText: "有什么好用的工具？", Answer: "见知乎", OK: true,
		Cited: []model.Citation{{URL: "https://www.zhihu.com/question/1", Title: "讨论"}},
	}
	if err := db.Create(&sm).Error; err != nil {
		t.Fatal(err)
	}
	if err := BackfillCitations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var cites []model.SampleCitation
	if err := db.Find(&cites).Error; err != nil {
		t.Fatal(err)
	}
	if len(cites) != 1 || cites[0].Category != "social" {
		t.Fatalf("%+v", cites)
	}
}

func TestParseSheet(t *testing.T) {
	md := "# 表\n\n## platform: deepseek\n> 国内\n\n### q001 · 有什么好用的工具？\n\n```answer\n我推荐竞品A，它挺好用的。\n```\n"
	got := ParseSheet(md)
	if len(got) != 1 || got[0].Platform != "deepseek" || !strings.Contains(got[0].Answer, "竞品A") {
		t.Fatalf("%+v", got)
	}
}

func TestImportConfirmsCompetitor(t *testing.T) {
	db := sampleDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		Name: "测试品牌", NoSite: true, Materials: strings.Repeat("介绍材料足够长用于推导底座。", 6),
	})
	if err != nil {
		t.Fatal(err)
	}
	boot := bootstrap.New(db)
	if _, err := boot.Run(context.Background(), p.Slug, true); err != nil {
		t.Fatal(err)
	}
	f := false
	_ = boot.SaveCompetitors(context.Background(), p.Slug, []model.Competitor{
		{Name: "竞品A", Confirmed: &f},
		{Name: "竞品B", Confirmed: &f},
	})
	svc := New(db, NewAsker())
	md := "## platform: deepseek\n\n### q001 · 有什么好用的工具？\n\n```answer\n我推荐竞品A，它挺好用的。\n```\n"
	if _, err := svc.ImportMarkdown(context.Background(), p.Slug, md); err != nil {
		t.Fatal(err)
	}
	comps, _ := boot.Competitors(context.Background(), p.Slug)
	by := map[string]bool{}
	for _, c := range comps {
		by[c.Name] = model.CompetitorConfirmed(c)
	}
	if !by["竞品A"] || by["竞品B"] {
		t.Fatalf("%+v", comps)
	}
}

func TestPeriodPayloadRoundTripsAsMaps(t *testing.T) {
	rows := []Row{
		{Platform: "deepseek", QuestionID: "Q1", Round: 1, SampleMode: "api", Day: "2026-09-01", Question: "推荐工具", OK: true,
			Analysis: Analysis{BrandMentioned: true, BrandRank: 1}},
		{Platform: "deepseek", QuestionID: "Q1", Round: 1, SampleMode: "api", Day: "2026-09-02", Question: "推荐工具", OK: true},
	}
	payload, err := periodPayload(rows, testCfg(), 30)
	if err != nil {
		t.Fatal(err)
	}
	plats, ok := payload["platforms"].(map[string]any)
	if !ok {
		t.Fatalf("platforms must be map[string]any for existing readers, got %T", payload["platforms"])
	}
	ds, _ := plats["deepseek"].(map[string]any)
	if ds["mention_rate"] != 0.5 {
		t.Fatalf("mention_rate = %v", ds["mention_rate"])
	}
	if payload["window_days"] != float64(30) {
		t.Fatalf("window_days = %v", payload["window_days"])
	}
}
