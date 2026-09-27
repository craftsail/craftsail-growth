// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestReportVisibilityMatchesDashboardRules(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Acme", Slug: "acme-report", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	day := time.Now().UTC().AddDate(0, 0, -1)
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	rows := []model.Sample{
		{ProjectID: p.ID, SampledOn: day, Platform: "openai", QID: "q1", Round: 1, SampleMode: "api", QuestionText: "best crm tools", OK: true, Mentioned: true},
		{ProjectID: p.ID, SampledOn: day, Platform: "openai", QID: "q1", Round: 2, SampleMode: "api", QuestionText: "best crm tools", OK: true, Mentioned: false},
		{ProjectID: p.ID, SampledOn: day, Platform: "openai", QID: "q2", Round: 1, SampleMode: "api", QuestionText: "is acme any good", OK: true, Mentioned: true},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out, err := New(db).Build(ctx, p.Slug)
	if err != nil {
		t.Fatal(err)
	}
	md := out.Markdown
	for _, want := range []string{
		"## AI visibility",
		"Visibility: **50%** (1 of 2 unbranded answers; 95% CI",
		"(small sample)",
		"Branded recognition: 100% of 1 branded answers",
		"## By engine",
		"## How to read this report",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q\n%s", want, md)
		}
	}
	if regexp.MustCompile(`[一-鿿]`).MatchString(md) {
		t.Fatalf("report must be English only:\n%s", md)
	}
}

func TestReadinessCardSkipsBlockedLayers(t *testing.T) {
	layers := []map[string]any{
		{"key": "access", "name": "Access", "status": "fail"},
		{"key": "discover", "name": "Discover", "status": "ok", "blocked_by": "Access"},
	}
	if got := passedLayers(layers); got != 0 {
		t.Fatalf("passed = %d, want 0", got)
	}
}
