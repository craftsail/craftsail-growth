// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const googleAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"

var googleScopes = []string{
	"openid",
	"email",
	"https://www.googleapis.com/auth/webmasters.readonly",
	"https://www.googleapis.com/auth/analytics.readonly",
}

func OAuthConfigured() bool {
	return strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID")) != "" &&
		strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")) != ""
}

func UserConnected() bool {
	return OAuthConfigured() && strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN")) != ""
}

func AuthCodeURL(clientID, redirectURI, state string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(googleScopes, " "))
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("state", state)
	return googleAuthURL + "?" + q.Encode()
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
}

func (c *Client) ExchangeCode(clientID, secret, code, redirectURI string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", clientID)
	form.Set("client_secret", secret)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	return c.postToken(form)
}

func (c *Client) UserAccessToken() (string, error) {
	refresh := strings.TrimSpace(os.Getenv("GOOGLE_REFRESH_TOKEN"))
	id := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID"))
	secret := strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"))
	if refresh == "" || id == "" || secret == "" {
		return "", fmt.Errorf("Google is not connected")
	}
	now := c.now()
	if tok, ok := c.cached("user:"+id, now); ok {
		return tok, nil
	}
	form := url.Values{}
	form.Set("refresh_token", refresh)
	form.Set("client_id", id)
	form.Set("client_secret", secret)
	form.Set("grant_type", "refresh_token")
	out, err := c.postToken(form)
	if err != nil {
		return "", err
	}
	c.store("user:"+id, out.AccessToken, now.Add(time.Duration(out.ExpiresIn)*time.Second-time.Minute))
	return out.AccessToken, nil
}

func (c *Client) postToken(form url.Values) (tokenResponse, error) {
	req, err := http.NewRequest(http.MethodPost, defaultTokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cli, err := c.httpClient()
	if err != nil {
		return tokenResponse{}, err
	}
	res, err := cli.Do(req)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("token request: %w", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 8192))
	if err != nil {
		return tokenResponse{}, err
	}
	if res.StatusCode >= 400 {
		return tokenResponse{}, fmt.Errorf("token HTTP %d %s", res.StatusCode, snippet(b))
	}
	var out tokenResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return tokenResponse{}, err
	}
	if out.AccessToken == "" {
		return tokenResponse{}, fmt.Errorf("token response has no access_token")
	}
	return out, nil
}
