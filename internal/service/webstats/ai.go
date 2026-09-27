// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import "strings"

// AIHosts are overseas AI referrer domains. Matching is a lowercase substring.
var AIHosts = []string{
	"chatgpt.com", "chat.openai.com", "perplexity.ai", "gemini.google.com",
	"copilot.microsoft.com", "claude.ai", "grok.com",
}

// AISource reports whether source or medium contains an AIHosts entry.
func AISource(source, medium string) bool {
	source = strings.ToLower(source)
	medium = strings.ToLower(medium)
	for _, host := range AIHosts {
		if strings.Contains(source, host) || strings.Contains(medium, host) {
			return true
		}
	}
	return false
}
