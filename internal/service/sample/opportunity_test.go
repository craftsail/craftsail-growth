// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"strings"
	"testing"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestBuildOpportunitiesFromCitationGaps(t *testing.T) {
	day := time.Date(2026, 9, 24, 0, 0, 0, 0, time.Local)
	rows := []model.SampleCitation{
		{QID: "q001", Domain: "zhihu.com", URL: "https://www.zhihu.com/question/1", Category: "social", PageType: "forum", SampledOn: day},
		{QID: "q001", Domain: "rival.cn", URL: "https://rival.cn/vs", Category: "competitor", PageType: "comparison", SampledOn: day},
		{QID: "q002", Domain: "example.com", URL: "https://example.com/guide", Category: "brand", PageType: "article", SampledOn: day.AddDate(0, 0, -1)},
	}
	texts := map[string]string{"q001": "有什么好用的 CRM？", "q002": "官网介绍"}
	got := BuildOpportunities(rows, texts, "example.com", day)
	var social, existing bool
	for _, op := range got {
		if strings.Contains(op.Title+op.Why, "修改百科") || strings.Contains(op.Title+op.Why, "编辑维基") {
			t.Fatalf("must not suggest editing wikipedia: %+v", op)
		}
		if op.Category == "social" && op.QID == "q001" && len(op.URLs) > 0 {
			social = true
		}
		if op.Category == "existing-content" {
			existing = true
		}
	}
	if !social || !existing {
		t.Fatalf("social=%v existing=%v %+v", social, existing, got)
	}
}
