// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type seqRoundTrip struct {
	codes   []int
	n       int
	err     []error
	lastURL string
}

func (s *seqRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	if req != nil && req.URL != nil {
		s.lastURL = req.URL.String()
	}
	i := s.n
	if i < len(s.err) && s.err[i] != nil {
		s.n++
		return nil, s.err[i]
	}
	code := 200
	if i < len(s.codes) {
		code = s.codes[i]
	}
	s.n++
	body := `{"choices":[{"message":{"content":"你好"}}],"model":"deepseek-v4-flash"}`
	if code != 200 {
		body = "err"
	}
	return &http.Response{
		StatusCode: code,
		Status:     http.StatusText(code),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func testAsker(codes []int, errs []error) (*Asker, *seqRoundTrip) {
	tr := &seqRoundTrip{codes: codes, err: errs}
	return &Asker{
		HTTP:  &http.Client{Transport: tr},
		Sleep: func(time.Duration) {},
		Env: func(k string) string {
			if k == "DEEPSEEK_API_KEY" {
				return "test-key"
			}
			return ""
		},
	}, tr
}

func TestAskRetry429ThenSuccess(t *testing.T) {
	a, tr := testAsker([]int{429, 500, 200}, nil)
	res := a.Ask("deepseek", "测试问题")
	if !res.OK || tr.n != 3 {
		t.Fatalf("ok=%v calls=%d err=%s", res.OK, tr.n, res.Error)
	}
}

func TestAskRetryExhausted(t *testing.T) {
	a, tr := testAsker([]int{500, 500, 500, 500, 500}, nil)
	res := a.Ask("deepseek", "测试问题")
	if res.OK || tr.n != 3 {
		t.Fatalf("ok=%v calls=%d", res.OK, tr.n)
	}
}

func TestAskTimeoutRetried(t *testing.T) {
	a, tr := testAsker([]int{200}, []error{timeoutErr{}, nil})
	res := a.Ask("deepseek", "测试问题")
	if !res.OK || tr.n != 2 {
		t.Fatalf("ok=%v calls=%d err=%s", res.OK, tr.n, res.Error)
	}
}

func TestAskNoRetry400(t *testing.T) {
	a, tr := testAsker([]int{400, 400, 400}, nil)
	res := a.Ask("deepseek", "测试问题")
	if res.OK || tr.n != 1 {
		t.Fatalf("ok=%v calls=%d", res.OK, tr.n)
	}
}

func TestAskCustomBasePicksCatalogModel(t *testing.T) {
	tr := &catalogTrip{models: []string{"grok-3-mini", "grok-4.3"}}
	a := &Asker{
		HTTP:  &http.Client{Transport: tr},
		Sleep: func(time.Duration) {},
		Env: func(k string) string {
			switch k {
			case "OPENAI_API_KEY":
				return "test-key"
			case "OPENAI_BASE":
				return "http://relay.test/v1"
			}
			return ""
		},
		picked: map[string]string{},
	}
	res := a.Ask("openai", "ping")
	if !res.OK {
		t.Fatalf("%s", res.Error)
	}
	if tr.usedModel != "grok-3-mini" {
		t.Fatalf("model %s", tr.usedModel)
	}
}

type catalogTrip struct {
	models    []string
	usedModel string
}

func (c *catalogTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Path == "/v1/models" || strings.HasSuffix(req.URL.Path, "/models") {
		var arr []map[string]string
		for _, id := range c.models {
			arr = append(arr, map[string]string{"id": id})
		}
		b, _ := json.Marshal(map[string]any{"data": arr})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(b))), Header: make(http.Header), Request: req}, nil
	}
	var body struct {
		Model string `json:"model"`
	}
	_ = json.NewDecoder(req.Body).Decode(&body)
	c.usedModel = body.Model
	ok := `{"choices":[{"message":{"content":"ok"}}],"model":"` + body.Model + `"}`
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(ok)), Header: make(http.Header), Request: req}, nil
}

func TestAskUsesCustomBase(t *testing.T) {
	a, tr := testAsker([]int{200}, nil)
	a.Env = func(k string) string {
		switch k {
		case "DEEPSEEK_API_KEY":
			return "test-key"
		case "DEEPSEEK_BASE":
			return "https://relay.example.com/v1"
		}
		return ""
	}
	res := a.Ask("deepseek", "测试问题")
	if !res.OK {
		t.Fatalf("%s", res.Error)
	}
	if tr.lastURL != "https://relay.example.com/v1/chat/completions" {
		t.Fatalf("url %s", tr.lastURL)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string { return "Timeout" }
func (timeoutErr) Timeout() bool { return true }
