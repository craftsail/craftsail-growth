// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"net/http"
	"strings"
	"testing"
)

func TestProxyFromEnvAddsHTTPScheme(t *testing.T) {
	t.Setenv("GOOGLE_HTTP_PROXY", "127.0.0.1:7890")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("HTTP_PROXY", "")
	u, err := ProxyFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if u == nil || u.String() != "http://127.0.0.1:7890" {
		t.Fatalf("%v", u)
	}
}

func TestProxyFromEnvRejectsSocks(t *testing.T) {
	t.Setenv("GOOGLE_HTTP_PROXY", "socks5://127.0.0.1:7891")
	if _, err := ProxyFromEnv(); err == nil {
		t.Fatal("expected socks rejection")
	}
}

func TestAPIErrorKeepsEnableURL(t *testing.T) {
	body := []byte(`{"error":{"code":403,"message":"Google Search Console API has not been used in project 123456789012 before or it is disabled. Enable it by visiting https://console.developers.google.com/apis/api/searchconsole.googleapis.com/overview?project=123456789012 then retry."}}`)
	err := apiError("gsc", 403, body)
	if err == nil || !strings.Contains(err.Error(), "project=123456789012") {
		t.Fatal(err)
	}
}

func TestHTTPClientUsesConfiguredProxy(t *testing.T) {
	t.Setenv("GOOGLE_HTTP_PROXY", "http://127.0.0.1:7890")
	cli, err := (&Client{}).httpClient()
	if err != nil {
		t.Fatal(err)
	}
	tr, ok := cli.Transport.(*http.Transport)
	if !ok || tr.Proxy == nil {
		t.Fatal("missing transport proxy")
	}
	req, _ := http.NewRequest(http.MethodGet, "https://oauth2.googleapis.com/token", nil)
	got, err := tr.Proxy(req)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Host != "127.0.0.1:7890" {
		t.Fatalf("%v", got)
	}
}
