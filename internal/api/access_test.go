// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/service/account"
)

func TestWrongTokenHeaderRejected(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("X-Craftsail-Growth-Token", "not-the-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
}

func TestLegacyTokenHeaderRejected(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("X-Seo-Geo-Audit-Token", testToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
}

func TestDisabledUserSessionRejected(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	r := testEngine(h)
	ctx := context.Background()
	u, err := h.accounts.CreateUser(ctx, "ana", pass, model.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := h.accounts.SignIn(ctx, "ana", pass)
	if err != nil {
		t.Fatal(err)
	}
	disabled := true
	if err := h.accounts.Update(ctx, u.ID, account.Patch{Disabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	if w := call(r, http.MethodGet, "/api/projects", "", tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
}

func TestOldAuthCookieRejected(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	if w := call(r, http.MethodPost, "/api/setup", `{"username":"root","password":"`+pass+`"}`, ""); w.Code != 200 {
		t.Fatalf("setup %d %s", w.Code, w.Body.String())
	}
	old := &http.Cookie{Name: "craftsail_growth_auth", Value: "whatever-was-here-before"}

	blocked := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	blocked.AddCookie(old)
	bw := httptest.NewRecorder()
	r.ServeHTTP(bw, blocked)
	if bw.Code != http.StatusUnauthorized {
		t.Fatalf("status %d %s", bw.Code, bw.Body.String())
	}

	sreq := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	sreq.AddCookie(old)
	sw := httptest.NewRecorder()
	r.ServeHTTP(sw, sreq)
	if !strings.Contains(sw.Body.String(), `"ok":false`) {
		t.Fatalf("session %s", sw.Body.String())
	}
}

func TestSessionWithTokenReportsServiceAdmin(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	if w := call(r, http.MethodPost, "/api/setup", `{"username":"root","password":"`+pass+`"}`, ""); w.Code != 200 {
		t.Fatalf("setup %d %s", w.Code, w.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	req.Header.Set("X-Craftsail-Growth-Token", testToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `"role":"admin"`) || !strings.Contains(body, `"username":"api-token"`) {
		t.Fatalf("status %d body %s", w.Code, body)
	}
}

func TestCSRFRequiresJSONForCookiePOST(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	r := testEngine(h)
	w := call(r, http.MethodPost, "/api/setup", `{"username":"root","password":"`+pass+`"}`, "")
	if w.Code != 200 {
		t.Fatalf("setup %d %s", w.Code, w.Body.String())
	}
	ck := cookieOf(t, w)

	plain := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(`{"no_site":true,"name":"Acme"}`))
	plain.Header.Set("Content-Type", "text/plain")
	plain.AddCookie(&http.Cookie{Name: sessionCookie, Value: ck})
	pw := httptest.NewRecorder()
	r.ServeHTTP(pw, plain)
	if pw.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain cookie post: status %d %s", pw.Code, pw.Body.String())
	}

	if w2 := call(r, http.MethodPost, "/api/projects", `{"no_site":true,"name":"Acme"}`, ck); w2.Code != 200 {
		t.Fatalf("json cookie post: status %d %s", w2.Code, w2.Body.String())
	}

	tplain := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(`{"no_site":true,"name":"Beta"}`))
	tplain.Header.Set("Content-Type", "text/plain")
	tplain.Header.Set("X-Craftsail-Growth-Token", testToken)
	tw := httptest.NewRecorder()
	r.ServeHTTP(tw, tplain)
	if tw.Code != 200 {
		t.Fatalf("token text/plain post: status %d %s", tw.Code, tw.Body.String())
	}
}

func TestTokenQueryOnlyForGET(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	get := httptest.NewRequest(http.MethodGet, "/api/projects?token="+testToken, nil)
	gw := httptest.NewRecorder()
	r.ServeHTTP(gw, get)
	if gw.Code != http.StatusOK {
		t.Fatalf("GET with token query: status %d %s", gw.Code, gw.Body.String())
	}

	post := httptest.NewRequest(http.MethodPost, "/api/projects?token="+testToken, strings.NewReader(`{"no_site":true,"name":"Query"}`))
	post.Header.Set("Content-Type", "application/json")
	pw := httptest.NewRecorder()
	r.ServeHTTP(pw, post)
	if pw.Code != http.StatusUnauthorized {
		t.Fatalf("POST with token query: status %d %s", pw.Code, pw.Body.String())
	}
}
