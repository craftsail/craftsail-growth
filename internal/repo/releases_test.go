// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
)

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func releaseResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestReleaseDownloadTrustAndLimits(t *testing.T) {
	calls := 0
	r := &Releases{Repository: "craftsail/craftsail-growth", Token: "secret", Transport: releaseTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("Authorization") != "" {
			t.Error("token sent to download")
		}
		return releaseResponse(200, "12345"), nil
	})}
	for _, raw := range []string{"http://github.com/craftsail/craftsail-growth/releases/download/v1/bin", "https://evil.test/bin", "https://github.com/craftsail/craftsail-growth/releases/download/../../evil", "https://github.com/craftsail/craftsail-growth/releases/download/%2e%2e/bin", "https://github.com/other/repo/releases/download/v1/bin", "https://github.com:8443/craftsail/craftsail-growth/releases/download/v1/bin"} {
		if err := r.Download(context.Background(), model.ReleaseAsset{URL: raw}, io.Discard, 100); err == nil {
			t.Fatal(raw)
		}
	}
	if calls != 0 {
		t.Fatal("untrusted URL requested")
	}
	a := model.ReleaseAsset{URL: "https://github.com/craftsail/craftsail-growth/releases/download/v1/bin"}
	if err := r.Download(context.Background(), a, io.Discard, 4); err == nil {
		t.Fatal("size limit ignored")
	}
	if err := r.Download(context.Background(), a, io.Discard, 5); err != nil {
		t.Fatal(err)
	}
}
func TestReleaseRedirectsDoNotLeakToken(t *testing.T) {
	for _, target := range []string{"https://objects.githubusercontent.com/asset", "https://evil.test/asset", "http://objects.githubusercontent.com/asset", "https://api.github.com:8443/asset"} {
		t.Run(target, func(t *testing.T) {
			calls := 0
			r := &Releases{Repository: "craftsail/craftsail-growth", Token: "secret", Transport: releaseTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Header.Get("Authorization") != "Bearer secret" {
						t.Error("missing API token")
					}
					res := releaseResponse(302, "")
					res.Header.Set("Location", target)
					return res, nil
				}
				if req.Header.Get("Authorization") != "" {
					t.Error("token leaked on redirect")
				}
				return releaseResponse(200, `[{"tag_name":"v1.0.0"}]`), nil
			})}
			_, err := r.List(context.Background())
			if target == "https://objects.githubusercontent.com/asset" {
				if err != nil || calls != 2 {
					t.Fatalf("%v %d", err, calls)
				}
			} else if err == nil || calls != 1 {
				t.Fatalf("unsafe redirect %v %d", err, calls)
			}
		})
	}
}
func TestReleaseDownloadFollowsGitHubAssetRedirect(t *testing.T) {
	r := &Releases{Repository: "craftsail/craftsail-growth", Token: "secret", Transport: releaseTransport(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Authorization") != "" {
			t.Error("token leaked")
		}
		if req.URL.Host == "github.com" {
			res := releaseResponse(302, "")
			res.Header.Set("Location", "https://release-assets.githubusercontent.com/blob?signature=opaque")
			return res, nil
		}
		return releaseResponse(200, "binary"), nil
	})}
	var dst bytes.Buffer
	if err := r.Download(context.Background(), model.ReleaseAsset{URL: "https://github.com/craftsail/craftsail-growth/releases/download/v1/bin"}, &dst, 100); err != nil {
		t.Fatal(err)
	}
	if dst.String() != "binary" {
		t.Fatal(dst.String())
	}
}
