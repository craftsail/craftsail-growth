// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

const pass = "correct-horse-battery"

func cookieOf(t *testing.T, w interface{ Result() *http.Response }) string {
	t.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie && c.Value != "" {
			return c.Value
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func TestAPIBlockedUntilSetup(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	if w := call(r, "GET", "/api/projects", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", w.Code)
	}
	w := call(r, "GET", "/api/session", "", "")
	if !strings.Contains(w.Body.String(), `"setup":true`) {
		t.Fatalf("session %s", w.Body.String())
	}
}

func TestSetupCreatesAdminOnce(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	if w := call(r, "POST", "/api/setup", `{"username":"root","password":"admin"}`, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("weak password %d", w.Code)
	}
	w := call(r, "POST", "/api/setup", `{"username":"root","password":"`+pass+`"}`, "")
	if w.Code != 200 {
		t.Fatalf("setup %d %s", w.Code, w.Body.String())
	}
	ck := cookieOf(t, w)
	if w := call(r, "GET", "/api/session", "", ck); !strings.Contains(w.Body.String(), `"role":"admin"`) {
		t.Fatalf("session %s", w.Body.String())
	}
	if w := call(r, "POST", "/api/setup", `{"username":"x2","password":"`+pass+`"}`, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("second setup %d", w.Code)
	}
}

func TestLoginLogout(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	r := testEngine(h)
	if _, err := h.accounts.CreateUser(context.Background(), "ana", pass, model.RoleMember); err != nil {
		t.Fatal(err)
	}
	if w := call(r, "POST", "/api/login", `{"username":"ana","password":"nope-nope-nope"}`, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad password %d", w.Code)
	}
	w := call(r, "POST", "/api/login", `{"username":"ana","password":"`+pass+`"}`, "")
	if w.Code != 200 {
		t.Fatalf("login %d %s", w.Code, w.Body.String())
	}
	ck := cookieOf(t, w)
	if w := call(r, "GET", "/api/projects", "", ck); w.Code != 200 {
		t.Fatalf("projects %d", w.Code)
	}
	call(r, "POST", "/api/logout", "", ck)
	if w := call(r, "GET", "/api/projects", "", ck); w.Code != http.StatusUnauthorized {
		t.Fatalf("after logout %d", w.Code)
	}
}

func TestLoginBeforeSetupRejected(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	w := call(r, "POST", "/api/login", `{"username":"root","password":"`+pass+`"}`, "")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "set up the account first") {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
}

// A seeded admin (from the environment, via SeedAdmin) must leave
// setup closed, the same as one created through /api/setup.
func TestSessionAndSetupAfterSeededAdmin(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	if err := h.accounts.SeedAdmin(context.Background(), "root", pass); err != nil {
		t.Fatal(err)
	}
	r := testEngine(h)
	if w := call(r, "GET", "/api/session", "", ""); !strings.Contains(w.Body.String(), `"setup":false`) {
		t.Fatalf("session %s", w.Body.String())
	}
	if w := call(r, "POST", "/api/setup", `{"username":"x2","password":"`+pass+`"}`, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("setup after seed %d %s", w.Code, w.Body.String())
	}
}

func TestTokenActsAsAdmin(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	if w := call(r, "GET", "/api/users", "", testToken); w.Code != 200 {
		t.Fatalf("token on admin route %d %s", w.Code, w.Body.String())
	}
}

// The default admin's password is public: until it is replaced, the API
// answers only the session and the password change.
func TestDefaultAdminMustChangePassword(t *testing.T) {
	h := testHandler(t, testDB(t))
	if err := h.accounts.SeedDefault(context.Background()); err != nil {
		t.Fatal(err)
	}
	r := testEngine(h)
	w := call(r, "POST", "/api/login", `{"username":"admin","password":"craftsailgrowth"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("login %d %s", w.Code, w.Body.String())
	}
	ck := cookieOf(t, w)
	if w := call(r, "GET", "/api/session", "", ck); !strings.Contains(w.Body.String(), `"must_change_password":true`) {
		t.Fatalf("session %s", w.Body.String())
	}
	for _, path := range []string{"/api/projects", "/api/users", "/api/keys"} {
		if w := call(r, "GET", path, "", ck); w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "4031") {
			t.Fatalf("%s before the change: %d %s", path, w.Code, w.Body.String())
		}
	}
	// No current password needed: the default one is public.
	w = call(r, "POST", "/api/me/password", `{"password":"`+pass+`"}`, ck)
	if w.Code != http.StatusOK {
		t.Fatalf("change %d %s", w.Code, w.Body.String())
	}
	ck = cookieOf(t, w)
	if w := call(r, "GET", "/api/projects", "", ck); w.Code != http.StatusOK {
		t.Fatalf("after the change %d %s", w.Code, w.Body.String())
	}
	if w := call(r, "GET", "/api/session", "", ck); !strings.Contains(w.Body.String(), `"must_change_password":false`) {
		t.Fatalf("session after %s", w.Body.String())
	}
	// From now on the current password is required again.
	if w := call(r, "POST", "/api/me/password", `{"password":"another-long-passphrase"}`, ck); w.Code != http.StatusBadRequest {
		t.Fatalf("change without current after the flag cleared: %d %s", w.Code, w.Body.String())
	}
}
