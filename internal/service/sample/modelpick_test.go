// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import "testing"

func TestChooseModelFallsBackWhenWantMissing(t *testing.T) {
	ids := []string{"gpt-image-1.5", "grok-3-mini", "grok-4.3"}
	got := chooseModel("gpt-4o-mini", "gpt-4o-mini", ids)
	if got != "grok-3-mini" {
		t.Fatalf("got %s", got)
	}
}

func TestChooseModelKeepsWantIfListed(t *testing.T) {
	got := chooseModel("grok-4.3", "grok-3-mini", []string{"grok-3-mini", "grok-4.3"})
	if got != "grok-4.3" {
		t.Fatalf("got %s", got)
	}
}

func TestChooseModelEmptyCatalogUsesWant(t *testing.T) {
	got := chooseModel("gpt-4o-mini", "gpt-4o-mini", nil)
	if got != "gpt-4o-mini" {
		t.Fatalf("got %s", got)
	}
}
