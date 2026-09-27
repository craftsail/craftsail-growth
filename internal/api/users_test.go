// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func idOf(t *testing.T, body []byte) uint64 {
	t.Helper()
	var v struct {
		Data struct {
			ID uint64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatal(err)
	}
	return v.Data.ID
}

func TestAdminManagesUsersAndAccess(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)

	res := call(r, "POST", "/api/users", `{"username":"new1","password":"`+pass+`","role":"member"}`, w.admin)
	if res.Code != 200 {
		t.Fatalf("create %d %s", res.Code, res.Body.String())
	}
	id := idOf(t, res.Body.Bytes())

	body := `{"items":[{"slug":"alpha","access":"view"},{"slug":"beta","access":"edit"}]}`
	if res := call(r, "PUT", fmt.Sprintf("/api/users/%d/access", id), body, w.admin); res.Code != 200 {
		t.Fatalf("access %d %s", res.Code, res.Body.String())
	}
	list := call(r, "GET", "/api/users", "", w.admin).Body.String()
	if !strings.Contains(list, `"slug":"beta","name":"Beta","access":"edit"`) {
		t.Fatalf("list lacks grants: %s", list)
	}
	if strings.Contains(list, "password") || strings.Contains(list, "hash") {
		t.Fatalf("list leaks password data: %s", list)
	}
	if res := call(r, "POST", "/api/users", `{"username":"new1","password":"`+pass+`"}`, w.admin); res.Code != http.StatusConflict {
		t.Fatalf("duplicate %d", res.Code)
	}
	if res := call(r, "POST", "/api/users", `{"username":"x9","password":"short"}`, w.admin); res.Code != http.StatusBadRequest {
		t.Fatalf("weak password %d", res.Code)
	}
	if res := call(r, "POST", "/api/users", `{"username":"x9","password":"`+pass+`","role":"member"}`, w.editor); res.Code != http.StatusForbidden {
		t.Fatalf("editor creates user: %d", res.Code)
	}
	if res := call(r, "DELETE", fmt.Sprintf("/api/users/%d", id), "", w.admin); res.Code != 200 {
		t.Fatalf("delete %d", res.Code)
	}
}

func TestChangeOwnPassword(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	if res := call(r, "POST", "/api/me/password", `{"current":"wrong-wrong-wrong","password":"brand-new-passphrase"}`, w.viewer); res.Code != http.StatusBadRequest {
		t.Fatalf("wrong current accepted: %d", res.Code)
	}
	res := call(r, "POST", "/api/me/password", `{"current":"`+pass+`","password":"brand-new-passphrase"}`, w.viewer)
	if res.Code != 200 {
		t.Fatalf("change %d %s", res.Code, res.Body.String())
	}
	// Other sessions end; the response carries a fresh cookie for this browser.
	if call(r, "GET", "/api/projects", "", w.viewer).Code != http.StatusUnauthorized {
		t.Fatal("old session survived password change")
	}
	if fresh := cookieOf(t, res); call(r, "GET", "/api/projects", "", fresh).Code != 200 {
		t.Fatal("fresh cookie does not work")
	}
}

func TestAdminCannotDeleteOrDemoteSelf(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	root, err := w.h.accounts.ByUsername(context.Background(), "root")
	if err != nil {
		t.Fatal(err)
	}
	id := root.ID
	if res := call(r, "DELETE", fmt.Sprintf("/api/users/%d", id), "", w.admin); res.Code != http.StatusBadRequest {
		t.Fatalf("self delete: %d", res.Code)
	}
	if res := call(r, "PATCH", fmt.Sprintf("/api/users/%d", id), `{"role":"member"}`, w.admin); res.Code != http.StatusBadRequest {
		t.Fatalf("self demote: %d", res.Code)
	}
}

func TestResetPasswordEndsSessions(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	vi, err := w.h.accounts.ByUsername(context.Background(), "vi")
	if err != nil {
		t.Fatal(err)
	}
	if res := call(r, "POST", fmt.Sprintf("/api/users/%d/password", vi.ID), `{"password":"reset-given-by-root"}`, w.admin); res.Code != 200 {
		t.Fatalf("reset %d %s", res.Code, res.Body.String())
	}
	if call(r, "GET", "/api/projects", "", w.viewer).Code != http.StatusUnauthorized {
		t.Fatal("viewer still signed in after reset")
	}
}
