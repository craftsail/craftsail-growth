// SPDX-License-Identifier: AGPL-3.0-or-later

package mdhtml

import (
	"strings"
	"testing"
)

func TestDocumentTables(t *testing.T) {
	h := Document("# 题\n\n| A | B |\n|---|---|\n| 1 | 2 |\n")
	if !strings.Contains(h, "<h1>题</h1>") || !strings.Contains(h, "<th>A</th>") || !strings.Contains(h, "<td>1</td>") {
		t.Fatalf("%s", h)
	}
}
