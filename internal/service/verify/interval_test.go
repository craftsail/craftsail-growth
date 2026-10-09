// SPDX-License-Identifier: AGPL-3.0-or-later

package verify

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestIssueAbsentCheck(t *testing.T) {
	task := model.Task{Acceptance: map[string]any{"type": "auto", "check": "issue.absent:SPA_SHELL"}}
	in := Inputs{Issues: []model.AuditIssue{{Code: "NO_JSONLD"}}, HaveIssues: true}
	if o := CheckWith(task, in); o.OK == nil || !*o.OK {
		t.Fatalf("SPA_SHELL absent should pass: %+v", o)
	}
	in.Issues = append(in.Issues, model.AuditIssue{Code: "SPA_SHELL", URL: "https://e.com/x"})
	if o := CheckWith(task, in); o.OK == nil || *o.OK {
		t.Fatalf("SPA_SHELL present should fail: %+v", o)
	}
	if o := CheckWith(task, Inputs{}); o.OK != nil {
		t.Fatalf("without an audit the check is inconclusive: %+v", o)
	}
}

func TestPromptUpUsesInterval(t *testing.T) {
	task := model.Task{
		Acceptance: map[string]any{"type": "auto", "check": "metrics.prompt_up:q1:api"},
		Baseline:   map[string]any{"prompt": map[string]any{"x": 1.0, "n": 10.0}},
	}
	up := Inputs{Prompt: map[string]map[string][2]int{"api": {"q1": {8, 10}}}}
	if o := CheckWith(task, up); o.OK == nil || !*o.OK {
		t.Fatalf("1/10 -> 8/10 should pass: %+v", o)
	}
	flat := Inputs{Prompt: map[string]map[string][2]int{"api": {"q1": {2, 10}}}}
	if o := CheckWith(task, flat); o.OK != nil {
		t.Fatalf("1/10 -> 2/10 is inconclusive, got %+v", o)
	}
}

func TestVisibilityNotDown(t *testing.T) {
	task := model.Task{
		Acceptance: map[string]any{"type": "auto", "check": "metrics.visibility_not_down:api"},
		Baseline:   map[string]any{"visibility": map[string]any{"x": 18.0, "n": 30.0}},
	}
	if o := CheckWith(task, Inputs{Visibility: map[string][2]int{"api": {17, 30}}}); o.OK == nil || !*o.OK {
		t.Fatalf("recovered to 17/30 should pass: %+v", o)
	}
	if o := CheckWith(task, Inputs{Visibility: map[string][2]int{"api": {6, 30}}}); o.OK == nil || *o.OK {
		t.Fatalf("still 6/30 should fail: %+v", o)
	}
}

func TestNextStatus(t *testing.T) {
	yes, no := true, false
	cases := []struct {
		from string
		ok   *bool
		want string
	}{
		{model.TaskOpen, &yes, model.TaskVerified},
		{model.TaskDone, &yes, model.TaskVerified},
		{model.TaskRegressed, &yes, model.TaskVerified},
		{model.TaskVerified, &no, model.TaskRegressed},
		{model.TaskDone, &no, model.TaskDone},
		{model.TaskVerified, nil, model.TaskVerified},
		{model.TaskDismissed, &yes, model.TaskDismissed},
	}
	for _, c := range cases {
		if got := NextStatus(c.from, c.ok); got != c.want {
			t.Fatalf("%s + %v -> %s, want %s", c.from, c.ok, got, c.want)
		}
	}
}

func TestReleasedTaskNeedsFreshCrawlSnapshot(t *testing.T) {
	release := int64(500)
	task := model.Task{ReleasedAt: &release, Affected: []string{"https://a.test/"}, Acceptance: map[string]any{"type": "auto", "check": "issue.absent:SPA_SHELL"}}
	in := Inputs{HaveIssues: true, Audit: model.Audit{RunAt: 600}, Pages: []model.AuditPage{{URL: "https://a.test/", CrawledAt: 400}}}
	if o := CheckWith(task, in); o.OK != nil {
		t.Fatal("old crawl verified release", o)
	}
	in.Pages[0].CrawledAt = 550
	if o := CheckWith(task, in); o.OK == nil || !*o.OK {
		t.Fatal(o)
	}
	task.Acceptance["check"] = "metrics.visibility_not_down:api"
	if o := CheckWith(task, in); o.OK != nil {
		t.Fatal("rolling metrics verified release", o)
	}
}
