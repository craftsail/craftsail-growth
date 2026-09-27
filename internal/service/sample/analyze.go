// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"github.com/craftsail/craftsail-growth/internal/model"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type Comp struct {
	Name    string
	Site    string
	Aliases []string
}

type Cfg struct {
	BrandName   string
	Aliases     []string
	Site        string
	Competitors []Comp
}

type Analysis struct {
	BrandMentioned       bool     `json:"brand_mentioned"`
	BrandRank            int      `json:"brand_rank"`
	Candidates           []string `json:"candidates"`
	CompetitorsMentioned []string `json:"competitors_mentioned"`
	CitedDomains         []string `json:"cited_domains"`
	OwnDomainCited       bool     `json:"own_domain_cited"`
	AnswerChars          int      `json:"answer_chars"`
	NeedsReview          bool     `json:"needs_review"`
	NegativeCues         []string `json:"negative_cues"`
}

var (
	urlRE   = regexp.MustCompile(`https?://[^\s\)\]"'，。；]+`)
	negRE   = regexp.MustCompile(`(?i)不是|并非|不属于|不同于|not |isn't|aren't`)
	negCues = regexp.MustCompile(`(?i)不推荐|避雷|缺点|劣势|投诉|差评|跑路|骗局|割韭菜|不靠谱|慎用|翻车|已倒闭|停止运营|维权|退款难|not recommended|avoid|scam|complaints?|lawsuit|shut ?down|worse than|downsides?`)
	sentEnd = []rune("。！？!?\n")
)

func AnalyzeAnswer(answer string, cfg Cfg, citations []Citation) Analysis {
	names, alias := entitiesOf(cfg)
	positions := map[string]int{}
	needs := false
	for _, n := range names {
		pos, negated := entityHit(answer, alias[n])
		positions[n] = pos
		needs = needs || negated
	}
	var ordered []string
	type pair struct {
		n string
		p int
	}
	var ps []pair
	for n, p := range positions {
		if p >= 0 {
			ps = append(ps, pair{n, p})
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].p < ps[j].p })
	for _, x := range ps {
		ordered = append(ordered, x.n)
	}
	present := map[string]bool{}
	for n, p := range positions {
		present[n] = p >= 0
	}

	var urls []string
	urls = append(urls, urlRE.FindAllString(answer, -1)...)
	for _, c := range citations {
		if c.URL != "" {
			urls = append(urls, c.URL)
		}
	}
	var domains []string
	seenD := map[string]bool{}
	for _, u := range urls {
		h := hostOf(u)
		if h != "" && !seenD[h] {
			seenD[h] = true
			domains = append(domains, h)
		}
	}
	sort.Strings(domains)
	own := ""
	if strings.TrimSpace(cfg.Site) != "" {
		own = hostOf(cfg.Site)
	}
	ownCited := false
	if own != "" {
		for _, d := range domains {
			if d == own || strings.HasSuffix(d, "."+own) {
				ownCited = true
			}
		}
	}

	neg := map[string]bool{}
	if present[cfg.BrandName] {
		for _, a := range alias[cfg.BrandName] {
			for _, sp := range aliasSpans(answer, a) {
				lo, hi := sp[0]-80, sp[1]+160
				if lo < 0 {
					lo = 0
				}
				if hi > len(answer) {
					hi = len(answer)
				}
				for _, m := range negCues.FindAllString(answer[lo:hi], -1) {
					neg[strings.ToLower(m)] = true
				}
			}
		}
	}
	var cues []string
	for k := range neg {
		cues = append(cues, k)
	}
	sort.Strings(cues)

	rank := 0
	for i, n := range ordered {
		if n == cfg.BrandName {
			rank = i + 1
			break
		}
	}
	var comps []string
	for _, n := range names {
		if n != cfg.BrandName && present[n] {
			comps = append(comps, n)
		}
	}
	if ordered == nil {
		ordered = []string{}
	}
	if comps == nil {
		comps = []string{}
	}
	return Analysis{
		BrandMentioned: present[cfg.BrandName], BrandRank: rank, Candidates: ordered,
		CompetitorsMentioned: comps, CitedDomains: domains, OwnDomainCited: ownCited,
		AnswerChars: len(answer), NeedsReview: needs || len(cues) > 0, NegativeCues: cues,
	}
}

// BrandInQuestion delegates to the single definition in the model package.
func BrandInQuestion(question string, cfg Cfg) bool {
	return model.IsPromptBranded(question, cfg.BrandName, cfg.Aliases, cfg.Site)
}

func entitiesOf(cfg Cfg) ([]string, map[string][]string) {
	alias := map[string][]string{}
	var names []string
	add := func(name string, extra []string) {
		if name == "" {
			return
		}
		names = append(names, name)
		alias[name] = append([]string{name}, extra...)
	}
	add(cfg.BrandName, cfg.Aliases)
	for _, c := range cfg.Competitors {
		add(c.Name, c.Aliases)
	}
	return names, alias
}

func entityHit(text string, aliases []string) (int, bool) {
	type sp struct{ s, e int }
	var hits []sp
	for _, a := range aliases {
		if a == "" {
			continue
		}
		for _, x := range aliasSpans(text, a) {
			hits = append(hits, sp{x[0], x[1]})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].s < hits[j].s })
	var valid []int
	negated := false
	for _, h := range hits {
		if negRE.MatchString(sentenceAt(text, h.s)) {
			negated = true
		} else {
			valid = append(valid, h.s)
		}
	}
	if len(valid) == 0 {
		return -1, negated
	}
	m := valid[0]
	for _, v := range valid[1:] {
		if v < m {
			m = v
		}
	}
	return m, negated
}

func aliasSpans(text, alias string) [][2]int {
	if alias == "" {
		return nil
	}
	low := strings.ToLower(text)
	al := strings.ToLower(alias)
	leftNeed := isLatinDigit(firstRune(alias))
	rightNeed := isLatinDigit(lastRune(alias))
	var out [][2]int
	for i := 0; i <= len(low)-len(al); {
		j := strings.Index(low[i:], al)
		if j < 0 {
			break
		}
		start := i + j
		end := start + len(al)
		if leftNeed && start > 0 && byteLatinDigit(text, start-1) {
			i = start + 1
			continue
		}
		if rightNeed && end < len(text) && byteLatinDigit(text, end) {
			i = start + 1
			continue
		}
		out = append(out, [2]int{start, end})
		i = end
	}
	return out
}

func sentenceAt(text string, pos int) string {
	if pos < 0 {
		pos = 0
	}
	if pos > len(text) {
		pos = len(text)
	}
	start := 0
	for i, r := range text {
		if i >= pos {
			break
		}
		for _, e := range sentEnd {
			if r == e {
				start = i + len(string(r))
			}
		}
	}
	end := len(text)
	for i := pos; i < len(text); {
		r, sz := decodeRune(text[i:])
		for _, e := range sentEnd {
			if r == e {
				return text[start : i+sz]
			}
		}
		i += sz
	}
	return text[start:end]
}

func firstRune(s string) rune {
	for _, r := range s {
		return r
	}
	return 0
}

func lastRune(s string) rune {
	var r rune
	for _, x := range s {
		r = x
	}
	return r
}

func isLatinDigit(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || unicode.IsDigit(r)
}

func byteLatinDigit(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	b := s[i]
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}

func decodeRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 0
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Host), "www.")
}

type Citation struct {
	URL, Title string
}
