// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSystemOperationsRequireWorkspaceAdmin(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	for _, op := range []struct{ method, path string }{{"GET", "version"}, {"POST", "update"}, {"POST", "rollback"}, {"POST", "restart"}} {
		for _, auth := range []string{"", w.viewer, w.editor, w.outsider} {
			got := call(r, op.method, "/api/system/"+op.path, `{"version":"v1.0.0"}`, auth).Code
			want := http.StatusForbidden
			if auth == "" {
				want = http.StatusUnauthorized
			}
			if got != want {
				t.Fatalf("%s: got %d want %d", op.path, got, want)
			}
		}
		// No updater is wired in this fixture: reaching its handler proves admin authorization.
		if got := call(r, op.method, "/api/system/"+op.path, `{}`, w.admin).Code; got != http.StatusConflict {
			t.Fatalf("admin %s: %d", op.path, got)
		}
	}
	req := httptest.NewRequest("POST", "/api/system/restart", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: w.admin})
	req.Header.Set("Content-Type", "text/plain")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("CSRF: %d", res.Code)
	}
}
