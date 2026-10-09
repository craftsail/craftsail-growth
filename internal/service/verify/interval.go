// SPDX-License-Identifier: AGPL-3.0-or-later

package verify

import (
	"fmt"
	"strings"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
)

// Inputs is everything the acceptance checks read, gathered once per run.
type Inputs struct {
	Audit      model.Audit
	Pages      []model.AuditPage
	Metrics    map[string]any
	Issues     []model.AuditIssue
	HaveIssues bool                         // false when no audit has run yet
	Visibility map[string][2]int            // access -> x, n over the current window
	Prompt     map[string]map[string][2]int // access -> qid -> x, n
}

// CheckWith handles the interval-based checks used by opportunities and
// falls back to Check for the older expressions.
func CheckWith(task model.Task, in Inputs) Outcome {
	expr, _ := task.Acceptance["check"].(string)
	if task.ReleasedAt != nil {
		if strings.HasPrefix(expr, "metrics.") {
			return Outcome{Note: "release effect is evaluated in its fixed observation window"}
		}
		if in.Audit.RunAt <= *task.ReleasedAt {
			return Outcome{Note: "run a fresh crawl and audit after release"}
		}
		if len(task.Affected) == 0 {
			return Outcome{Note: "release targets need fresh crawl evidence"}
		}
		by := map[string]model.AuditPage{}
		for _, p := range in.Pages {
			by[p.URL] = p
		}
		for _, u := range task.Affected {
			if p, ok := by[u]; !ok || p.CrawledAt <= *task.ReleasedAt {
				return Outcome{Note: "release targets need fresh crawl evidence"}
			}
		}
	}
	yes, no := true, false
	switch {
	case strings.HasPrefix(expr, "issue.absent:"):
		code := strings.TrimPrefix(expr, "issue.absent:")
		if !in.HaveIssues {
			return Outcome{Note: "no audit yet; run a crawl and audit"}
		}
		n := 0
		for _, r := range in.Issues {
			if r.Code == code {
				n++
			}
		}
		if n == 0 {
			return Outcome{OK: &yes, Note: code + " is gone"}
		}
		return Outcome{OK: &no, Note: fmt.Sprintf("%s still found %d times", code, n)}
	case strings.HasPrefix(expr, "metrics.prompt_up:"):
		parts := strings.Split(strings.TrimPrefix(expr, "metrics.prompt_up:"), ":")
		access := "api"
		if len(parts) > 1 {
			access = parts[1]
		}
		bx, bn := baseXN(task.Baseline, "prompt")
		cur, ok := in.Prompt[access][parts[0]]
		if !ok || bn == 0 || cur[1] == 0 {
			return Outcome{Note: "missing baseline or current samples for this prompt"}
		}
		note := fmt.Sprintf("%d/%d -> %d/%d", bx, bn, cur[0], cur[1])
		switch metrics.Change(bx, bn, cur[0], cur[1]) {
		case metrics.ChangeUp:
			return Outcome{OK: &yes, Note: note + ", a significant rise"}
		case metrics.ChangeDown:
			return Outcome{OK: &no, Note: note + ", a significant drop"}
		}
		return Outcome{Note: note + ", within noise; keep sampling"}
	case strings.HasPrefix(expr, "metrics.visibility_not_down:"):
		access := strings.TrimPrefix(expr, "metrics.visibility_not_down:")
		bx, bn := baseXN(task.Baseline, "visibility")
		cur := in.Visibility[access]
		if bn == 0 || cur[1] == 0 {
			return Outcome{Note: "missing baseline or current samples"}
		}
		if metrics.Change(bx, bn, cur[0], cur[1]) == metrics.ChangeDown {
			return Outcome{OK: &no, Note: fmt.Sprintf("still significantly below the baseline %d/%d (now %d/%d)", bx, bn, cur[0], cur[1])}
		}
		return Outcome{OK: &yes, Note: fmt.Sprintf("back within noise of the baseline (now %d/%d)", cur[0], cur[1])}
	}
	return Check(task, in.Audit, in.Pages, in.Metrics)
}

func baseXN(b map[string]any, key string) (int, int) {
	m, _ := b[key].(map[string]any)
	return toInt(m["x"]), toInt(m["n"])
}

func toInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}

// NextStatus: a pass verifies; a fail only regresses something that was
// verified; an inconclusive check changes nothing; dismissed stays dismissed.
func NextStatus(from string, ok *bool) string {
	if from == model.TaskDismissed || ok == nil {
		return from
	}
	if *ok {
		return model.TaskVerified
	}
	if from == model.TaskVerified {
		return model.TaskRegressed
	}
	return from
}
