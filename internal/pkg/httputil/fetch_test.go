// SPDX-License-Identifier: AGPL-3.0-or-later

package httputil

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type seqTransport struct {
	codes []int
	n     int
	uas   []string
}

func (s *seqTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.uas = append(s.uas, req.Header.Get("User-Agent"))
	code := s.codes[s.n]
	if s.n < len(s.codes)-1 {
		s.n++
	}
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader("<html>ok</html>")),
		Header:     http.Header{"Content-Type": []string{"text/html"}},
		Request:    req,
	}, nil
}

func clientWith(codes ...int) (*Client, *seqTransport) {
	tr := &seqTransport{codes: codes}
	return &Client{HTTP: &http.Client{Transport: tr}, Sleep: func() {}}, tr
}

func TestRetry500Then200(t *testing.T) {
	c, tr := clientWith(500, 200)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 1})
	if res.Status != 200 || tr.n != 1 {
		t.Fatalf("status=%d calls=%d", res.Status, len(tr.uas))
	}
	if len(tr.uas) != 2 {
		t.Fatalf("calls=%d", len(tr.uas))
	}
}

func TestRetry429(t *testing.T) {
	c, tr := clientWith(429, 200)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 1})
	if res.Status != 200 || len(tr.uas) != 2 {
		t.Fatalf("status=%d calls=%d", res.Status, len(tr.uas))
	}
}

func TestNoRetry404(t *testing.T) {
	c, tr := clientWith(404)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 1})
	if res.Status != 404 || len(tr.uas) != 1 {
		t.Fatalf("status=%d calls=%d", res.Status, len(tr.uas))
	}
}

func TestRetryExhaustedReturnsLast(t *testing.T) {
	c, tr := clientWith(500, 500)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 1})
	if res.Status != 500 || len(tr.uas) != 2 {
		t.Fatalf("status=%d calls=%d", res.Status, len(tr.uas))
	}
}

func Test403FallsBackToBrowserUA(t *testing.T) {
	c, tr := clientWith(403, 200)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 0})
	if res.Status != 200 || !res.UAFallback {
		t.Fatalf("status=%d fallback=%v", res.Status, res.UAFallback)
	}
	if !strings.Contains(tr.uas[0], "geo-skill") {
		t.Fatalf("first ua %q", tr.uas[0])
	}
	if strings.Contains(tr.uas[1], "geo-skill") {
		t.Fatalf("second ua %q", tr.uas[1])
	}
}

func Test403OnBothUAs(t *testing.T) {
	c, tr := clientWith(403, 403)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 0})
	if res.Status != 403 || len(tr.uas) != 2 {
		t.Fatalf("status=%d calls=%d", res.Status, len(tr.uas))
	}
}

func TestExplicitUANeverFallsBack(t *testing.T) {
	c, tr := clientWith(403)
	res := c.Fetch("http://x.test/", FetchOpts{Retries: 0, UA: "AI-Bot/1.0"})
	if res.Status != 403 || len(tr.uas) != 1 {
		t.Fatalf("status=%d calls=%d", res.Status, len(tr.uas))
	}
}

func TestSkipNonHTMLPath(t *testing.T) {
	c, _ := clientWith(200)
	res := c.Fetch("http://x.test/app.zip", FetchOpts{})
	if res.Status != 0 || res.Error == "" {
		t.Fatalf("%+v", res)
	}
}
