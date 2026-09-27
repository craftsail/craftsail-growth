// SPDX-License-Identifier: AGPL-3.0-or-later

package slug

import (
	"regexp"
	"strings"
)

// Valid matches the Python SLUG_OK rule: 1–48 chars, starts with
// lowercase latin / digit / CJK unified ideograph, then those plus hyphen.
var validRe = regexp.MustCompile(`^[a-z0-9一-鿿][a-z0-9一-鿿-]{0,47}$`)

var (
	schemeRe = regexp.MustCompile(`(?i)^https?://`)
	keepRe   = regexp.MustCompile(`[^a-z0-9一-鿿]+`)
)

func Slugify(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = schemeRe.ReplaceAllString(s, "")
	s = keepRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	runes := []rune(s)
	if len(runes) > 48 {
		s = string(runes[:48])
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		return "project"
	}
	return s
}

func Valid(s string) bool {
	return validRe.MatchString(s)
}
