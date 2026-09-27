// SPDX-License-Identifier: AGPL-3.0-or-later

package crawl

import (
	"testing"

	"github.com/craftsail/craftsail-growth/internal/pkg/httputil"
)

func TestAnalyzeCollectsInternalOutLinks(t *testing.T) {
	html := `<html><body>
<a href="/pricing">p</a><a href="https://example.com/docs#x">d</a>
<a href="https://other.com/">o</a><a href="mailto:a@b.c">m</a><a href="/pricing">dup</a>
</body></html>`
	doc := Analyze("https://example.com/", httputil.Result{URL: "https://example.com/", FinalURL: "https://example.com/", Status: 200, HTML: html})
	want := map[string]bool{"https://example.com/pricing": true, "https://example.com/docs": true}
	if len(doc.OutLinks) != 2 {
		t.Fatalf("outlinks = %v", doc.OutLinks)
	}
	for _, l := range doc.OutLinks {
		if !want[l] {
			t.Fatalf("unexpected %s", l)
		}
	}
}
