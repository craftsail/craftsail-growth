// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"github.com/craftsail/craftsail-growth/internal/service/metrics"
	"math"
	"sort"
	"strings"
)

type Row struct {
	PromptRevision  string
	Platform        string
	QuestionID      string
	Round           int
	SampleMode      string
	Question        string
	Market          string
	Day             string // YYYY-MM-DD; part of the dedup key
	OK              bool
	BrandInQuestion bool
	Analysis        Analysis
}

type ProbeStats struct {
	Samples           int      `json:"samples"`
	RecognizedRate    *float64 `json:"recognized_rate"`
	OwnDomainCiteRate *float64 `json:"own_domain_cite_rate"`
}

type ModeStats struct {
	Samples            int               `json:"samples"`
	MentionRate        *float64          `json:"mention_rate"`
	MentionCI          *metrics.Interval `json:"mention_ci"`
	MentionN           int               `json:"mention_n"`
	Top1Rate           *float64          `json:"top1_rate"`
	Top3Rate           *float64          `json:"top3_rate"`
	AvgRank            *float64          `json:"avg_rank"`
	OwnDomainCiteRate  *float64          `json:"own_domain_cite_rate"`
	ShareOfVoice       *float64          `json:"share_of_voice"`
	CompetitorMentions map[string]int    `json:"competitor_mentions"`
	TopCitedDomains    map[string]int    `json:"top_cited_domains"`
	Probe              ProbeStats        `json:"probe"`
}

type PlatStats struct {
	ModeStats
	Market string               `json:"market"`
	Label  string               `json:"label"`
	ByMode map[string]ModeStats `json:"by_mode,omitempty"`
}

func DedupRows(rows []Row) []Row {
	seen := map[string]Row{}
	var order []string
	for _, r := range rows {
		k := r.Day + "|" + r.Platform + "|" + r.QuestionID + "|" + itoa(r.Round) + "|" + r.SampleMode + "|" + r.PromptRevision
		if _, ok := seen[k]; !ok {
			order = append(order, k)
		}
		seen[k] = r
	}
	out := make([]Row, 0, len(order))
	for _, k := range order {
		out = append(out, seen[k])
	}
	return out
}

func Aggregate(rows []Row, cfg Cfg) map[string]PlatStats {
	by := map[string][]Row{}
	for _, r := range rows {
		by[r.Platform] = append(by[r.Platform], r)
	}
	out := map[string]PlatStats{}
	for plat, all := range by {
		modes := map[string][]Row{}
		for _, r := range all {
			mode := r.SampleMode
			if mode == "" {
				mode = "api"
			}
			modes[mode] = append(modes[mode], r)
		}
		byMode := map[string]ModeStats{}
		for mode, rs := range modes {
			byMode[mode] = modeStats(rs, cfg, plat)
		}
		st := PlatStats{Market: MarketOf(plat), Label: LabelOf(plat), ByMode: byMode}
		if len(byMode) == 1 {
			for _, ms := range byMode {
				st.ModeStats = ms
				if ms.Samples > 0 && all[0].Market != "" {
					st.Market = all[0].Market
				}
			}
		}
		out[plat] = st
	}
	return out
}

func modeStats(all []Row, cfg Cfg, plat string) ModeStats {
	obs := make([]metrics.Obs, 0, len(all))
	dom := map[string]int{}
	for _, r := range all {
		branded := r.BrandInQuestion || BrandInQuestion(r.Question, cfg)
		obs = append(obs, metrics.Obs{
			QID: r.QuestionID, Platform: plat, Access: metrics.AccessOf(r.SampleMode), Day: r.Day,
			OK: true, Branded: branded, Mentioned: r.Analysis.BrandMentioned, Rank: r.Analysis.BrandRank,
			OwnCited: r.Analysis.OwnDomainCited, Competitors: r.Analysis.CompetitorsMentioned,
		})
		if !branded {
			for _, d := range r.Analysis.CitedDomains {
				dom[d]++
			}
		}
	}
	access := "api"
	if len(all) > 0 {
		access = metrics.AccessOf(all[0].SampleMode)
	}
	s := metrics.Compute(obs, metrics.Filter{Access: access})
	round := func(r metrics.Rate) *float64 {
		if r.Value == nil {
			return nil
		}
		return pRound(*r.Value, 3)
	}
	out := ModeStats{
		Samples: s.Visibility.N, MentionRate: round(s.Visibility), MentionCI: s.Visibility.CI, MentionN: s.Visibility.N,
		Top1Rate: round(s.Top1), Top3Rate: round(s.Top3),
		ShareOfVoice: s.ShareOfVoice, CompetitorMentions: s.Competitors, TopCitedDomains: topN(dom, 15),
		Probe: ProbeStats{Samples: s.Recognition.N, RecognizedRate: round(s.Recognition)},
	}
	if s.AvgRank != nil {
		out.AvgRank = pRound(*s.AvgRank, 2)
	}
	if stringsTrim(cfg.Site) != "" {
		out.OwnDomainCiteRate = round(s.OwnCited)
		// probe own-cite keeps its meaning: branded runs citing the owned domain
		c, n := 0, 0
		for _, o := range obs {
			if o.Branded {
				n++
				if o.OwnCited {
					c++
				}
			}
		}
		if n > 0 {
			out.Probe.OwnDomainCiteRate = pRound(float64(c)/float64(n), 3)
		}
	}
	return out
}

func pRound(v float64, places int) *float64 {
	p := math.Pow(10, float64(places))
	x := math.Round(v*p) / p
	return &x
}

func stringsTrim(s string) string { return strings.TrimSpace(s) }

func topN(m map[string]int, n int) map[string]int {
	type kv struct {
		k string
		v int
	}
	var ks []kv
	for k, v := range m {
		ks = append(ks, kv{k, v})
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i].v > ks[j].v })
	if len(ks) > n {
		ks = ks[:n]
	}
	out := map[string]int{}
	for _, x := range ks {
		out[x.k] = x.v
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	if neg {
		return "-" + string(d)
	}
	return string(d)
}
