// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"sort"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type OppRef struct {
	QID    string `json:"qid,omitempty"`
	Text   string `json:"text,omitempty"`
	URL    string `json:"url,omitempty"`
	Domain string `json:"domain,omitempty"`
	Title  string `json:"title,omitempty"`
}

type Opportunity struct {
	Category   string   `json:"category"`
	Title      string   `json:"title"`
	Why        string   `json:"why"`
	QID        string   `json:"qid"`
	URLs       []string `json:"urls"`
	Difficulty string   `json:"difficulty"`
	Prompts    []OppRef `json:"prompts,omitempty"`
	Own        []OppRef `json:"own,omitempty"`
	Rivals     []OppRef `json:"rivals,omitempty"`
}

func BuildOpportunities(rows []model.SampleCitation, questions map[string]string, own string, day time.Time) []Opportunity {
	today := day.Format("2006-01-02")
	ownHost := bareHost(own)
	type acc struct {
		social, reviews, comps []string
		urls                   []string
		own                    bool
		days                   map[string]map[string]int
	}
	byQ := map[string]*acc{}
	var droppedOwn []string
	prevDay := ""
	for _, c := range rows {
		d := c.SampledOn.Format("2006-01-02")
		if d != today && (prevDay == "" || d > prevDay) {
			prevDay = d
		}
	}
	todayURLs := map[string]bool{}
	for _, c := range rows {
		if c.SampledOn.Format("2006-01-02") == today {
			todayURLs[c.URL] = true
		}
	}
	if prevDay != "" && ownHost != "" {
		for _, c := range rows {
			if c.SampledOn.Format("2006-01-02") != prevDay || todayURLs[c.URL] {
				continue
			}
			if hostMatch(bareHost(c.Domain), ownHost) || c.Category == "brand" {
				droppedOwn = append(droppedOwn, c.URL)
			}
		}
	}
	for _, c := range rows {
		if c.SampledOn.Format("2006-01-02") != today {
			continue
		}
		a := byQ[c.QID]
		if a == nil {
			a = &acc{days: map[string]map[string]int{}}
			byQ[c.QID] = a
		}
		switch c.Category {
		case "brand":
			a.own = true
		case "social":
			a.social = appendUnique(a.social, c.Domain)
			a.urls = appendUnique(a.urls, c.URL)
		case "reviews":
			a.reviews = appendUnique(a.reviews, c.Domain)
			a.urls = appendUnique(a.urls, c.URL)
		case "competitor":
			a.comps = appendUnique(a.comps, c.Domain)
			a.urls = appendUnique(a.urls, c.URL)
		}
	}
	for _, c := range rows {
		if c.Domain == "" {
			continue
		}
		a := byQ[c.QID]
		if a == nil {
			continue
		}
		d := c.SampledOn.Format("2006-01-02")
		if a.days[d] == nil {
			a.days[d] = map[string]int{}
		}
		a.days[d][c.Domain]++
	}
	var out []Opportunity
	for qid, a := range byQ {
		if a.own || (len(a.comps) == 0 && len(a.social) == 0 && len(a.reviews) == 0) {
			continue
		}
		q := questions[qid]
		if q == "" {
			q = qid
		}
		op := Opportunity{QID: qid, URLs: capList(a.urls, 3), Difficulty: difficultyOf(a.days), Prompts: []OppRef{{QID: qid, Text: q}}}
		switch {
		case len(a.social) > 0:
			op.Category = "social"
			op.Title = "Show up in the discussion: " + trimTitle(q)
			op.Why = "Answers to this buyer prompt cite " + joinDomains(a.social) + ". Contribute verifiable statements there; no fake reviews or sock puppets."
		case len(a.reviews) > 0:
			op.Category = "outreach"
			op.Title = "Get into the reviews: " + trimTitle(q)
			op.Why = "Review sites " + joinDomains(a.reviews) + " already feed this answer. The brand domain is not cited."
		default:
			op.Category = "creation"
			op.Title = "Write a comparison: " + trimTitle(q)
			op.Why = "Competitor domains " + joinDomains(a.comps) + " are cited and yours are not. Write a comparison or ranking; do not edit encyclopedia entries yourself."
		}
		if op.Difficulty == "locked-in" {
			op.Why += " The cited sources for this prompt are stable, so this is not a quick win."
		}
		out = append(out, op)
	}
	if len(droppedOwn) > 0 {
		out = append(out, Opportunity{
			Category: "existing-content", Title: "Recover owned pages that stopped being cited",
			Why:        "These owned URLs were cited last period and not this one. Check they still exist and can be crawled before rewriting them.",
			URLs:       capList(droppedOwn, 3),
			Difficulty: "n/a",
		})
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := difficultyRank(out[i].Difficulty), difficultyRank(out[j].Difficulty)
		if di != dj {
			return di < dj
		}
		return out[i].Title < out[j].Title
	})
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func difficultyOf(days map[string]map[string]int) string {
	var in []DayDomains
	for d, counts := range days {
		in = append(in, DayDomains{Date: d, Counts: counts})
	}
	_, label := Stability(in)
	return label
}

func difficultyRank(label string) int {
	switch label {
	case "wide-open":
		return 0
	case "contested":
		return 1
	case "n/a":
		return 2
	default:
		return 3
	}
}

func appendUnique(ss []string, v string) []string {
	if v == "" {
		return ss
	}
	for _, s := range ss {
		if s == v {
			return ss
		}
	}
	return append(ss, v)
}

func capList(ss []string, n int) []string {
	if len(ss) > n {
		return ss[:n]
	}
	return ss
}

func joinDomains(ss []string) string {
	if len(ss) == 0 {
		return "third-party pages"
	}
	return strings.Join(ss, "、")
}

func trimTitle(q string) string {
	q = strings.TrimSpace(q)
	r := []rune(q)
	if len(r) > 28 {
		return string(r[:28]) + "…"
	}
	return q
}
