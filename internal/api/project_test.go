// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/pkg/resp"
	"github.com/craftsail/craftsail-growth/internal/service/account"
	"github.com/craftsail/craftsail-growth/internal/service/project"
)

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) resp.Envelope {
	t.Helper()
	var env resp.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("body %s: %v", w.Body.String(), err)
	}
	return env
}

func TestListProjectsEmpty(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	w := call(r, http.MethodGet, "/api/projects", "", testToken)
	if w.Code != 200 {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Code != 0 {
		t.Fatalf("code %d", env.Code)
	}
	data, _ := env.Data.(map[string]any)
	items, _ := data["items"].([]any)
	if items == nil {
		t.Fatalf("items missing: %#v", env.Data)
	}
	if len(items) != 0 {
		t.Fatalf("len=%d", len(items))
	}
}

func TestCreateAndGetProject(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	body := `{"url":"https://acme.example","name":"Acme","market":"cn"}`
	w := call(r, http.MethodPost, "/api/projects", body, testToken)
	if w.Code != 200 {
		t.Fatalf("create status %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	data, _ := env.Data.(map[string]any)
	if data["slug"] != "acme" {
		t.Fatalf("slug %#v", data["slug"])
	}

	w = call(r, http.MethodGet, "/api/projects/acme", "", testToken)
	if w.Code != 200 {
		t.Fatalf("get status %d %s", w.Code, w.Body.String())
	}
}

func TestCreateNoSiteWithoutName(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	w := call(r, http.MethodPost, "/api/projects", `{"no_site":true,"market":"cn"}`, testToken)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
	env := decodeEnvelope(t, w)
	if env.Code == 0 {
		t.Fatal("expected error code")
	}
}

func TestAuthRequiredWhenTokenSet(t *testing.T) {
	db := testDB(t)
	h := NewHandler(project.New(db), nil, nil, nil, nil, "secret-token")
	h.accounts = account.NewForTest(db)
	r := testEngine(h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("X-Craftsail-Growth-Token", "secret-token")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("authed status %d %s", w.Code, w.Body.String())
	}
}

func TestPatchMonitorRunsPerDay(t *testing.T) {
	r := testEngine(testHandler(t, testDB(t)))
	create := call(r, http.MethodPost, "/api/projects", `{"name":"Acme","slug":"acme","market":"global","no_site":true}`, testToken)
	if create.Code != 200 {
		t.Fatalf("create %d %s", create.Code, create.Body.String())
	}
	bad := call(r, http.MethodPatch, "/api/projects/acme/monitor", `{"every_days":7,"runs_per_day":11}`, testToken)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("11 runs per day must be rejected, got %d", bad.Code)
	}
	ok := call(r, http.MethodPatch, "/api/projects/acme/monitor", `{"every_days":7,"runs_per_day":5}`, testToken)
	if ok.Code != 200 || !bytes.Contains(ok.Body.Bytes(), []byte(`"monitor_runs_per_day":5`)) {
		t.Fatalf("patch %d %s", ok.Code, ok.Body.String())
	}
}
