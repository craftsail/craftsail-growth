// SPDX-License-Identifier: AGPL-3.0-or-later

package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRecordsRedirectHops(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/b", http.StatusMovedPermanently) })
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/c", http.StatusFound) })
	mux.HandleFunc("/c", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	res := New().Fetch(srv.URL+"/a", FetchOpts{})
	if res.Status != 200 || len(res.Redirects) != 2 {
		t.Fatalf("status=%d redirects=%v", res.Status, res.Redirects)
	}
}

func TestFetchNoRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer srv.Close()
	if res := New().Fetch(srv.URL+"/", FetchOpts{}); len(res.Redirects) != 0 {
		t.Fatalf("redirects=%v", res.Redirects)
	}
}
