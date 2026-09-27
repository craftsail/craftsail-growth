// SPDX-License-Identifier: AGPL-3.0-or-later

package slug

import "testing"

func TestSlugifyStripsSchemeAndPunctuation(t *testing.T) {
	got := Slugify("https://Acme-Tools.io")
	if got != "acme-tools-io" {
		t.Fatalf("Slugify() = %q, want acme-tools-io", got)
	}
}

func TestSlugifyFallsBackToProject(t *testing.T) {
	if got := Slugify("***"); got != "project" {
		t.Fatalf("Slugify() = %q, want project", got)
	}
}

func TestSlugifyTruncatesTo48(t *testing.T) {
	long := "abcdefghijabcdefghijabcdefghijabcdefghijabcdefghijX"
	got := Slugify(long)
	if len([]rune(got)) > 48 {
		t.Fatalf("slug length %d > 48: %q", len([]rune(got)), got)
	}
}

func TestValidAcceptsHanAndRejectsBad(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"acme", true},
		{"品牌", true},
		{"acme-tools", true},
		{"", false},
		{"-bad", false},
		{"Bad", false},
		{"has space", false},
	}
	for _, tc := range cases {
		if got := Valid(tc.in); got != tc.want {
			t.Errorf("Valid(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
