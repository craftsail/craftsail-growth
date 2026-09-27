// SPDX-License-Identifier: AGPL-3.0-or-later

package generate

import (
	"regexp"
	"strings"
	"unicode"
)

type Facts struct {
	Definition string
	Numbers    []Num
	Suitable   []string
	Unsuitable []string
	Raw        string
}

type Num struct {
	Fact, Value, Source string
}

func ParseFacts(md string) Facts {
	out := Facts{Raw: md}
	if sec := firstSection(md, "One-line definition", "一句话定义"); sec != "" {
		var quoted []string
		for _, ln := range strings.Split(sec, "\n") {
			s := strings.TrimSpace(ln)
			if strings.HasPrefix(s, ">") {
				quoted = append(quoted, strings.TrimSpace(strings.TrimPrefix(s, ">")))
			}
		}
		line := ""
		if len(quoted) > 0 {
			line = strings.Join(quoted, " ")
		} else {
			for _, ln := range strings.Split(sec, "\n") {
				s := strings.TrimSpace(ln)
				if s != "" && !strings.HasPrefix(s, "#") && !strings.HasPrefix(s, "-") && !strings.HasPrefix(s, "|") {
					line = s
					break
				}
			}
		}
		line = regexp.MustCompile(`\*\*(.+?)\*\*`).ReplaceAllString(line, "$1")
		line = regexp.MustCompile("`(.+?)`").ReplaceAllString(line, "$1")
		line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
		out.Definition = stripCJKSpaces(strings.TrimSpace(line))
	}
	if sec := firstSection(md, "Key numbers", "关键数字"); sec != "" {
		row := regexp.MustCompile(`(?m)^\s*\|([^|\n]+)\|([^|\n]+)\|([^|\n]+)\|`)
		for _, m := range row.FindAllStringSubmatch(sec, -1) {
			a, b, c := strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
			if a == "" || a == "Fact" || a == "Item" || a == "事实" || a == "---" || a == "项" || isSep(a) {
				continue
			}
			out.Numbers = append(out.Numbers, Num{Fact: a, Value: b, Source: c})
		}
	}
	out.Suitable = bulletsLabeled(md, "**Good fit", "**Not a fit")
	if out.Suitable == nil {
		out.Suitable = bulletsLabeled(md, "**适合", "**不适合")
	}
	out.Unsuitable = bulletsLabeled(md, "**Not a fit", "")
	if out.Unsuitable == nil {
		out.Unsuitable = bulletsLabeled(md, "**不适合", "")
	}
	return out
}

// bulletsLabeled returns the "- " bullets after label, stopping at the next
// "##" heading or at stop when it is set.
func bulletsLabeled(md, label, stop string) []string {
	i := strings.Index(md, label)
	if i < 0 {
		return nil
	}
	rest := md[i:]
	if j := strings.Index(rest, "\n"); j >= 0 {
		rest = rest[j+1:]
	}
	if k := strings.Index(rest, "\n##"); k >= 0 {
		rest = rest[:k]
	}
	if stop != "" {
		if k := strings.Index(rest, stop); k >= 0 {
			rest = rest[:k]
		}
	}
	var out []string
	for _, ln := range strings.Split(rest, "\n") {
		s := strings.TrimSpace(ln)
		if strings.HasPrefix(s, "-") {
			out = append(out, strings.TrimSpace(strings.TrimPrefix(s, "-")))
		}
	}
	return out
}

// firstSection accepts the current English heading and the legacy one.
func firstSection(md string, titles ...string) string {
	for _, t := range titles {
		if sec := section(md, t); sec != "" {
			return sec
		}
	}
	return ""
}

func section(md, title string) string {
	re := regexp.MustCompile(`(?s)##\s*` + regexp.QuoteMeta(title) + `.*?\n(.*?)(?:\n##|\z)`)
	m := re.FindStringSubmatch(md)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func isSep(s string) bool {
	for _, r := range s {
		if r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

func stripCJKSpaces(s string) string {
	rs := []rune(s)
	var out []rune
	for i, r := range rs {
		if r == ' ' && i > 0 && i+1 < len(rs) && isCJK(rs[i-1]) && isCJK(rs[i+1]) {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func isCJK(r rune) bool {
	return unicode.In(r, unicode.Han) || (r >= 0x3000 && r <= 0x303F) || (r >= 0xFF00 && r <= 0xFFEF)
}
