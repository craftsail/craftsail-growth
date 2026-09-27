// SPDX-License-Identifier: AGPL-3.0-or-later

package bootstrap

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func bootDB(t *testing.T) *gorm.DB {
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

func TestSkipLLMTemplates(t *testing.T) {
	db := bootDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		Name: "甲工", NoSite: true,
		Materials: "这是一款面向中小团队的生成引擎优化工具，用来诊断 AI 是否提及品牌并给出可验收工单。",
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	res, err := svc.Run(context.Background(), p.Slug, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Questions < 7 {
		t.Fatalf("questions=%d", res.Questions)
	}
	md, err := svc.Facts(context.Background(), p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Brand facts") {
		t.Fatalf("facts %q", md[:min(80, len(md))])
	}
	qs, _ := svc.Questions(context.Background(), p.Slug)
	if qs[0].Intent == "" {
		t.Fatal("intent empty")
	}
}

func TestLLMExtractsBrandAndFiltersFakes(t *testing.T) {
	db := bootDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{
		URL: "https://t.example.com", Name: "旧名",
	})
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Page{ProjectID: p.ID, URL: "https://t.example.com/", Title: "首页",
		StatusCode: 200, Text: strings.Repeat("这是一款诊断工具 ", 30), WordCount: 80})
	svc := New(db)
	svc.LLM = &fakeLLM{seq: []map[string]any{
		{"name": "甲工智能", "aliases": []any{"甲工"}, "industry": "GEO", "definition": "甲工智能是面向团队的 GEO 工具。",
			"uncertain": []any{"成立时间"}},
		{"competitors": []any{
			map[string]any{"name": "竞品A", "market": "cn"},
			map[string]any{"name": "工具A", "market": "cn"},
		}},
		{"questions": []any{
			map[string]any{"id": "q001", "group": "推荐", "market": "cn", "text": "有什么好用的 GEO 工具？"},
		}},
	}}
	res, err := svc.Run(context.Background(), p.Slug, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Brand.Industry != "GEO" {
		t.Fatalf("brand %+v", res.Brand)
	}
	comps, _ := svc.Competitors(context.Background(), p.Slug)
	if len(comps) != 1 || comps[0].Name != "竞品A" {
		t.Fatalf("%+v", comps)
	}
	got, _ := project.New(db).Get(context.Background(), p.Slug)
	if got.Name != "甲工智能" {
		t.Fatalf("name %s", got.Name)
	}
}

// fakeLLM answers by prompt kind: seq holds the brand, competitor and
// prompt-library answers in that order. The last two are asked concurrently.
type fakeLLM struct {
	seq []map[string]any
}

func (f *fakeLLM) AskJSON(ctx context.Context, prompt string) (map[string]any, error) {
	i := 0
	switch {
	case strings.Contains(prompt, "List real competitors"):
		i = 1
	case strings.Contains(prompt, "Design a prompt library"):
		i = 2
	}
	if i >= len(f.seq) {
		return nil, nil
	}
	return f.seq[i], nil
}

// slowLLM counts how many calls are in flight at once.
type slowLLM struct {
	fakeLLM
	mu            sync.Mutex
	inFlight, max int
}

func (s *slowLLM) AskJSON(ctx context.Context, prompt string) (map[string]any, error) {
	s.mu.Lock()
	s.inFlight++
	if s.inFlight > s.max {
		s.max = s.inFlight
	}
	s.mu.Unlock()
	time.Sleep(50 * time.Millisecond)
	s.mu.Lock()
	s.inFlight--
	s.mu.Unlock()
	return s.fakeLLM.AskJSON(ctx, prompt)
}

func TestCompetitorsAndPromptsAreAskedTogether(t *testing.T) {
	db := bootDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{URL: "https://t.example.com", Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Page{ProjectID: p.ID, URL: "https://t.example.com/", Title: "Home",
		StatusCode: 200, Text: strings.Repeat("Acme converts documents. ", 30), WordCount: 90})
	svc := New(db)
	llm := &slowLLM{fakeLLM: fakeLLM{seq: []map[string]any{
		{"name": "Acme", "industry": "document tools", "definition": "Acme converts documents."},
		{"competitors": []any{map[string]any{"name": "Pandoc"}}},
		{"questions": []any{map[string]any{"id": "q001", "group": "推荐", "text": "Which tool converts Word to PDF?"}}},
	}}}
	svc.LLM = llm
	if _, err := svc.Run(context.Background(), p.Slug, false); err != nil {
		t.Fatal(err)
	}
	if llm.max != 2 {
		t.Fatalf("max concurrent calls %d, want 2", llm.max)
	}
}

func TestRunStopsWhenCancelled(t *testing.T) {
	db := bootDB(t)
	p, err := project.New(db).Create(context.Background(), project.CreateInput{URL: "https://t.example.com", Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Page{ProjectID: p.ID, URL: "https://t.example.com/", Title: "Home",
		StatusCode: 200, Text: strings.Repeat("Acme converts documents. ", 30), WordCount: 90})
	svc := New(db)
	svc.LLM = &fakeLLM{seq: []map[string]any{{"name": "Acme"}}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Run(ctx, p.Slug, false); err == nil {
		t.Fatal("want an error after cancel, got nil")
	}
	if qs, _ := svc.Questions(context.Background(), p.Slug); len(qs) != 0 {
		t.Fatalf("saved %d prompts after cancel", len(qs))
	}
}
