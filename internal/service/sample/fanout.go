// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"strings"
	"time"
	"unicode"
)

type WordStat struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

type QuerySnap struct {
	Queries     []WordStat `json:"queries"`
	Unavailable int        `json:"unavailable"`
	Added       []WordStat `json:"added"`
	Dropped     []WordStat `json:"dropped"`
	Preserved   []WordStat `json:"preserved"`
}

func (s *Service) QuerySnap(ctx context.Context, projectID uint64, day time.Time) QuerySnap {
	return s.querySnap(ctx, projectID, day)
}

func (s *Service) querySnap(ctx context.Context, projectID uint64, day time.Time) QuerySnap {
	rows, err := s.samples.List(ctx, projectID, day, "", "", 2000)
	if err != nil {
		return QuerySnap{}
	}
	var in []struct {
		Question string
		Queries  []string
	}
	for _, sm := range rows {
		in = append(in, struct {
			Question string
			Queries  []string
		}{sm.QuestionText, sm.WebQueries})
	}
	return SummarizeQueries(in)
}

func SummarizeQueries(samples []struct {
	Question string
	Queries  []string
}) QuerySnap {
	qCount := map[string]int{}
	added := map[string]int{}
	dropped := map[string]int{}
	preserved := map[string]int{}
	unavailable := 0
	for _, sm := range samples {
		prompt := tokenSet(sm.Question)
		for _, q := range sm.Queries {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			if q == WebQueriesUnavailable {
				unavailable++
				continue
			}
			if strings.EqualFold(q, strings.TrimSpace(sm.Question)) {
				continue
			}
			qCount[q]++
			seen := tokenSet(q)
			for tok := range seen {
				if prompt[tok] {
					preserved[tok]++
				} else {
					added[tok]++
				}
			}
			for tok := range prompt {
				if !seen[tok] {
					dropped[tok]++
				}
			}
		}
	}
	return QuerySnap{
		Queries: topWords(qCount, 12), Unavailable: unavailable,
		Added: topWords(added, 8), Dropped: topWords(dropped, 8), Preserved: topWords(preserved, 8),
	}
}

func tokenSet(s string) map[string]bool {
	out := map[string]bool{}
	var latin []rune
	flush := func() {
		if len(latin) >= 2 {
			out[strings.ToLower(string(latin))] = true
		}
		latin = latin[:0]
	}
	var cjk []rune
	flushCJK := func() {
		if len(cjk) >= 2 {
			out[string(cjk)] = true
		}
		cjk = cjk[:0]
	}
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			flushCJK()
			latin = append(latin, r)
		case unicode.Is(unicode.Han, r):
			flush()
			cjk = append(cjk, r)
		default:
			flush()
			flushCJK()
		}
	}
	flush()
	flushCJK()
	return out
}

func topWords(m map[string]int, n int) []WordStat {
	type kv struct {
		k string
		v int
	}
	var ks []kv
	for k, v := range m {
		ks = append(ks, kv{k, v})
	}
	for i := 0; i < len(ks); i++ {
		for j := i + 1; j < len(ks); j++ {
			if ks[j].v > ks[i].v || (ks[j].v == ks[i].v && ks[j].k < ks[i].k) {
				ks[i], ks[j] = ks[j], ks[i]
			}
		}
	}
	if len(ks) > n {
		ks = ks[:n]
	}
	var out []WordStat
	for _, x := range ks {
		out = append(out, WordStat{Word: x.k, Count: x.v})
	}
	return out
}
