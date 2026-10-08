// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
)

type progressSources struct{ items []opportunity.Item }

func (s progressSources) Collect(context.Context, string) ([]opportunity.Item, error) {
	return s.items, nil
}

func TestProgressPermissionsAndReviewedAcceptance(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	ctx := context.Background()
	p, _ := w.h.projects.Get(ctx, "alpha")
	path := "/api/projects/alpha/progress"
	for _, cookie := range []string{w.viewer, w.editor} {
		if res := call(r, "GET", path, "", cookie); res.Code != 200 {
			t.Fatalf("read %d", res.Code)
		}
	}
	if res := call(r, "GET", path, "", w.outsider); res.Code != 404 {
		t.Fatalf("outsider read %d", res.Code)
	}
	p.Brand.Definition = "A useful tool for teams"
	w.h.projects.Save(ctx, p)
	brand := call(r, "GET", "/api/projects/alpha/brand", "", w.viewer)
	var payload struct {
		Data struct {
			Revision string `json:"review_revision"`
		} `json:"data"`
	}
	json.Unmarshal(brand.Body.Bytes(), &payload)
	if payload.Data.Revision != model.BrandReviewRevision(p) {
		t.Fatal("display revision mismatch")
	}
	body := fmt.Sprintf(`{"kind":"brand","revision":%q}`, payload.Data.Revision)
	if res := call(r, "POST", path, body, testToken); res.Code != 403 {
		t.Fatalf("automation confirmation %d", res.Code)
	}
	if res := call(r, "POST", path, body, w.viewer); res.Code != 403 {
		t.Fatalf("viewer write %d", res.Code)
	}
	if res := call(r, "POST", path, body, w.outsider); res.Code != 404 {
		t.Fatalf("outsider write %d", res.Code)
	}
	if res := call(r, "POST", path, body, w.editor); res.Code != 200 {
		t.Fatalf("confirm %d %s", res.Code, res.Body.String())
	}
	p.Brand.Definition = "Changed tool"
	w.h.projects.Save(ctx, p)
	if res := call(r, "POST", path, body, w.editor); res.Code != 409 {
		t.Fatalf("stale revision %d %s", res.Code, res.Body.String())
	}
	if res := call(r, "POST", path, `{"kind":"audit_helpful","audit_id":9999}`, w.editor); res.Code != 400 {
		t.Fatalf("invalid evidence %d", res.Code)
	}
	// A title-only acceptance, an unreviewed acceptance and a dismissal are not value.
	items := []opportunity.Item{
		{Key: "audit:one", Source: "audit", Kind: "one", Title: "One", Why: "Observed issue", URLs: []string{"https://a.com"}},
		{Key: "audit:two", Source: "audit", Kind: "two", Title: "Two", Why: "Observed issue", URLs: []string{"https://a.com"}},
		{Key: "audit:title", Source: "audit", Kind: "title", Title: "Only title"},
		{Key: "audit:bot", Source: "audit", Kind: "bot", Title: "Bot", Why: "Observed issue", URLs: []string{"https://a.com"}},
		{Key: "audit:review", Source: "audit", Kind: "review", Title: "Review", Why: "Observed issue", URLs: []string{"https://a.com"}},
	}
	w.h.opportunity = opportunity.New(progressSources{items}, &repo.Tasks{DB: w.db}, func(context.Context, string) (uint64, error) { return p.ID, nil })
	if res := call(r, "POST", "/api/projects/alpha/opportunities/accept", `{"key":"audit:bot","reviewed":true}`, testToken); res.Code != 200 {
		t.Fatalf("automation acceptance %d", res.Code)
	}
	stateBefore, _ := w.h.projects.Progress(ctx, p.Slug)
	if stateBefore.FirstValueAt != nil {
		t.Fatal("service token counted as user value")
	}
	for _, tc := range []struct{ action, body string }{{"accept", `{"key":"audit:one"}`}, {"dismiss", `{"key":"audit:two","reviewed":true}`}, {"accept", `{"key":"audit:title","reviewed":true}`}} {
		if res := call(r, "POST", "/api/projects/alpha/opportunities/"+tc.action, tc.body, w.editor); res.Code != 200 {
			t.Fatalf("%d %s", res.Code, res.Body.String())
		}
		state, _ := w.h.projects.Progress(ctx, p.Slug)
		if state.FirstValueAt != nil {
			t.Fatal("non-value counted")
		}
	}
	if res := call(r, "POST", "/api/projects/alpha/opportunities/accept", `{"key":"audit:review","reviewed":true}`, w.editor); res.Code != 200 {
		t.Fatalf("%d %s", res.Code, res.Body.String())
	}
	state, _ := w.h.projects.Progress(ctx, p.Slug)
	if state.FirstValueAt == nil || state.FirstValueKind != "action_accepted" || state.FirstValueRef == 0 || state.FirstValueBy == 0 {
		t.Fatalf("%#v", state)
	}
	a := model.Audit{ProjectID: p.ID, PageCount: 2}
	w.db.Create(&a)
	if res := call(r, "POST", path, fmt.Sprintf(`{"kind":"audit_helpful","audit_id":%d}`, a.ID), w.editor); res.Code != 200 {
		t.Fatalf("helpful %d %s", res.Code, res.Body.String())
	}
	after, _ := w.h.projects.Progress(ctx, p.Slug)
	if after.FirstValueKind != "action_accepted" || after.FirstValueRef != state.FirstValueRef {
		t.Fatal("first value replaced")
	}
}
