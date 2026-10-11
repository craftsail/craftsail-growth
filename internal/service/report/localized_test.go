// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"strings"
	"testing"
	"time"
)

func TestLocalizedWeeklyReportsWithoutGoogleAndSavedLanguage(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	projects := project.New(db)
	p, err := projects.Create(ctx, project.CreateInput{Name: "<unsafe>", NoSite: true})
	if err != nil {
		t.Fatal(err)
	}
	p.DemoScenario = "geo"
	if err := projects.Save(ctx, p); err != nil {
		t.Fatal(err)
	}
	// "Shipped" uses a rolling 7-day window (not the calendar week), so an
	// observation released 6 days ago always shows and one released 8 days
	// ago never does, regardless of which weekday the test runs on.
	tasks := []model.Task{{Code: "T1", Title: "Ship A"}, {Code: "T2", Title: "Ship B"}}
	if err := (&repo.Tasks{DB: db}).Replace(ctx, p.ID, tasks); err != nil {
		t.Fatal(err)
	}
	obs := &repo.Observations{DB: db}
	now := time.Now()
	recent := model.Observation{ProjectID: p.ID, TaskID: tasks[0].ID, TaskCode: tasks[0].Code, Hypothesis: "Shipped recently", ReleasedAt: now.AddDate(0, 0, -6).Unix()}
	stale := model.Observation{ProjectID: p.ID, TaskID: tasks[1].ID, TaskCode: tasks[1].Code, Hypothesis: "Shipped long ago", ReleasedAt: now.AddDate(0, 0, -8).Unix()}
	if err := obs.Create(ctx, &recent); err != nil {
		t.Fatal(err)
	}
	if err := obs.Create(ctx, &stale); err != nil {
		t.Fatal(err)
	}
	svc := New(db)
	for _, lang := range []string{"en", "zh", "pt"} {
		out, err := svc.Build(ctx, p.Slug, lang)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.HTML, `<html lang="`+lang+`">`) || strings.Contains(out.HTML, "<unsafe>") {
			t.Fatal("language or HTML escape", out.HTML)
		}
		for _, key := range []string{"health", "shipped", "ai", "actions", "observations", "next", "rulesTitle"} {
			if !strings.Contains(out.Markdown, reportText(lang, "weeklyReport."+key)) {
				t.Fatal("missing section", lang, key)
			}
		}
		if !strings.Contains(out.Markdown, reportText(lang, "demoGuide.banner")) {
			t.Fatal("demo provenance missing")
		}
		if strings.Contains(out.Markdown, "{source}") || strings.Contains(out.Markdown, "{value}") {
			t.Fatal("unresolved placeholders")
		}
		shippedSection := section(out.Markdown, reportText(lang, "weeklyReport.shipped"))
		if !strings.Contains(shippedSection, "Shipped recently") {
			t.Fatalf("shipped section missing an observation inside the rolling 7-day window %s:\n%s", lang, shippedSection)
		}
		if strings.Contains(shippedSection, "Shipped long ago") {
			t.Fatalf("shipped section kept an observation outside the rolling 7-day window %s:\n%s", lang, shippedSection)
		}
		saved, _ := projects.Get(ctx, p.Slug)
		if saved.ReportLanguage != lang {
			t.Fatal("language not saved")
		}
		latest, _ := svc.Latest(ctx, p.Slug)
		if latest.Language != lang {
			t.Fatal("report language stale")
		}
	}
	out, err := svc.Build(ctx, p.Slug)
	if err != nil || out.Language != "pt" {
		t.Fatal("scheduled language", out, err)
	}
	if _, err := svc.Build(ctx, p.Slug, "xx"); err == nil {
		t.Fatal("invalid language accepted")
	}
	var count int64
	db.Model(&model.Report{}).Count(&count)
	if count != 1 {
		t.Fatal("same-day report not replaced", count)
	}
	if !strings.Contains(reportTokens, "--color-primary-700") || strings.Contains(docCSS, "#1f4e79") {
		t.Fatal("report palette drift")
	}
}

// section returns the markdown between a "## heading" and the next one, so
// a test can check one section without matching text that a later section
// (e.g. "Due observations") also happens to print.
func section(markdown, heading string) string {
	start := strings.Index(markdown, "## "+heading)
	if start < 0 {
		return ""
	}
	rest := markdown[start+len("## "+heading):]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}
