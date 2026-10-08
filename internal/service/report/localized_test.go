// SPDX-License-Identifier: AGPL-3.0-or-later

package report

import (
	"context"
	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/project"
	"strings"
	"testing"
)

func TestLocalizedWeeklyReportsWithoutGoogleAndSavedLanguage(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	projects := project.New(db)
	p, err := projects.Create(ctx, project.CreateInput{Name: "<unsafe>", NoSite: true})
	if err != nil {
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
		for _, key := range []string{"health", "stage", "ai", "actions", "observations", "next", "rulesTitle"} {
			if !strings.Contains(out.Markdown, reportText(lang, "weeklyReport."+key)) {
				t.Fatal("missing section", lang, key)
			}
		}
		if strings.Contains(out.Markdown, "{source}") || strings.Contains(out.Markdown, "{value}") {
			t.Fatal("unresolved placeholders")
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
