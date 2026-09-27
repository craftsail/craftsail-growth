// SPDX-License-Identifier: AGPL-3.0-or-later

package mdhtml

import (
	"regexp"
	"strings"
)

var (
	reComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	reH       = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	reLI      = regexp.MustCompile(`^\s*[-*]\s+(.*)$`)
	reNLI     = regexp.MustCompile(`^\s*\d+[.、]\s+(.*)$`)
	reBold    = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reLink    = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reCode    = regexp.MustCompile("`([^`]+)`")
	reSep     = regexp.MustCompile(`^\|[\s:|-]+\|$`)
)

// Document keeps heading levels (Python report.md_to_html) and renders tables.
func Document(md string) string {
	return convert(md, 0)
}

func convert(md string, demote int) string {
	md = reComment.ReplaceAllString(md, "")
	lines := strings.Split(md, "\n")
	var out []string
	inCode, inList := false, false
	inline := func(s string) string {
		s = strings.ReplaceAll(s, "&", "&amp;")
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
		s = reBold.ReplaceAllString(s, "<strong>$1</strong>")
		s = reLink.ReplaceAllString(s, `<a href="$2">$1</a>`)
		s = reCode.ReplaceAllString(s, "<code>$1</code>")
		return s
	}
	closeList := func() {
		if inList {
			out = append(out, "</ul>")
			inList = false
		}
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			closeList()
			if inCode {
				out = append(out, "</code></pre>")
			} else {
				out = append(out, "<pre><code>")
			}
			inCode = !inCode
			continue
		}
		if inCode {
			out = append(out, strings.ReplaceAll(strings.ReplaceAll(line, "&", "&amp;"), "<", "&lt;"))
			continue
		}
		if strings.HasPrefix(line, "|") && i+1 < len(lines) && reSep.MatchString(strings.TrimSpace(lines[i+1])) {
			closeList()
			head := splitRow(line)
			i += 2
			var rows [][]string
			for i < len(lines) && strings.HasPrefix(lines[i], "|") {
				rows = append(rows, splitRow(lines[i]))
				i++
			}
			i--
			var b strings.Builder
			b.WriteString(`<div class="scroll"><table><thead><tr>`)
			for _, h := range head {
				b.WriteString("<th>" + inline(h) + "</th>")
			}
			b.WriteString("</tr></thead><tbody>")
			for _, r := range rows {
				b.WriteString("<tr>")
				for _, c := range r {
					b.WriteString("<td>" + inline(c) + "</td>")
				}
				b.WriteString("</tr>")
			}
			b.WriteString("</tbody></table></div>")
			out = append(out, b.String())
			continue
		}
		if strings.TrimSpace(line) == "---" {
			closeList()
			out = append(out, "<hr>")
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			closeList()
			out = append(out, "<blockquote>"+inline(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), ">")))+"</blockquote>")
			continue
		}
		li := reLI.FindStringSubmatch(line)
		if li == nil {
			li = reNLI.FindStringSubmatch(line)
		}
		if inList && li == nil {
			closeList()
		}
		if m := reH.FindStringSubmatch(line); m != nil {
			closeList()
			n := len(m[1]) + demote
			if n > 6 {
				n = 6
			}
			if n < 1 {
				n = 1
			}
			out = append(out, sprintfH(n, inline(m[2])))
		} else if li != nil {
			if !inList {
				out = append(out, "<ul>")
				inList = true
			}
			out = append(out, "<li>"+inline(li[1])+"</li>")
		} else if strings.TrimSpace(line) != "" {
			out = append(out, "<p>"+inline(strings.TrimSpace(line))+"</p>")
		}
	}
	closeList()
	if inCode {
		out = append(out, "</code></pre>")
	}
	return strings.Join(out, "\n")
}

func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func sprintfH(n int, inner string) string {
	tag := string(rune('0' + n))
	return "<h" + tag + ">" + inner + "</h" + tag + ">"
}
