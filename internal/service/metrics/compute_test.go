// SPDX-License-Identifier: AGPL-3.0-or-later

package metrics

import "testing"

func obs(qid, access string, branded, mentioned bool, rank int, comps ...string) Obs {
	return Obs{QID: qid, Platform: "deepseek", Access: access, Day: "2026-09-01", OK: true,
		Branded: branded, Mentioned: mentioned, Rank: rank, Competitors: comps}
}

func TestComputeExcludesBrandedFromVisibility(t *testing.T) {
	rows := []Obs{
		obs("q1", "api", false, true, 1, "Adidas"),
		obs("q2", "api", false, false, 0, "Adidas", "Puma"),
		obs("q3", "api", true, true, 1),
	}
	s := Compute(rows, Filter{})
	if s.Visibility.N != 2 || s.Visibility.X != 1 {
		t.Fatalf("visibility = %+v", s.Visibility)
	}
	if s.Recognition.N != 1 || s.Recognition.X != 1 {
		t.Fatalf("recognition = %+v", s.Recognition)
	}
	if s.Visibility.CI == nil || s.Visibility.Value == nil || *s.Visibility.Value != 0.5 {
		t.Fatalf("value/ci = %+v", s.Visibility)
	}
	if !s.Visibility.LowSample {
		t.Fatal("n=2 must be flagged as low sample")
	}
}

func TestComputeNeverMixesAccess(t *testing.T) {
	rows := []Obs{
		obs("q1", "api", false, true, 1),
		obs("q2", "api", false, true, 1),
		obs("q1", "web", false, false, 0),
	}
	s := Compute(rows, Filter{})
	if s.Access != "api" || s.Visibility.N != 2 {
		t.Fatalf("default access should be the larger group, got %s n=%d", s.Access, s.Visibility.N)
	}
	w := Compute(rows, Filter{Access: "web"})
	if w.Access != "web" || w.Visibility.N != 1 || w.Visibility.X != 0 {
		t.Fatalf("web = %+v", w)
	}
}

func TestComputeShareOfVoiceCountsEachCompetitor(t *testing.T) {
	rows := []Obs{
		obs("q1", "api", false, true, 2, "Adidas"),
		obs("q2", "api", false, false, 0, "Adidas", "Puma"),
	}
	s := Compute(rows, Filter{})
	if s.BrandEvents != 1 || s.CompetitorEvents != 3 {
		t.Fatalf("events brand=%d comp=%d", s.BrandEvents, s.CompetitorEvents)
	}
	if s.ShareOfVoice == nil || *s.ShareOfVoice != 0.25 {
		t.Fatalf("sov = %v", s.ShareOfVoice)
	}
	if s.Top1.X != 0 || s.Top3.X != 1 {
		t.Fatalf("top1=%+v top3=%+v", s.Top1, s.Top3)
	}
}

func TestComputeEmptyIsNotZero(t *testing.T) {
	s := Compute([]Obs{{QID: "q1", Access: "api", OK: false}}, Filter{})
	if s.Visibility.Value != nil || s.Visibility.CI != nil || s.ShareOfVoice != nil {
		t.Fatalf("no successful runs must be 'not measured', got %+v", s)
	}
	if s.Failed != 1 {
		t.Fatalf("failed = %d", s.Failed)
	}
}

func TestComputePlatformFilter(t *testing.T) {
	a := obs("q1", "api", false, true, 1)
	b := obs("q1", "api", false, false, 0)
	b.Platform = "kimi"
	s := Compute([]Obs{a, b}, Filter{Platform: "kimi"})
	if s.Visibility.N != 1 || s.Visibility.X != 0 {
		t.Fatalf("platform filter = %+v", s.Visibility)
	}
}

func TestAccessOf(t *testing.T) {
	for mode, want := range map[string]string{"": "api", "api": "api", "manual": "web", "web": "web", "scraped": "web"} {
		if got := AccessOf(mode); got != want {
			t.Fatalf("AccessOf(%q) = %q, want %q", mode, got, want)
		}
	}
}
