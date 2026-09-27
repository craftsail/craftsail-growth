// SPDX-License-Identifier: AGPL-3.0-or-later

package audit

import (
	"os"
	"regexp"
	"testing"
)

func TestRegistryEntriesAreComplete(t *testing.T) {
	sev := map[string]bool{SevCritical: true, SevWarning: true, SevInfo: true}
	layer := map[string]bool{LayerAccess: true, LayerDiscover: true, LayerUnderstand: true, LayerCite: true}
	surface := map[string]bool{SurfaceSEO: true, SurfaceGEO: true, SurfaceBoth: true}
	scope := map[string]bool{ScopeSite: true, ScopePage: true, ScopeContent: true}
	ev := map[string]bool{EvStandard: true, EvVendor: true, EvExperiment: true, EvObservational: true, EvHeuristic: true}
	for code, is := range Registry {
		if is.Code != code {
			t.Fatalf("%s: Code field = %q", code, is.Code)
		}
		if !sev[is.Severity] || !layer[is.Layer] || !surface[is.Surface] || !scope[is.Scope] || !ev[is.Evidence] {
			t.Fatalf("%s: bad enum %+v", code, is)
		}
		if is.Title == "" || is.Why == "" || is.Fix == "" {
			t.Fatalf("%s: missing text", code)
		}
		if is.Evidence != EvHeuristic && len(is.Refs) == 0 {
			t.Fatalf("%s: evidence %s needs at least one reference", code, is.Evidence)
		}
		for _, r := range is.Refs {
			if _, ok := References[r]; !ok {
				t.Fatalf("%s: unknown reference %q", code, r)
			}
		}
		if (is.Evidence == EvObservational || is.Evidence == EvHeuristic) && is.Severity == SevCritical {
			t.Fatalf("%s: observational or heuristic rules cannot be critical", code)
		}
	}
}

func TestEveryEmittedCodeIsRegistered(t *testing.T) {
	re := regexp.MustCompile(`(?:issue|find|siteFind)\("([A-Z0-9_]+)"`)
	for _, f := range []string{"score.go", "seo.go", "site.go"} {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(src), -1) {
			if _, ok := Registry[m[1]]; !ok {
				t.Fatalf("%s emits %s but it is not in Registry", f, m[1])
			}
		}
	}
}

func TestLookupUnknownCodeIsSafe(t *testing.T) {
	is := Lookup("NOT_A_CODE")
	if is.Code != "NOT_A_CODE" || is.Severity != SevInfo {
		t.Fatalf("got %+v", is)
	}
}
