// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestMarkdownListsWorstPagesAndIssues(t *testing.T) {
	p := &model.Project{Name: "甲工", Site: "https://example.com"}
	rep := &Report{
		Audit: model.Audit{
			AvgScore: 30, PageCount: 1,
			GradeDistribution: map[string]int{"D": 1},
		},
		Findings: []IssueRow{{Finding: Finding{Code: "NO_SITEMAP"}, Severity: SevWarning, Layer: LayerDiscover}},
		Pages: []model.AuditPage{{
			URL: "https://example.com/sales", Title: "Contact Sales", Score: 25.6, Grade: "D", WordCount: 34,
			IssueCodes: []string{"SPA_SHELL"},
			Dimensions: map[string]any{"可抓取性": 5.0, "内容长度": 0.0},
			Blocks:     map[string]any{"定义": false, "数字事实": false},
		}},
	}
	md := Markdown(p, rep)
	for _, want := range []string{"甲工", "https://example.com/sales", "Static HTML has no content", "SPA_SHELL", "No sitemap.xml", "Missing blocks: definition, numbers", "Crawlability (15)", "Crawlability 5/15"} {
		if !strings.Contains(md, want) {
			t.Fatalf("missing %q in %s", want, md)
		}
	}
}
