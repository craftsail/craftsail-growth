// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultTokenURI = "https://oauth2.googleapis.com/token"
	googleScope     = "https://www.googleapis.com/auth/webmasters.readonly https://www.googleapis.com/auth/analytics.readonly"
	tokenSkew       = 60 * time.Second
	jwtTTL          = time.Hour
)

// SA is a Google service-account credential.
type SA struct {
	Email      string
	PrivateKey string
	TokenURI   string
}

type tokenEntry struct {
	token string
	until time.Time
}

// Client exchanges a service-account JWT for an access token and reads GSC/GA4.
type Client struct {
	HTTP *http.Client
	Now  func() time.Time
	// RowLimit is the GSC searchAnalytics page size. Values <= 0 use 25000.
	RowLimit int
	// GALimit is the GA4 runReport page size. Values <= 0 use 100000.
	GALimit int
	// OnPage receives each successful API page so the caller can archive the raw body.
	OnPage func(report, request, body string)

	mu    sync.Mutex
	cache map[string]tokenEntry
}

// ParseSA requires client_email and private_key. An empty token_uri becomes the Google token endpoint.
func ParseSA(raw string) (SA, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SA{}, fmt.Errorf("the service account needs client_email and private_key")
	}
	var doc struct {
		ClientEmail string `json:"client_email"`
		PrivateKey  string `json:"private_key"`
		TokenURI    string `json:"token_uri"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return SA{}, fmt.Errorf("invalid service account JSON: %w", err)
	}
	email := strings.TrimSpace(doc.ClientEmail)
	key := strings.TrimSpace(doc.PrivateKey)
	if email == "" || key == "" {
		return SA{}, fmt.Errorf("the service account needs client_email and private_key")
	}
	uri := strings.TrimSpace(doc.TokenURI)
	if uri == "" {
		uri = defaultTokenURI
	}
	return SA{Email: email, PrivateKey: key, TokenURI: uri}, nil
}

// AccessToken returns a cached Google access token, refreshing via RS256 JWT when needed.
func (c *Client) AccessToken(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	now := c.now()
	if tok, ok := c.cached(raw, now); ok {
		return tok, nil
	}
	sa, err := ParseSA(raw)
	if err != nil {
		return "", err
	}
	assertion, err := signJWT(sa, now)
	if err != nil {
		return "", err
	}
	// Keep the URN literal. QueryEscape would turn ':' into %3A.
	form := "grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=" + url.QueryEscape(assertion)
	req, err := http.NewRequest(http.MethodPost, sa.TokenURI, strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cli, err := c.httpClient()
	if err != nil {
		return "", err
	}
	res, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 8192))
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("token HTTP %d %s", res.StatusCode, snippet(b))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return "", fmt.Errorf("token response: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}
	exp := now.Add(jwtTTL)
	if out.ExpiresIn > 0 {
		tokExp := now.Add(time.Duration(out.ExpiresIn) * time.Second)
		if tokExp.Before(exp) {
			exp = tokExp
		}
	}
	c.store(raw, out.AccessToken, exp.Add(-tokenSkew))
	return out.AccessToken, nil
}

// KeyCard reports whether GOOGLE_SA_JSON parses as a service account.
func KeyCard() map[string]any {
	sa, err := ParseSA(os.Getenv("GOOGLE_SA_JSON"))
	tail := ""
	if err == nil {
		tail = emailTail(sa.Email)
	}
	return map[string]any{
		"code":      "google",
		"name":      "Google Search Console / Analytics",
		"market":    "global",
		"key_env":   "GOOGLE_SA_JSON",
		"ready":     err == nil,
		"key_tail":  tail,
		"manual":    false,
		"proxy_set": strings.TrimSpace(os.Getenv("GOOGLE_HTTP_PROXY")) != "",
	}
}

// Verify exchanges raw, or GOOGLE_SA_JSON when raw is empty, for an access token.
func Verify(raw string) error {
	if strings.TrimSpace(raw) == "" {
		raw = os.Getenv("GOOGLE_SA_JSON")
	}
	_, err := (&Client{}).AccessToken(raw)
	return err
}

func (c *Client) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) cached(raw string, now time.Time) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	ent, ok := c.cache[raw]
	if !ok || !now.Before(ent.until) {
		return "", false
	}
	return ent.token, true
}

func (c *Client) store(raw, token string, until time.Time) {
	if c == nil || !until.After(c.now()) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = map[string]tokenEntry{}
	}
	c.cache[raw] = tokenEntry{token: token, until: until}
}

func signJWT(sa SA, now time.Time) (string, error) {
	hdr, err := json.Marshal(struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{Alg: "RS256", Typ: "JWT"})
	if err != nil {
		return "", err
	}
	iat := now.Unix()
	claims, err := json.Marshal(struct {
		Iss   string `json:"iss"`
		Scope string `json:"scope"`
		Aud   string `json:"aud"`
		Iat   int64  `json:"iat"`
		Exp   int64  `json:"exp"`
	}{
		Iss:   sa.Email,
		Scope: googleScope,
		Aud:   sa.TokenURI,
		Iat:   iat,
		Exp:   iat + int64(jwtTTL/time.Second),
	})
	if err != nil {
		return "", err
	}
	input := b64(hdr) + "." + b64(claims)
	key, err := parseRSAPrivateKey(sa.PrivateKey)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return input + "." + b64(sig), nil
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		block, _ = pem.Decode([]byte(strings.ReplaceAll(pemStr, `\n`, "\n")))
	}
	if block == nil {
		return nil, fmt.Errorf("private_key is not PEM")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private_key is not RSA")
		}
		return rk, nil
	}
	rk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private_key: %w", err)
	}
	return rk, nil
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func emailTail(email string) string {
	if len(email) >= 4 {
		return email[len(email)-4:]
	}
	return email
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	r := []rune(s)
	if len(r) > 500 {
		return string(r[:500])
	}
	return s
}

func apiError(api string, status int, body []byte) error {
	msg := googleMessage(body)
	if u := enableURL(msg); u != "" {
		name := "Search Console API"
		if api == "ga" {
			name = "Google Analytics Data API"
		}
		return fmt.Errorf("%s is not enabled in this Google Cloud project. Enable it, wait a minute or two, then sync again: %s", name, u)
	}
	if status == 403 && strings.Contains(strings.ToLower(msg), "sufficient permission") {
		email := serviceAccountEmail()
		if api == "ga" {
			return fmt.Errorf("service account %s cannot read this GA4 property. Check that the property ID is the numeric ID from Admin > Property settings (not the account ID and not a G- measurement ID), give this email Viewer access, wait a minute or two, then sync again", email)
		}
		return fmt.Errorf("service account %s has no access to this Search Console property. Add this email under Settings > Users and permissions for the property", email)
	}
	if msg == "" {
		msg = snippet(body)
	}
	return fmt.Errorf("%s HTTP %d %s", api, status, msg)
}

func googleMessage(body []byte) string {
	var obj struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &obj) == nil && strings.TrimSpace(obj.Error.Message) != "" {
		return strings.TrimSpace(obj.Error.Message)
	}
	return ""
}

func serviceAccountEmail() string {
	sa, err := ParseSA(os.Getenv("GOOGLE_SA_JSON"))
	if err != nil {
		return "service account"
	}
	return sa.Email
}

func enableURL(msg string) string {
	if !strings.Contains(msg, "has not been used") && !strings.Contains(msg, "it is disabled") {
		return ""
	}
	i := strings.Index(msg, "https://")
	if i < 0 {
		return ""
	}
	rest := msg[i:]
	if end := strings.IndexAny(rest, " \n\t"); end >= 0 {
		return rest[:end]
	}
	return rest
}
