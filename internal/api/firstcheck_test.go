// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"testing"
)

func TestFirstCheckJobUsesProjectEditPermission(t *testing.T) {
	w := newWorld(t)
	r := testEngine(w.h)
	gate := make(chan struct{})
	defer close(gate)
	w.h.jobs.Register("first-check", func(ctx context.Context, _ string, _ map[string]any, _ func(string)) error {
		select {
		case <-gate:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	body := `{"action":"first-check"}`
	path := "/api/projects/alpha/jobs"
	if res := call(r, "POST", path, body, w.viewer); res.Code != 403 {
		t.Fatalf("viewer %d", res.Code)
	}
	if res := call(r, "POST", path, body, w.outsider); res.Code != 404 {
		t.Fatalf("outsider %d", res.Code)
	}
	if res := call(r, "POST", path, body, w.editor); res.Code != 200 {
		t.Fatalf("editor %d %s", res.Code, res.Body.String())
	}
	if res := call(r, "POST", path, body, w.editor); res.Code != 409 {
		t.Fatalf("duplicate %d %s", res.Code, res.Body.String())
	}
}
