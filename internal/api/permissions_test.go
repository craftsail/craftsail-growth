// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gorm.io/gorm"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/bootstrap"
	"github.com/craftsail/craftsail-growth/internal/service/jobs"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

type world struct {
	h                               *Handler
	db                              *gorm.DB
	admin, editor, viewer, outsider string // session cookies
}

// newWorld: projects "alpha" and "beta"; editor edits alpha; viewer views
// alpha; outsider has nothing.
func newWorld(t *testing.T) *world {
	t.Helper()
	db := testDB(t)
	h := testHandler(t, db)
	h.EnvRoot = t.TempDir()
	h.bootstrap = bootstrap.New(db)
	ctx := context.Background()
	ps := project.New(db)
	alpha, err := ps.Create(ctx, project.CreateInput{Name: "Alpha", Slug: "alpha", NoSite: true, Materials: strings.Repeat("Alpha makes tools. ", 10)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ps.Create(ctx, project.CreateInput{Name: "Beta", Slug: "beta", NoSite: true, Materials: strings.Repeat("Beta makes tools. ", 10)}); err != nil {
		t.Fatal(err)
	}
	w := &world{h: h, db: db}
	h.jobs = jobs.New(db)
	mk := func(name, role, access string) string {
		u, err := h.accounts.CreateUser(ctx, name, pass, role)
		if err != nil {
			t.Fatal(err)
		}
		if access != "" {
			if err := h.accounts.Grant(ctx, u.ID, alpha.ID, access); err != nil {
				t.Fatal(err)
			}
		}
		tok, err := h.accounts.SignIn(ctx, name, pass)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	w.admin = mk("root", model.RoleAdmin, "")
	w.editor = mk("ed", model.RoleMember, model.AccessEdit)
	w.viewer = mk("vi", model.RoleMember, model.AccessView)
	w.outsider = mk("out", model.RoleMember, "")
	return w
}

func TestEveryAPIRouteDeclaresAPermission(t *testing.T) {
	h := testHandler(t, testDB(t))
	r := testEngine(h)
	public := map[string]bool{"POST /api/login": true, "POST /api/logout": true, "POST /api/setup": true, "GET /api/session": true}
	for _, rt := range r.Routes() {
		key := rt.Method + " " + rt.Path
		if !strings.HasPrefix(rt.Path, "/api/") || public[key] {
			continue
		}
		if _, ok := h.perms[key]; !ok {
			t.Errorf("%s has no permission; register it with h.route", key)
		}
	}
}

func TestProjectListIsFiltered(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	cases := []struct {
		name, cookie string
		want         []string
	}{{"admin", w.admin, []string{"alpha", "beta"}}, {"viewer", w.viewer, []string{"alpha"}}, {"outsider", w.outsider, nil}}
	for _, c := range cases {
		body := call(r, "GET", "/api/projects", "", c.cookie).Body.String()
		for _, slug := range []string{"alpha", "beta"} {
			has := strings.Contains(body, `"slug":"`+slug+`"`)
			if has != contains(c.want, slug) {
				t.Errorf("%s: %s listed=%v", c.name, slug, has)
			}
		}
	}
	if body := call(r, "GET", "/api/projects", "", w.viewer).Body.String(); !strings.Contains(body, `"access":"view"`) {
		t.Errorf("viewer list lacks access field: %s", body)
	}
}

func TestPermissionMatrix(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	const ok, forbidden, missing = http.StatusOK, http.StatusForbidden, http.StatusNotFound
	cases := []struct {
		method, path, body              string
		admin, editor, viewer, outsider int
	}{
		// read a shared project
		{"GET", "/api/projects/alpha/questions", "", ok, ok, ok, missing},
		// an unshared project answers 404, never 403
		{"GET", "/api/projects/beta/questions", "", ok, missing, missing, missing},
		// edit a shared project
		{"PUT", "/api/projects/alpha/questions", `{"items":[]}`, ok, ok, forbidden, missing},
		{"PUT", "/api/projects/alpha/competitors", `{"items":[]}`, ok, ok, forbidden, missing},
		// workspace routes are admin-only, whatever the project access
		{"GET", "/api/keys", "", ok, forbidden, forbidden, forbidden},
		{"POST", "/api/projects", `{"name":"Gamma","no_site":true,"materials":"Gamma makes tools and more tools for teams."}`, ok, forbidden, forbidden, forbidden},
		{"PATCH", "/api/projects/alpha", `{"name":"Alpha 2"}`, ok, forbidden, forbidden, forbidden},
	}
	for _, c := range cases {
		for _, who := range []struct {
			name, cookie string
			want         int
		}{{"admin", w.admin, c.admin}, {"editor", w.editor, c.editor}, {"viewer", w.viewer, c.viewer}, {"outsider", w.outsider, c.outsider}} {
			res := call(r, c.method, c.path, c.body, who.cookie)
			if res.Code != who.want {
				t.Errorf("%s %s as %s: %d, want %d (%s)", c.method, c.path, who.name, res.Code, who.want, res.Body.String())
			}
		}
	}
}

func TestJobsFollowProjectAccess(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	if got := call(r, "GET", "/api/jobs?slug=beta", "", w.viewer).Code; got != http.StatusNotFound {
		t.Fatalf("viewer lists jobs of unshared project: %d", got)
	}
	if got := call(r, "GET", "/api/jobs", "", w.viewer).Code; got != http.StatusForbidden {
		t.Fatalf("viewer lists all jobs: %d", got)
	}
	if got := call(r, "POST", "/api/projects/alpha/jobs", `{"action":"crawl"}`, w.viewer).Code; got != http.StatusForbidden {
		t.Fatalf("viewer starts a project job: %d", got)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// jobFor stores a finished job on a project and returns its id.
func (w *world) jobFor(t *testing.T, slug string) uint64 {
	t.Helper()
	p, err := project.New(w.db).Get(context.Background(), slug)
	if err != nil {
		t.Fatal(err)
	}
	j := model.Job{ProjectID: &p.ID, Action: "crawl", Status: "running", Log: "x"}
	if err := w.db.Create(&j).Error; err != nil {
		t.Fatal(err)
	}
	return j.ID
}

func TestJobByIDFollowsProjectAccess(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	alpha, beta := w.jobFor(t, "alpha"), w.jobFor(t, "beta")
	cases := []struct {
		name, method, path, cookie string
		want                       int
	}{
		{"viewer reads own project's job", "GET", fmt.Sprintf("/api/jobs/%d", alpha), w.viewer, http.StatusOK},
		{"viewer reads other project's job", "GET", fmt.Sprintf("/api/jobs/%d", beta), w.viewer, http.StatusNotFound},
		{"outsider reads a job", "GET", fmt.Sprintf("/api/jobs/%d", alpha), w.outsider, http.StatusNotFound},
		{"viewer stops a job", "POST", fmt.Sprintf("/api/jobs/%d/stop", alpha), w.viewer, http.StatusForbidden},
		{"editor stops other project's job", "POST", fmt.Sprintf("/api/jobs/%d/stop", beta), w.editor, http.StatusNotFound},
		{"token reads any job", "GET", fmt.Sprintf("/api/jobs/%d", beta), testToken, http.StatusOK},
	}
	for _, c := range cases {
		if got := call(r, c.method, c.path, `{}`, c.cookie).Code; got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

// Structural guarantee: writes need edit or admin (or a handler check on
// the short list below), and project permissions sit on :slug routes.
func TestPermissionTableShape(t *testing.T) {
	h := testHandler(t, testDB(t))
	testEngine(h)
	handlerChecked := map[string]bool{"POST /api/jobs/:id/stop": true, "POST /api/me/password": true}
	for key, p := range h.perms {
		method, path, _ := strings.Cut(key, " ")
		if method != "GET" && p != permEdit && p != permAdmin && !handlerChecked[key] {
			t.Errorf("%s changes data but is %q", key, p)
		}
		if (p == permView || p == permEdit) && !strings.Contains(path, ":slug") {
			t.Errorf("%s is %q but has no :slug", key, p)
		}
	}
}

func TestEditorCannotRenameThroughBrand(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	if res := call(r, "PUT", "/api/projects/alpha/brand", `{"name":"Hijacked","brand":{}}`, w.editor); res.Code != http.StatusOK {
		t.Fatalf("editor saves brand: %d %s", res.Code, res.Body.String())
	}
	p, err := project.New(w.db).Get(context.Background(), "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Alpha" {
		t.Fatalf("editor renamed the project to %q", p.Name)
	}
}

func TestAccessChangesAreAllOrNothing(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	out, err := w.h.accounts.ByUsername(context.Background(), "out")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"items":[{"slug":"alpha","access":"view"},{"slug":"nope","access":"view"}]}`
	if res := call(r, "PUT", fmt.Sprintf("/api/users/%d/access", out.ID), body, w.admin); res.Code == http.StatusOK {
		t.Fatal("unknown project accepted")
	}
	if got := call(r, "GET", "/api/projects/alpha", "", w.outsider).Code; got != http.StatusNotFound {
		t.Fatalf("first grant applied despite the error: %d", got)
	}
}

func TestPromotionClearsOldGrants(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	vi, err := w.h.accounts.ByUsername(context.Background(), "vi")
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"admin", "member"} {
		if res := call(r, "PATCH", fmt.Sprintf("/api/users/%d", vi.ID), `{"role":"`+role+`"}`, w.admin); res.Code != http.StatusOK {
			t.Fatalf("set %s: %d %s", role, res.Code, res.Body.String())
		}
	}
	grants, err := w.h.accounts.Grants(context.Background(), vi.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(grants) != 0 {
		t.Fatalf("old grants came back after demotion: %v", grants)
	}
}

func TestAdminCannotResetOwnPasswordWithoutCurrent(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	root, err := w.h.accounts.ByUsername(context.Background(), "root")
	if err != nil {
		t.Fatal(err)
	}
	if res := call(r, "POST", fmt.Sprintf("/api/users/%d/password", root.ID), `{"password":"another-long-passphrase"}`, w.admin); res.Code != http.StatusBadRequest {
		t.Fatalf("self reset: %d", res.Code)
	}
}
