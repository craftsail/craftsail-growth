// SPDX-License-Identifier: AGPL-3.0-or-later

package httputil

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const MaxBytes = 4_000_000

const UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/124.0 Safari/537.36 geo-skill/1.0"

const UABrowser = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
	"(KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var skipExt = regexp.MustCompile(`(?i)\.(zip|gz|tgz|bz2|7z|rar|dmg|pkg|exe|msi|apk|ipa|deb|rpm|bin|iso` +
	`|mp4|mov|avi|mkv|mp3|wav|flac|png|jpe?g|gif|webp|svg|ico|bmp|tiff` +
	`|woff2?|ttf|eot|css|js|csv|xlsx?|docx?|pptx?|pdf)(\?|$)`)
var skipPath = regexp.MustCompile(`(?i)/(downloads?|dl|releases?|assets|static|cdn)/`)

func Fetchable(u string) bool {
	if skipExt.MatchString(u) || skipPath.MatchString(u) {
		return false
	}
	tail := strings.ToLower(u)
	tail = strings.TrimRight(tail, "/")
	if i := strings.LastIndex(tail, "/"); i >= 0 {
		tail = tail[i+1:]
	}
	return tail != "download" && tail != "dl"
}

type Result struct {
	URL         string
	FinalURL    string
	Status      int
	HTML        string
	ContentType string
	XRobotsTag  string
	Elapsed     float64
	Error       string
	UAFallback  bool
	Redirects   []string // URLs followed before the final response
}

type FetchOpts struct {
	Timeout time.Duration
	Retries int
	UA      string
}

type Client struct {
	HTTP  *http.Client
	Sleep func()
}

func New() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 12 * time.Second},
		Sleep: func() {
			time.Sleep(1500 * time.Millisecond)
		},
	}
}

func (c *Client) Fetch(raw string, opt FetchOpts) Result {
	if !Fetchable(raw) {
		return Result{URL: raw, FinalURL: raw, Error: "skipped: not a web page (download, media or static file)"}
	}
	timeout := opt.Timeout
	if timeout == 0 {
		timeout = 12 * time.Second
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = func() { time.Sleep(1500 * time.Millisecond) }
	}
	var hops []string
	hc := *httpClient
	hc.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return http.ErrUseLastResponse
		}
		hops = append(hops, req.URL.String())
		return nil
	}
	last := ""
	uaPlan := []string{opt.UA}
	if opt.UA == "" {
		uaPlan = []string{UA, UABrowser}
	}
	for uaIdx, curUA := range uaPlan {
		for attempt := 0; attempt <= opt.Retries; attempt++ {
			t0 := time.Now()
			hops = hops[:0]
			req, err := http.NewRequest(http.MethodGet, raw, nil)
			if err != nil {
				last = err.Error()
				break
			}
			req.Header.Set("User-Agent", curUA)
			req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
			if uaIdx > 0 {
				req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
			}
			r, err := hc.Do(req)
			if err != nil {
				last = err.Error()
				if attempt < opt.Retries {
					sleep()
					continue
				}
				break
			}
			if (r.StatusCode >= 500 || r.StatusCode == 429) && attempt < opt.Retries {
				io.Copy(io.Discard, r.Body)
				r.Body.Close()
				sleep()
				continue
			}
			if (r.StatusCode == 403 || r.StatusCode == 406) && uaIdx+1 < len(uaPlan) {
				io.Copy(io.Discard, r.Body)
				r.Body.Close()
				last = "HTTP " + r.Status + " (default user agent blocked)"
				break
			}
			ctype := r.Header.Get("Content-Type")
			xrobots := r.Header.Get("X-Robots-Tag")
			cl := strings.ToLower(ctype)
			if ctype != "" && !strings.Contains(cl, "html") && !strings.Contains(cl, "text/plain") && !strings.Contains(cl, "xml") {
				io.Copy(io.Discard, r.Body)
				r.Body.Close()
				final := raw
				if r.Request != nil && r.Request.URL != nil {
					final = r.Request.URL.String()
				}
				return Result{URL: raw, FinalURL: final, Status: r.StatusCode, ContentType: ctype,
					XRobotsTag: xrobots, Elapsed: time.Since(t0).Seconds(), Redirects: append([]string(nil), hops...),
					Error: "skipped non-HTML content (" + strings.Split(ctype, ";")[0] + ")"}
			}
			limited := io.LimitReader(r.Body, MaxBytes)
			rawBody, _ := io.ReadAll(limited)
			r.Body.Close()
			enc := charsetOf(rawBody)
			html := decode(rawBody, enc)
			final := raw
			if r.Request != nil && r.Request.URL != nil {
				final = r.Request.URL.String()
			}
			return Result{
				URL: raw, FinalURL: final, Status: r.StatusCode, HTML: html,
				ContentType: ctype, XRobotsTag: xrobots,
				Elapsed: time.Since(t0).Seconds(), UAFallback: uaIdx > 0, Redirects: append([]string(nil), hops...),
			}
		}
	}
	return Result{URL: raw, FinalURL: raw, Error: last}
}

func (c *Client) FetchText(raw string) string {
	res := c.Fetch(raw, FetchOpts{Timeout: 8 * time.Second, Retries: 0})
	if res.Status == 200 {
		return res.HTML
	}
	return ""
}

var charsetRe = regexp.MustCompile(`(?i)charset=["']?([\w\-]+)`)

func charsetOf(raw []byte) string {
	head := raw
	if len(head) > 4000 {
		head = head[:4000]
	}
	if m := charsetRe.FindSubmatch(head); len(m) > 1 {
		return string(m[1])
	}
	return "utf-8"
}

func decode(raw []byte, enc string) string {
	// UTF-8 replace; other encodings treated as UTF-8 with replacement for v1.
	return string(raw)
}
