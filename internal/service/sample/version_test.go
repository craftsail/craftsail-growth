// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"strings"
	"testing"
	"time"
)

func TestPortugueseLibraryVersionedRunsAndReadonlyPreview(t *testing.T) {
	db := sampleDB(t)
	ctx := context.Background()
	projects := project.New(db)
	p, err := projects.Create(ctx, project.CreateInput{Name: "Test brand", NoSite: true, Materials: strings.Repeat("A tool for small teams. ", 20)})
	if err != nil {
		t.Fatal(err)
	}
	lang, region := "pt", "BR"
	if _, err := projects.Update(ctx, p.Slug, project.UpdateInput{SamplingLanguage: &lang, TargetRegion: &region}); err != nil {
		t.Fatal(err)
	}
	boot := bootstrap.New(db)
	if _, err := boot.Run(ctx, p.Slug, true); err != nil {
		t.Fatal(err)
	}
	qs, err := boot.Questions(ctx, p.Slug)
	if err != nil || len(qs) == 0 || qs[0].Language != "pt" || !strings.Contains(qs[0].Text, "Quais") {
		t.Fatalf("Portuguese %+v %v", qs, err)
	}
	s := New(db, nil)
	s.BypassAvailable = true
	s.Sleep = func(time.Duration) {}
	s.Ask = func(platform, question string) AskResult {
		return AskResult{OK: true, Answer: "Test brand is useful.", Model: "model-v1"}
	}
	preview, err := s.Preview(ctx, p.Slug, RunInput{Platforms: []string{"openai"}, Repeat: 2, Limit: 2})
	if err != nil || preview.Calls != 4 || preview.Questions != 2 {
		t.Fatalf("preview %+v %v", preview, err)
	}
	var count int64
	db.Model(&model.SampleRun{}).Count(&count)
	if count != 0 {
		t.Fatal("preview created a run")
	}
	db.Model(&model.QuestionLibrary{}).Count(&count)
	if count != 1 {
		t.Fatal("preview wrote a library")
	}
	first, err := s.Run(ctx, p.Slug, RunInput{Platforms: []string{"openai"}, Repeat: 2, Limit: 2})
	if err != nil || first.Count != 4 {
		t.Fatalf("run %+v %v", first, err)
	}
	qs[0].Text += " Para uma equipe remota?"
	if err := boot.SaveQuestions(ctx, p.Slug, qs); err != nil {
		t.Fatal(err)
	}
	second, err := s.Run(ctx, p.Slug, RunInput{Platforms: []string{"openai"}, Repeat: 1, Limit: 1})
	if err != nil || second.Count != 1 {
		t.Fatalf("run %+v %v", second, err)
	}
	var rows []model.Sample
	db.Where("project_id = ?", p.ID).Find(&rows)
	if len(rows) != 5 {
		t.Fatalf("version overwrote same-day answers: %d", len(rows))
	}
	versions := map[string]bool{}
	for _, r := range rows {
		if r.SamplingLanguage != "pt" || r.TargetRegion != "BR" || r.PromptRevision == "" {
			t.Fatalf("metadata %+v", r)
		}
		versions[r.PromptRevision] = true
	}
	if len(versions) != 2 {
		t.Fatal(versions)
	}
	view, err := s.Measure(ctx, p.Slug, MeasureQuery{Range: "30d", Access: "api", Language: "pt", Revision: preview.Revision})
	if err != nil || view.Runs != 4 || view.MixedVersions {
		t.Fatalf("filtered %+v %v", view, err)
	}
	// Editing a project setting never rewrites the existing prompt library.
	en := "en"
	if _, err := projects.Update(ctx, p.Slug, project.UpdateInput{SamplingLanguage: &en}); err != nil {
		t.Fatal(err)
	}
	current, _ := boot.Questions(ctx, p.Slug)
	if current[0].Text != qs[0].Text || current[0].Language != "pt" {
		t.Fatal("settings rewrote questions")
	}
}
func TestVersionScopeExcludesUnknownAndChangedPromptMix(t *testing.T) {
	rows := []model.Sample{{ID: 1, OK: true, QID: "q", QuestionText: "Which tool?", Platform: "x", SampleMode: "api", SamplingLanguage: "pt", TargetRegion: "BR", PromptRevision: "one", Raw: map[string]any{"model": "m", "searched": true, "strategy_version": "api-v1"}}}
	in := measureBuild{Rows: rows}
	key, counts := sampleScopeKey(rows, MeasureQuery{Access: "api"}, in)
	if key == "" {
		t.Fatal("known scope rejected")
	}
	a := &MeasureView{ScopeKey: key, ScopeCounts: counts}
	b := &MeasureView{ScopeKey: key, ScopeCounts: map[string]int{"x:api:q": 2}}
	if !ComparableScopes(a, b) {
		t.Fatal("same proportions rejected")
	}
	rows[0].PromptRevision = ""
	if k, _ := sampleScopeKey(rows, MeasureQuery{}, in); k != "" {
		t.Fatal("legacy is comparable")
	}
	b.ScopeCounts["x:api:other"] = 1
	if ComparableScopes(a, b) {
		t.Fatal("changed prompt mix accepted")
	}
}

func TestQuestionLibraryKeepsDisabledStateAndSnapshot(t *testing.T) {
	db := sampleDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Versions", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	boot := bootstrap.New(db)
	rows := []model.Question{{QID: "q1", Text: "Which tool?", Language: "en", Enabled: false}}
	if err := boot.SaveQuestions(ctx, p.Slug, rows); err != nil {
		t.Fatal(err)
	}
	saved, err := boot.Questions(ctx, p.Slug)
	if err != nil || len(saved) != 1 || saved[0].Enabled {
		t.Fatalf("disabled state %+v %v", saved, err)
	}
	history, err := boot.Libraries(ctx, p.Slug)
	if err != nil || len(history) != 1 || history[0].Questions[0].Enabled {
		t.Fatalf("snapshot %+v %v", history, err)
	}
}
