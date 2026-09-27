// SPDX-License-Identifier: AGPL-3.0-or-later

package webstats

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type tokenRT struct {
	body string
	n    int
}

func (t *tokenRT) RoundTrip(req *http.Request) (*http.Response, error) {
	t.n++
	b, _ := io.ReadAll(req.Body)
	t.body = string(b)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"access_token":"ya29.test","expires_in":3600}`)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func testSAJSON(t *testing.T) string {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	raw, _ := json.Marshal(map[string]string{
		"type":         "service_account",
		"client_email": "sa@test.iam.gserviceaccount.com",
		"private_key":  string(pemBytes),
		"token_uri":    "https://oauth2.googleapis.com/token",
	})
	return string(raw)
}

func TestAccessTokenFromServiceAccount(t *testing.T) {
	tr := &tokenRT{}
	c := &Client{HTTP: &http.Client{Transport: tr}, Now: func() time.Time { return time.Unix(1_700_000_000, 0) }}
	tok, err := c.AccessToken(testSAJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	if tok != "ya29.test" {
		t.Fatalf("tok %q", tok)
	}
	if !strings.Contains(tr.body, "urn:ietf:params:oauth:grant-type:jwt-bearer") {
		t.Fatalf("body %s", tr.body)
	}
}

func TestParseSARequiresEmailAndKey(t *testing.T) {
	if _, err := ParseSA(`{"type":"service_account"}`); err == nil {
		t.Fatal("expected error")
	}
}
