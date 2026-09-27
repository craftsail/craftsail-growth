// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"os"
	"strings"
	"testing"
)

// Every rule in the registry needs a translation in each dashboard language.
func TestRuleTranslationsCoverRegistry(t *testing.T) {
	for _, loc := range []string{"zh", "pt"} {
		b, err := os.ReadFile("../../../web/src/i18n/rules/" + loc + ".ts")
		if err != nil {
			t.Fatal(err)
		}
		for code := range Registry {
			if !strings.Contains(string(b), `"`+code+`": {`) {
				t.Errorf("web/src/i18n/rules/%s.ts has no translation for %s", loc, code)
			}
		}
	}
}
