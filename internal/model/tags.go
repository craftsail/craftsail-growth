// SPDX-License-Identifier: AGPL-3.0-or-later

package model

import "strings"

const (
	TagBranded   = "branded"
	TagUnbranded = "unbranded"
)

func ComputeSystemTags(text, brand string, aliases []string, site string) []string {
	if IsPromptBranded(text, brand, aliases, site) {
		return []string{TagBranded}
	}
	return []string{TagUnbranded}
}

// IsPromptBranded is the single definition of a branded prompt: the prompt
// text names the brand, one of its aliases, the owned host, or the host's
// first label (at least 3 characters).
func IsPromptBranded(text, brand string, aliases []string, site string) bool {
	q := strings.ToLower(text)
	for _, n := range append([]string{brand}, aliases...) {
		n = strings.ToLower(strings.TrimSpace(n))
		if n != "" && strings.Contains(q, n) {
			return true
		}
	}
	host := bareDomain(site)
	if host == "" {
		return false
	}
	if strings.Contains(q, host) {
		return true
	}
	root := strings.Split(host, ".")[0]
	return len(root) >= 3 && strings.Contains(q, root)
}

func bareDomain(site string) string {
	site = strings.ToLower(strings.TrimSpace(site))
	site = strings.TrimPrefix(site, "https://")
	site = strings.TrimPrefix(site, "http://")
	site = strings.TrimPrefix(site, "www.")
	if i := strings.IndexAny(site, "/:?"); i >= 0 {
		site = site[:i]
	}
	return site
}

func SanitizeTags(tags []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func EffectiveBranded(systemTags, userTags []string) bool {
	systemBranded := containsTag(systemTags, TagBranded)
	userBranded := containsTag(userTags, TagBranded)
	userUnbranded := containsTag(userTags, TagUnbranded)
	if userBranded && !userUnbranded {
		return true
	}
	if userUnbranded && !userBranded {
		return false
	}
	return systemBranded
}

func MatchTags(systemTags, userTags, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	wantsBranded := containsTag(filter, TagBranded)
	wantsUnbranded := containsTag(filter, TagUnbranded)
	if wantsBranded || wantsUnbranded {
		if EffectiveBranded(systemTags, userTags) {
			if wantsBranded {
				return true
			}
		} else if wantsUnbranded {
			return true
		}
	}
	all := append(append([]string{}, systemTags...), userTags...)
	for _, tag := range filter {
		if tag == TagBranded || tag == TagUnbranded {
			continue
		}
		if containsTag(all, tag) {
			return true
		}
	}
	return false
}

func containsTag(tags []string, want string) bool {
	want = strings.ToLower(want)
	for _, tag := range tags {
		if strings.ToLower(tag) == want {
			return true
		}
	}
	return false
}
