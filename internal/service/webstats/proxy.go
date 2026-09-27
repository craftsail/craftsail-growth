// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const googleHTTPTimeout = 60 * time.Second

// ProxyFromEnv reads GOOGLE_HTTP_PROXY, then HTTPS_PROXY, then HTTP_PROXY.
// A host:port value is treated as an HTTP proxy. socks5 is rejected.
func ProxyFromEnv() (*url.URL, error) {
	raw := firstEnv("GOOGLE_HTTP_PROXY", "HTTPS_PROXY", "HTTP_PROXY")
	if raw == "" {
		return nil, nil
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Scheme == "" {
		return nil, fmt.Errorf("invalid proxy address, for example http://127.0.0.1:7890")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u, nil
	default:
		return nil, fmt.Errorf("use an HTTP proxy, for example http://127.0.0.1:7890")
	}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func (c *Client) httpClient() (*http.Client, error) {
	if c != nil && c.HTTP != nil {
		return c.HTTP, nil
	}
	proxy, err := ProxyFromEnv()
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   20 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 45 * time.Second,
	}
	if proxy != nil {
		tr.Proxy = http.ProxyURL(proxy)
	}
	return &http.Client{Timeout: googleHTTPTimeout, Transport: tr}, nil
}
