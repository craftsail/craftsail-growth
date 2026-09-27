// SPDX-License-Identifier: AGPL-3.0-or-later

package metrics

// LowSampleN is a heuristic: below this many runs the dashboard marks the
// rate as a small sample. It is not a statistical threshold.
const LowSampleN = 30

// DefaultWindowDays matches the dashboard's default range so that reports,
// plans and verification read the same numbers.
const DefaultWindowDays = 30

// Obs is one engine answer reduced to what the formulas need.
type Obs struct {
	QID         string
	Platform    string
	Access      string // api | web
	Day         string // YYYY-MM-DD
	OK          bool
	Branded     bool
	Mentioned   bool
	Rank        int // 1-based order of first appearance among brand and competitors; 0 = absent
	OwnCited    bool
	Competitors []string
}

type Filter struct {
	Access   string // "" picks the access with the most successful unbranded runs
	Platform string // "" = all engines
}

type Rate struct {
	X         int       `json:"x"`
	N         int       `json:"n"`
	Value     *float64  `json:"value"`
	CI        *Interval `json:"ci"`
	LowSample bool      `json:"low_sample"`
}

func NewRate(x, n int) Rate {
	r := Rate{X: x, N: n, LowSample: n < LowSampleN}
	if n > 0 {
		v := float64(x) / float64(n)
		r.Value = &v
		r.CI = Wilson(x, n)
	}
	return r
}

type Snapshot struct {
	Access           string         `json:"access"`
	Runs             int            `json:"runs"`
	Failed           int            `json:"failed"`
	Visibility       Rate           `json:"visibility"`
	Recognition      Rate           `json:"recognition"`
	Top1             Rate           `json:"top1"`
	Top3             Rate           `json:"top3"`
	OwnCited         Rate           `json:"own_cited"`
	AvgRank          *float64       `json:"avg_rank"`
	ShareOfVoice     *float64       `json:"share_of_voice"`
	BrandEvents      int            `json:"brand_events"`
	CompetitorEvents int            `json:"competitor_events"`
	Competitors      map[string]int `json:"competitors"`
}

// AccessOf maps a stored sample_mode to an access group. Anything that is
// not a direct API call is a web/app observation.
func AccessOf(sampleMode string) string {
	if sampleMode == "" || sampleMode == "api" {
		return "api"
	}
	return "web"
}

// DefaultAccess returns the access group with the most successful unbranded
// runs. Ties go to api.
func DefaultAccess(rows []Obs, platform string) string {
	n := map[string]int{}
	for _, r := range rows {
		if r.OK && !r.Branded && (platform == "" || r.Platform == platform) {
			n[r.Access]++
		}
	}
	if n["web"] > n["api"] {
		return "web"
	}
	return "api"
}

// Accesses lists access groups that have at least one row.
func Accesses(rows []Obs) []string {
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Access] = true
	}
	out := []string{}
	for _, a := range []string{"api", "web"} {
		if seen[a] {
			out = append(out, a)
		}
	}
	return out
}

// Compute never mixes access groups: with an empty Filter.Access it picks
// DefaultAccess and reports which one it used.
func Compute(rows []Obs, f Filter) Snapshot {
	access := f.Access
	if access == "" {
		access = DefaultAccess(rows, f.Platform)
	}
	s := Snapshot{Access: access, Competitors: map[string]int{}}
	vis, visN, rec, recN, t1, t3, own := 0, 0, 0, 0, 0, 0, 0
	rankSum, rankN := 0, 0
	for _, r := range rows {
		if r.Access != access || (f.Platform != "" && r.Platform != f.Platform) {
			continue
		}
		if !r.OK {
			s.Failed++
			continue
		}
		s.Runs++
		if r.Branded {
			recN++
			if r.Mentioned {
				rec++
			}
			continue
		}
		visN++
		if r.OwnCited {
			own++
		}
		if r.Mentioned {
			vis++
			s.BrandEvents++
			if r.Rank == 1 {
				t1++
			}
			if r.Rank >= 1 && r.Rank <= 3 {
				t3++
			}
			if r.Rank > 0 {
				rankSum += r.Rank
				rankN++
			}
		}
		for _, c := range r.Competitors {
			if c == "" {
				continue
			}
			s.CompetitorEvents++
			s.Competitors[c]++
		}
	}
	s.Visibility = NewRate(vis, visN)
	s.Recognition = NewRate(rec, recN)
	s.Top1 = NewRate(t1, visN)
	s.Top3 = NewRate(t3, visN)
	s.OwnCited = NewRate(own, visN)
	if rankN > 0 {
		v := float64(rankSum) / float64(rankN)
		s.AvgRank = &v
	}
	if total := s.BrandEvents + s.CompetitorEvents; total > 0 {
		v := float64(s.BrandEvents) / float64(total)
		s.ShareOfVoice = &v
	}
	return s
}
