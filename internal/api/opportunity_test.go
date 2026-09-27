// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/repo"
	"github.com/craftsail/craftsail-growth/internal/service/opportunity"
)

type fixedSources []opportunity.Item

func (f fixedSources) Collect(ctx context.Context, slug string) ([]opportunity.Item, error) {
	return f, nil
}

func TestAcceptOpportunityTwiceConflicts(t *testing.T) {
	db := testDB(t)
	h := testHandler(t, db)
	h.opportunity = opportunity.New(
		fixedSources{{Key: "audit:SPA_SHELL", Source: "audit", Priority: "P0", Title: "Static HTML has no content",
			Acceptance: map[string]any{"type": "auto", "check": "issue.absent:SPA_SHELL"}}},
		&repo.Tasks{DB: db},
		func(context.Context, string) (uint64, error) { return 1, nil },
	)
	r := testEngine(h)
	post := func() int {
		return call(r, http.MethodPost, "/api/projects/acme/opportunities/accept", `{"key":"audit:SPA_SHELL"}`, testToken).Code
	}
	if code := post(); code != http.StatusOK {
		t.Fatalf("first accept = %d", code)
	}
	if code := post(); code != http.StatusConflict {
		t.Fatalf("second accept = %d, want 409", code)
	}
	w := call(r, http.MethodGet, "/api/projects/acme/opportunities", "", testToken)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte(`"status":"open"`)) {
		t.Fatalf("list = %d %s", w.Code, w.Body.String())
	}
}
