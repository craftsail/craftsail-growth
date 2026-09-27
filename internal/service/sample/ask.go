// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type AskResult struct {
	OK         bool
	Answer     string
	Citations  []Citation
	WebQueries []string
	Raw        map[string]any
	Error      string
	Searched   bool
	Model      string
}

type Asker struct {
	HTTP   *http.Client
	Sleep  func(time.Duration)
	Env    func(string) string
	picked map[string]string
	mu     sync.Mutex // guards picked; runs ask several engines at once
}

func NewAsker() *Asker {
	return &Asker{
		HTTP:   &http.Client{Timeout: 120 * time.Second},
		Sleep:  time.Sleep,
		Env:    os.Getenv,
		picked: map[string]string{},
	}
}

func (a *Asker) env(k string) string {
	if a.Env != nil {
		return a.Env(k)
	}
	return os.Getenv(k)
}

func (a *Asker) Ask(platform, question string) AskResult {
	return a.AskCtx(context.Background(), platform, question)
}

// AskCtx is Ask that gives up when ctx is cancelled, for example when a job
// is stopped, instead of waiting out the HTTP timeout and retries.
func (a *Asker) AskCtx(ctx context.Context, platform, question string) AskResult {
	p, ok := Lookup(platform)
	if !ok || p.Manual {
		return AskResult{Error: "unknown engine"}
	}
	key := a.env(p.KeyEnv)
	if key == "" {
		return AskResult{Error: "missing environment variable " + p.KeyEnv}
	}
	if p.Protocol == "ark" {
		return a.askOpenAICompat(ctx, p, key, question) // ark fallback to chat completions for v1
	}
	if p.Protocol == "anthropic" {
		return a.askAnthropic(ctx, p, key, question)
	}
	return a.askOpenAICompat(ctx, p, key, question)
}

func (a *Asker) askOpenAICompat(ctx context.Context, p Provider, key, question string) AskResult {
	model := a.pickModel(p)
	body, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": question}},
		"temperature": 0.7,
	})
	delays := []time.Duration{time.Second, 3 * time.Second}
	var last AskResult
	for attempt := 0; attempt < len(delays)+1; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseForEnv(p, a.env)+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return AskResult{Error: err.Error()}
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		res, err := a.do(req)
		if err != nil {
			last = AskResult{Error: err.Error()}
			if ctx.Err() == nil && isTimeout(err) && attempt < len(delays) {
				a.sleep(delays[attempt])
				continue
			}
			return last
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			last = AskResult{Error: "HTTP " + res.Status + ": " + trunc(string(raw), 300)}
			if (res.StatusCode == 429 || res.StatusCode >= 500) && attempt < len(delays) {
				a.sleep(delays[attempt])
				continue
			}
			return last
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return AskResult{Error: err.Error()}
		}
		answer := openaiContent(data)
		refs := openaiCitations(data)
		queries := QueriesFromPayload(data)
		model, _ := data["model"].(string)
		if model == "" {
			model = a.pickModel(p)
		}
		searched := p.Search || len(refs) > 0 || len(queries) > 0
		return AskResult{OK: true, Answer: answer, Citations: refs, WebQueries: queries, Raw: data, Model: model, Searched: searched}
	}
	return last
}

func (a *Asker) askAnthropic(ctx context.Context, p Provider, key, question string) AskResult {
	body, _ := json.Marshal(map[string]any{
		"model": ModelForEnv(p, a.env), "max_tokens": 4096,
		"messages": []map[string]string{{"role": "user", "content": question}},
	})
	delays := []time.Duration{time.Second, 3 * time.Second}
	var last AskResult
	for attempt := 0; attempt < len(delays)+1; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, BaseForEnv(p, a.env)+"/messages", bytes.NewReader(body))
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("content-type", "application/json")
		res, err := a.do(req)
		if err != nil {
			last = AskResult{Error: err.Error()}
			if ctx.Err() == nil && isTimeout(err) && attempt < len(delays) {
				a.sleep(delays[attempt])
				continue
			}
			return last
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			last = AskResult{Error: "HTTP " + res.Status + ": " + trunc(string(raw), 300)}
			if (res.StatusCode == 429 || res.StatusCode >= 500) && attempt < len(delays) {
				a.sleep(delays[attempt])
				continue
			}
			return last
		}
		var data map[string]any
		_ = json.Unmarshal(raw, &data)
		if data["stop_reason"] == "refusal" {
			return AskResult{Error: "refused by the safety classifier (stop_reason=refusal)"}
		}
		var answer strings.Builder
		if arr, ok := data["content"].([]any); ok {
			for _, b := range arr {
				m, _ := b.(map[string]any)
				if m["type"] == "text" {
					answer.WriteString(asStr(m["text"]))
				}
			}
		}
		model := asStr(data["model"])
		return AskResult{OK: true, Answer: answer.String(), Model: model}
	}
	return last
}

func (a *Asker) do(req *http.Request) (*http.Response, error) {
	c := a.HTTP
	if c == nil {
		c = http.DefaultClient
	}
	return c.Do(req)
}

func (a *Asker) sleep(d time.Duration) {
	if a.Sleep != nil {
		a.Sleep(d)
	}
}

func ModelForEnv(p Provider, env func(string) string) string {
	if p.ModelEnv != "" && env != nil {
		if v := env(p.ModelEnv); v != "" {
			return v
		}
	}
	return p.Model
}

func openaiContent(data map[string]any) string {
	ch, _ := data["choices"].([]any)
	if len(ch) == 0 {
		return ""
	}
	m, _ := ch[0].(map[string]any)
	msg, _ := m["message"].(map[string]any)
	return asStr(msg["content"])
}

func openaiCitations(data map[string]any) []Citation {
	var refs []Citation
	seen := map[string]bool{}
	add := func(u, title string) {
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		refs = append(refs, Citation{URL: u, Title: title})
	}
	if si, ok := data["search_info"].(map[string]any); ok {
		if arr, ok := si["search_results"].([]any); ok {
			for _, item := range arr {
				m, _ := item.(map[string]any)
				add(asStr(m["url"]), asStr(m["title"]))
			}
		}
	}
	if arr, ok := data["search_results"].([]any); ok {
		for _, item := range arr {
			m, _ := item.(map[string]any)
			add(asStr(m["url"]), asStr(m["title"]))
		}
	}
	if arr, ok := data["citations"].([]any); ok {
		for _, u := range arr {
			if s, ok := u.(string); ok {
				add(s, "")
			}
		}
	}
	return refs
}

func asStr(v any) string {
	s, _ := v.(string)
	return s
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "timeout") || strings.Contains(s, "deadline")
}
