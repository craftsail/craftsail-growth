// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"strings"
	"testing"
)

func TestAuthCodeURLHasOfflineConsent(t *testing.T) {
	raw := AuthCodeURL("client", "http://127.0.0.1:8765/api/google/callback", "nonce.acme")
	if !strings.Contains(raw, "access_type=offline") || !strings.Contains(raw, "prompt=consent") {
		t.Fatal(raw)
	}
	if !strings.Contains(raw, "webmasters.readonly") || !strings.Contains(raw, "analytics.readonly") {
		t.Fatal(raw)
	}
	if !strings.Contains(raw, "redirect_uri=") || !strings.Contains(raw, "state=nonce.acme") {
		t.Fatal(raw)
	}
}
