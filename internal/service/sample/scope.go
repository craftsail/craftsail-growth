// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"encoding/json"
	"github.com/craftsail/craftsail-growth/internal/model"
	"sort"
)

func sampleScopes(rows []model.Sample) ([]string, []string, []string) {
	a, b, c := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		a[r.SamplingLanguage] = true
		b[r.TargetRegion] = true
		c[r.PromptRevision] = true
	}
	list := func(values map[string]bool) []string {
		out := []string{}
		for k := range values {
			if k == "" {
				k = "unknown"
			}
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	return list(a), list(b), list(c)
}
func sampleScopeKey(rows []model.Sample, q MeasureQuery, in measureBuild) (string, map[string]int) {
	strata := map[string]bool{}
	settings := map[string]string{}
	counts := map[string]int{}
	known := true
	for _, r := range rows {
		if !keepMeasureSample(r, q, in) || !r.OK || measureBranded(r, in) {
			continue
		}
		if r.PromptRevision == "" || r.SamplingLanguage == "" {
			known = false
		}
		strata[model.RowKey(r.PromptRevision, r.SamplingLanguage, r.TargetRegion)] = true
		modelName, _ := r.Raw["model"].(string)
		strategy, _ := r.Raw["strategy_version"].(string)
		if strategy == "" {
			known = false
		}
		if r.SampleMode == "api" {
			if _, ok := r.Raw["searched"].(bool); !ok {
				known = false
			}
		} else {
			mode, _ := r.Raw["session_mode"].(string)
			if mode == "" {
				known = false
			}
		}
		if modelName == "" {
			known = false
		}
		v, _ := json.Marshal([]any{modelName, r.Raw["searched"], r.Raw["strategy_version"], r.Raw["session_mode"]})
		key := r.Platform + ":" + r.SampleMode
		if old := settings[key]; old != "" && old != string(v) {
			known = false
		}
		settings[key] = string(v)
		counts[key+":"+r.QID]++
	}
	if !known || len(strata) != 1 {
		return "", counts
	}
	b, _ := json.Marshal([]any{strata, settings})
	return model.RowKey(string(b)), counts
}
func ComparableScopes(a, b *MeasureView) bool {
	if a == nil || b == nil || a.Truncated || b.Truncated || a.ScopeKey == "" || a.ScopeKey != b.ScopeKey || len(a.ScopeCounts) != len(b.ScopeCounts) {
		return false
	}
	an, bn := 0, 0
	for _, n := range a.ScopeCounts {
		an += n
	}
	for _, n := range b.ScopeCounts {
		bn += n
	}
	if an == 0 || bn == 0 {
		return false
	}
	for k, n := range a.ScopeCounts {
		if n*bn != b.ScopeCounts[k]*an {
			return false
		}
	}
	return true
}
