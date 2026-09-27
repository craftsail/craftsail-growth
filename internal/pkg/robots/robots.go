// SPDX-License-Identifier: AGPL-3.0-or-later

package robots

import (
	"regexp"
	"strings"
)

type Group struct {
	Agents []string
	Rules  []Rule
}

type Rule struct {
	Allow bool
	Path  string
}

func Parse(txt string) []Group {
	var groups []Group
	idx := -1
	lastWasAgent := false
	for _, raw := range strings.Split(txt, "\n") {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		if line == "" || !strings.Contains(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		field, value = strings.ToLower(strings.TrimSpace(field)), strings.TrimSpace(value)
		switch field {
		case "user-agent":
			if idx < 0 || !lastWasAgent {
				groups = append(groups, Group{})
				idx = len(groups) - 1
			}
			groups[idx].Agents = append(groups[idx].Agents, strings.ToLower(value))
			lastWasAgent = true
		case "allow", "disallow":
			lastWasAgent = false
			if idx >= 0 && value != "" {
				groups[idx].Rules = append(groups[idx].Rules, Rule{Allow: field == "allow", Path: value})
			}
		default:
			lastWasAgent = false
		}
	}
	return groups
}

func Decision(groups []Group, ua, path string) (bool, string) {
	uaL := strings.ToLower(ua)
	var specific *Group
	specLen := -1
	var wildcard *Group
	for i := range groups {
		g := &groups[i]
		for _, a := range g.Agents {
			if a == "*" {
				if wildcard == nil {
					wildcard = g
				}
			} else if a != "" && (strings.Contains(uaL, a) || strings.Contains(a, uaL)) && len(a) > specLen {
				specific, specLen = g, len(a)
			}
		}
	}
	g := specific
	if g == nil {
		g = wildcard
	}
	if g == nil {
		return true, ""
	}
	if path == "" {
		path = "/"
	}
	matchLen, allowed, rule := -1, true, ""
	for _, r := range g.Rules {
		if !ruleRX(r.Path).MatchString(path) {
			continue
		}
		plen := len(r.Path)
		if plen > matchLen || (plen == matchLen && r.Allow && !allowed) {
			matchLen, allowed = plen, r.Allow
			prefix := "Disallow: "
			if r.Allow {
				prefix = "Allow: "
			}
			rule = prefix + r.Path
		}
	}
	return allowed, rule
}

func ruleRX(pattern string) *regexp.Regexp {
	rx := regexp.QuoteMeta(pattern)
	rx = strings.ReplaceAll(rx, `\*`, ".*")
	if strings.HasSuffix(rx, `\$`) {
		rx = rx[:len(rx)-2] + "$"
	}
	return regexp.MustCompile("^" + rx)
}
