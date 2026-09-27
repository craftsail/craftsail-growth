// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

func chooseModel(want, officialDefault string, ids []string) string {
	if len(ids) == 0 {
		if want != "" {
			return want
		}
		return officialDefault
	}
	has := map[string]bool{}
	for _, id := range ids {
		has[id] = true
	}
	if want != "" && has[want] {
		return want
	}
	for _, cand := range []string{officialDefault, "grok-3-mini", "grok-4.3", "grok-4.5", "gpt-4o-mini"} {
		if cand != "" && has[cand] {
			return cand
		}
	}
	for _, id := range ids {
		l := strings.ToLower(id)
		if strings.Contains(l, "image") || strings.Contains(l, "video") || strings.Contains(l, "imagine") {
			continue
		}
		return id
	}
	return ids[0]
}

func (a *Asker) pickModel(p Provider) string {
	want := ModelForEnv(p, a.env)
	if strings.TrimSpace(a.env(p.BaseEnv)) == "" {
		return want
	}
	a.mu.Lock()
	m, ok := a.picked[p.Code]
	a.mu.Unlock()
	if ok {
		return m
	}
	m = chooseModel(want, p.Model, a.listModels(p))
	a.mu.Lock()
	if a.picked == nil {
		a.picked = map[string]string{}
	}
	a.picked[p.Code] = m
	a.mu.Unlock()
	return m
}

func (a *Asker) listModels(p Provider) []string {
	key := a.env(p.KeyEnv)
	req, err := http.NewRequest(http.MethodGet, BaseForEnv(p, a.env)+"/models", nil)
	if err != nil {
		return nil
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := a.do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != 200 {
		return nil
	}
	var data struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	var ids []string
	for _, m := range data.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids
}
