// SPDX-License-Identifier: AGPL-3.0-or-later

package verify

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

func plat(mention any) map[string]any {
	return map[string]any{
		"market": "cn", "samples": 3, "mention_rate": mention,
		"own_domain_cite_rate": mention, "top_cited_domains": map[string]any{},
	}
}

func TestEngineAvgSkipsNil(t *testing.T) {
	m := map[string]any{"platforms": map[string]any{
		"a": plat(0.6), "b": plat(nil),
	}}
	got := EngineAvg(m, "mention_rate")
	if got == nil || *got < 0.59 || *got > 0.61 {
		t.Fatalf("%v", got)
	}
}

func TestEngineAvgAllNil(t *testing.T) {
	m := map[string]any{"platforms": map[string]any{"a": plat(nil)}}
	if EngineAvg(m, "mention_rate") != nil {
		t.Fatal("expected nil")
	}
}

func TestCheckAllNoneMarketNotFailed(t *testing.T) {
	m := map[string]any{"platforms": map[string]any{"a": plat(nil)}}
	task := model.Task{Acceptance: map[string]any{"type": "auto", "check": "metrics.mention_rate_gte:cn:0.3"}}
	o := Check(task, model.Audit{}, nil, m)
	if o.OK != nil {
		t.Fatalf("ok=%v note=%s", o.OK, o.Note)
	}
}

func TestCheckSitemap(t *testing.T) {
	task := model.Task{Acceptance: map[string]any{"type": "auto", "check": "site.has_sitemap"}}
	o := Check(task, model.Audit{Site: map[string]any{"has_sitemap": true, "sitemap_url_count": 12}}, nil, nil)
	if o.OK == nil || !*o.OK {
		t.Fatalf("%+v", o)
	}
}

func TestCheckBlockBaseline(t *testing.T) {
	task := model.Task{
		Acceptance:    map[string]any{"type": "auto", "check": "pages.block:定义"},
		BaselineCount: 10,
		Affected:      []string{"u1", "u2"},
	}
	pages := []model.AuditPage{
		{URL: "a", Blocks: map[string]any{"定义": false}},
		{URL: "b", Blocks: map[string]any{"定义": true}},
		{URL: "c", Blocks: map[string]any{"定义": false}},
	}
	o := Check(task, model.Audit{}, pages, nil)
	if o.OK == nil || !*o.OK {
		t.Fatalf("2 missing vs baseline 10 target 5, got %+v", o)
	}
}

func TestMentionCheckIgnoresMarket(t *testing.T) {
	g := plat(0.8)
	g["market"] = "global"
	m := map[string]any{"platforms": map[string]any{"cn": plat(0.2), "global": g}}
	for _, expr := range []string{"metrics.mention_rate_gte:0.5", "metrics.mention_rate_gte:cn:0.5"} {
		o := Check(model.Task{Acceptance: map[string]any{"type": "auto", "check": expr}}, model.Audit{}, nil, m)
		if o.OK == nil || !*o.OK {
			t.Fatalf("%s: the average of every engine is 50%%, got ok=%v note=%s", expr, o.OK, o.Note)
		}
	}
}
