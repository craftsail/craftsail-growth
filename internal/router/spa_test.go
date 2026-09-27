// SPDX-License-Identifier: AGPL-3.0-or-later

package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func get(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestSPAServesIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	serveSPA(r, fstest.MapFS{
		"index.html":       {Data: []byte(`<div id="root"></div>`)},
		"assets/index.css": {Data: []byte(`body{}`)},
	})
	for _, p := range []string{"/", "/projects/demo/overview"} {
		w := get(t, r, p)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "root") {
			t.Fatalf("%s: status %d body %q", p, w.Code, w.Body.String())
		}
	}
	if w := get(t, r, "/assets/index.css"); !strings.HasPrefix(w.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("css content type %q", w.Header().Get("Content-Type"))
	}
}

func TestSPANotBuilt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	serveSPA(r, fstest.MapFS{".gitkeep": {}})
	w := get(t, r, "/")
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "make build") {
		t.Fatalf("status %d body %q", w.Code, w.Body.String())
	}
}

func TestSPAAPI404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SPA(r)
	w := get(t, r, "/api/missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"code"`) {
		t.Fatalf("body %s", w.Body.String())
	}
}
