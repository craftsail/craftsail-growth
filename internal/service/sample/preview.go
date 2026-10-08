// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"context"
	"fmt"
	"github.com/craftsail/craftsail-growth/internal/model"
	"time"
)

type Preview struct {
	Calls     int      `json:"calls"`
	Questions int      `json:"questions"`
	Repeat    int      `json:"repeat"`
	Engines   []string `json:"engines"`
	Available []string `json:"available"`
	EstTokens int      `json:"est_tokens"`
	Revision  string   `json:"revision"`
}

// Preview performs no writes and makes no provider requests. Credential
// presence indicates eligibility, not proof that credentials will succeed.
func (s *Service) Preview(ctx context.Context, slug string, in RunInput) (*Preview, error) {
	if in.Repeat == 0 {
		in.Repeat = 1
	}
	if in.Repeat < 1 || in.Repeat > 10 || in.Limit < 0 || in.Limit > 1000 {
		return nil, fmt.Errorf("repeat must be 1–10; limit must be 0–1000")
	}
	p, err := s.projects.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	_, qs, err := s.cfgOf(ctx, p)
	if err != nil {
		return nil, err
	}
	records := make([]model.Question, len(qs))
	for i := range qs {
		records[i] = qs[i].Record
	}
	revision := model.LibraryRevision(records, p.SamplingLanguage, p.TargetRegion)
	for i := range qs {
		qs[i].Revision = revision
	}
	out := &Preview{Repeat: in.Repeat, Revision: revision, Engines: []string{}, Available: []string{}}
	selected := map[string]bool{}
	for _, code := range in.Platforms {
		selected[code] = true
	}
	for _, provider := range Providers {
		if provider.Manual || (!s.BypassAvailable && !Available(provider.Code)) {
			continue
		}
		out.Available = append(out.Available, provider.Code)
		if len(selected) == 0 || selected[provider.Code] {
			out.Engines = append(out.Engines, provider.Code)
		}
	}
	calls, err := s.planCalls(ctx, p, qs, out.Engines, in, in.Repeat, dateOnly(time.Now()))
	if err != nil {
		return nil, err
	}
	unique := map[string]bool{}
	for _, c := range calls {
		unique[c.q.ID] = true
	}
	out.Questions = len(unique)
	out.Calls = len(calls)
	out.EstTokens = estimateTokens(out.Calls, 1, 1)
	return out, nil
}
