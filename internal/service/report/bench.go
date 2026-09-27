// SPDX-License-Identifier: AGPL-3.0-or-later

package report

// The tables below come from the CN-GEO citation dataset (187,818 deduplicated
// citations). They are observational and
// only shown when answers already cite some of these Chinese domains.

import "strings"

type CrossRow struct {
	Domain, Category, Ecosystem string
	National, Yours             int
}

type EcoGap struct {
	Domain, Why string
}

type PosHit struct {
	Domain   string
	Position float64
	Yours    int
}

type Bench struct {
	Covered, Missing []CrossRow
	CoverageRate     float64
	EcoGaps          []EcoGap
	HighPos          []PosHit
}

var crossPlatform = []struct {
	Domain, Category, Eco string
	National              int
}{
	{"qq.com", "Content platform", "Tencent", 11017},
	{"toutiao.com", "Content platform", "ByteDance", 9911},
	{"sohu.com", "News media", "Sohu", 8000},
	{"maigoo.com", "Rankings and recommendations", "Maigoo", 6741},
	{"smzdm.com", "Interest community", "SMZDM", 6547},
	{"163.com", "News media", "NetEase", 4777},
	{"chinapp.com", "Rankings and recommendations", "Chinapp", 3955},
	{"cnpp.cn", "Rankings and recommendations", "—", 3429},
	{"ctrip.com", "Local services", "Trip.com", 3273},
	{"sina.cn", "News media", "Sina", 3112},
	{"sina.com.cn", "News media", "Sina", 1951},
	{"dianping.com", "Local services", "Meituan", 1895},
	{"cnblogs.com", "Developer community", "Cnblogs", 1388},
	{"zol.com.cn", "Tech media", "ZOL", 987},
	{"36kr.com", "Business media", "—", 969},
}

var ecosystem = []EcoGap{
	{"baidu.com", "Baidu AI 37.7%, ERNIE 29.0%"},
	{"sm.cn", "Qwen 19.2% (Quark / Shenma)"},
	{"iesdouyin.com", "Doubao app 28.1%, Doubao web 12.0%, Douyin AI 100%"},
	{"toutiao.com", "secondary Doubao entry"},
	{"qq.com", "Yuanbao 20.5%"},
}

var topPosition = []struct {
	Domain string
	Pos    float64
}{
	{"sm.cn", 5.31}, {"cnpp.cn", 6.10}, {"xnnews.com.cn", 6.25}, {"askci.com", 6.27},
	{"maigoo.com", 6.36}, {"phb123.com", 7.12}, {"iimedia.cn", 7.27}, {"uc.cn", 7.32},
	{"cnpp100.com", 7.35}, {"csdn.net", 7.53},
}

func CompareCited(cited map[string]int) Bench {
	hit := func(base string) int {
		n := 0
		base = strings.ToLower(strings.TrimPrefix(base, "www."))
		for d, c := range cited {
			d = strings.ToLower(strings.TrimPrefix(d, "www."))
			if d == base || strings.HasSuffix(d, "."+base) {
				n += c
			}
		}
		return n
	}
	var b Bench
	for _, row := range crossPlatform {
		n := hit(row.Domain)
		item := CrossRow{Domain: row.Domain, Category: row.Category, Ecosystem: row.Eco, National: row.National, Yours: n}
		if n > 0 {
			b.Covered = append(b.Covered, item)
		} else {
			b.Missing = append(b.Missing, item)
		}
	}
	if len(crossPlatform) > 0 {
		b.CoverageRate = float64(int(float64(len(b.Covered))/float64(len(crossPlatform))*1000+0.5)) / 1000
	}
	for _, g := range ecosystem {
		if hit(g.Domain) == 0 {
			b.EcoGaps = append(b.EcoGaps, g)
		}
	}
	for _, p := range topPosition {
		if n := hit(p.Domain); n > 0 {
			b.HighPos = append(b.HighPos, PosHit{Domain: p.Domain, Position: p.Pos, Yours: n})
		}
	}
	return b
}
