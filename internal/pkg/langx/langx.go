// SPDX-License-Identifier: AGPL-3.0-or-later

// Package langx guesses the main language of a text. It only tells Chinese
// from everything else, which is what prompt and snippet generation need.
package langx

import "unicode"

// Detect returns "zh" when Han characters make up at least a fifth of the
// letters in text, and "en" otherwise (including empty text).
func Detect(text string) string {
	han, letters := 0, 0
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r):
			han++
			letters++
		case unicode.IsLetter(r):
			letters++
		}
	}
	if letters > 0 && han*5 >= letters {
		return "zh"
	}
	return "en"
}
