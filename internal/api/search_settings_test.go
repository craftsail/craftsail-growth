// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"testing"
)

func TestSearchSettingsValidationAndPermissions(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	path := "/api/projects/alpha"
	body := `{"search_mode":"established","search_min_impressions":800}`
	if res := call(r, "PATCH", path, body, w.editor); res.Code != 403 {
		t.Fatalf("editor %d", res.Code)
	}
	if res := call(r, "PATCH", path, body, w.viewer); res.Code != 403 {
		t.Fatalf("viewer %d", res.Code)
	}
	if res := call(r, "PATCH", path, body, w.outsider); res.Code != 403 {
		t.Fatalf("outsider %d", res.Code)
	}
	if res := call(r, "PATCH", path, body, w.admin); res.Code != 200 {
		t.Fatalf("admin %d %s", res.Code, res.Body.String())
	}
	for _, body := range []string{`{"search_mode":"unknown"}`, `{"search_mode":"new_site","search_min_impressions":99}`, `{"search_min_impressions":1000001}`, `{"search_min_impressions":500.5}`} {
		if res := call(r, "PATCH", path, body, w.admin); res.Code != 400 {
			t.Fatalf("invalid %s: %d %s", body, res.Code, res.Body.String())
		}
	}
	p, err := w.h.projects.Get(context.Background(), "alpha")
	if err != nil || p.SearchMode != "established" || p.SearchMinImpressions != 800 {
		t.Fatalf("persisted %#v %v", p, err)
	}
}
