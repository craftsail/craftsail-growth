// SPDX-License-Identifier: AGPL-3.0-or-later

package generate

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func TestJSONLDFAQUsesEveryEnabledPrompt(t *testing.T) {
	qs := []model.Question{
		{QID: "c1", Market: "cn", Text: "中文问题", Enabled: true},
		{QID: "g1", Market: "global", Text: "English question", Enabled: true},
		{QID: "off", Text: "Disabled question", Enabled: false},
	}
	p := &model.Project{Name: "Acme", Site: "https://acme.test", Market: model.MarketAll}
	faqs := JSONLD(p, Facts{}, qs)["faq-page"].(map[string]any)["mainEntity"].([]any)
	if len(faqs) != 2 {
		t.Fatalf("got %v, want the two enabled prompts", faqs)
	}
}
