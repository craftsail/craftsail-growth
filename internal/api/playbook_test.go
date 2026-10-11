// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/playbook"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func TestPlaybookRoutes(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	h.playbook = playbook.New(db)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Quill", URL: "https://quill.test/"})
	if err != nil {
		t.Fatal(err)
	}
	slug := p.Slug
	viewer, err := h.accounts.CreateUser(ctx, "viewer", pass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.accounts.Grant(ctx, viewer.ID, p.ID, model.AccessView); err != nil {
		t.Fatal(err)
	}
	viewerToken, err := h.accounts.SignIn(ctx, "viewer", pass)
	if err != nil {
		t.Fatal(err)
	}
	r := testEngine(h)

	w := call(r, "GET", "/api/projects/"+slug+"/playbook", "", testToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"seo_stage":"seoNew"`) {
		t.Fatalf("get: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/stage", `{"product_stage":"s1"}`, testToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"product_stage":"s1"`) {
		t.Fatalf("stage: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/stage", `{"product_stage":"s7"}`, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad stage: %d", w.Code)
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/signals/brandFilter", `{"confirmed":true}`, testToken)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"brandFilter":{"state":"met"`) {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+slug+"/playbook/signals/newFast", `{"confirmed":true}`, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("automatic signal: %d", w.Code)
	}

	// A view-only user can read the status but not change it.
	if res := call(r, "GET", "/api/projects/"+slug+"/playbook", "", viewerToken); res.Code != http.StatusOK {
		t.Fatalf("viewer read: %d %s", res.Code, res.Body.String())
	}
	if res := call(r, "PUT", "/api/projects/"+slug+"/playbook/stage", `{"product_stage":"s1"}`, viewerToken); res.Code != http.StatusForbidden {
		t.Fatalf("viewer stage write: %d", res.Code)
	}
	if res := call(r, "PUT", "/api/projects/"+slug+"/playbook/signals/brandFilter", `{"confirmed":true}`, viewerToken); res.Code != http.StatusForbidden {
		t.Fatalf("viewer signal write: %d", res.Code)
	}
}

// Both PUT handlers must guard a nil playbook service the same way the GET
// handler does, instead of panicking on a nil pointer deref.
func TestPlaybookRoutesNoService(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	ctx := context.Background()
	p, err := project.New(db).Create(ctx, project.CreateInput{Name: "Quill", URL: "https://quill.test/"})
	if err != nil {
		t.Fatal(err)
	}
	r := testEngine(h)

	w := call(r, "PUT", "/api/projects/"+p.Slug+"/playbook/stage", `{"product_stage":"s1"}`, testToken)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("stage without service: %d %s", w.Code, w.Body.String())
	}
	w = call(r, "PUT", "/api/projects/"+p.Slug+"/playbook/signals/brandFilter", `{"confirmed":true}`, testToken)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("signal without service: %d %s", w.Code, w.Body.String())
	}
}
