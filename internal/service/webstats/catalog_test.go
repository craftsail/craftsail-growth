// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "testing"

func TestCatalogMarksUnverified(t *testing.T) {
	raw := []byte(`{"siteEntry":[{"siteUrl":"sc-domain:ex.com","permissionLevel":"siteOwner"},{"siteUrl":"https://other.com/","permissionLevel":"siteUnverifiedUser"}]}`)
	got := ParseGSCSites(raw)
	if len(got) != 2 || !got[0].Selectable || got[1].Selectable {
		t.Fatalf("%#v", got)
	}
	if got[0].SiteURL != "sc-domain:ex.com" {
		t.Fatalf("normalized %q", got[0].SiteURL)
	}
}
