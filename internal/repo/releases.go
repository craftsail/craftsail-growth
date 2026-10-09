// SPDX-License-Identifier: AGPL-3.0-or-later

package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
)

var releaseRepository = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+$`)
var ErrNoRelease = errors.New("no published release")

// Releases only accepts GitHub URLs. Tokens are sent to the API, never assets.
// Transport is injectable for offline tests; production uses the default transport.
type Releases struct {
	Repository, Token string
	Transport         http.RoundTripper
}

func (r *Releases) client() *http.Client {
	return &http.Client{Timeout: 5 * time.Minute, Transport: r.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || !trustedReleaseURL(req.URL) {
			return errors.New("untrusted release redirect")
		}
		req.Header.Del("Authorization")
		return nil
	}}
}
func trustedReleaseURL(u *url.URL) bool {
	if u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	switch u.Hostname() {
	case "api.github.com", "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	}
	return false
}
func (r *Releases) fetch(ctx context.Context, raw string, dst io.Writer, limit int64, api bool) error {
	u, err := url.Parse(raw)
	if err != nil || !trustedReleaseURL(u) {
		return errors.New("untrusted release URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "craftsail-growth-updater")
	if api && u.Hostname() == "api.github.com" && r.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.Token)
	}
	res, err := r.client().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound && api {
		return ErrNoRelease
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("release server returned HTTP %d", res.StatusCode)
	}
	if res.ContentLength > limit {
		return errors.New("release response too large")
	}
	n, err := io.Copy(dst, io.LimitReader(res.Body, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		return errors.New("release response too large")
	}
	return nil
}
func (r *Releases) List(ctx context.Context) ([]model.Release, error) {
	if !releaseRepository.MatchString(r.Repository) {
		return nil, errors.New("invalid release repository")
	}
	var b strings.Builder
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := r.fetch(ctx, "https://api.github.com/repos/"+r.Repository+"/releases?per_page=30", &b, 4<<20, true); err != nil {
		return nil, err
	}
	var rows []model.Release
	if err := json.Unmarshal([]byte(b.String()), &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNoRelease
	}
	return rows, nil
}
func (r *Releases) Download(ctx context.Context, asset model.ReleaseAsset, dst io.Writer, limit int64) error {
	// Initial asset URL must belong to this repository, not merely any GitHub host.
	u, err := url.Parse(asset.URL)
	prefix := "/" + r.Repository + "/releases/download/"
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || !strings.HasPrefix(u.Path, prefix) {
		return errors.New("asset does not belong to release repository")
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, prefix), "/")
	if len(parts) != 2 || parts[0] == "" || parts[0] == "." || parts[0] == ".." || parts[1] == "" || parts[1] == "." || parts[1] == ".." || strings.Contains(u.Path, "\\") {
		return errors.New("invalid release asset path")
	}
	return r.fetch(ctx, asset.URL, dst, limit, false)
}
