// SPDX-License-Identifier: AGPL-3.0-or-later

package sample

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func Verify(code, key, base string) error {
	p, ok := Lookup(code)
	if !ok || p.Manual {
		return fmt.Errorf("this engine has no API")
	}
	if strings.TrimSpace(key) == "" {
		key = osGet(p.KeyEnv)
	}
	if key == "" {
		return fmt.Errorf("no API key set")
	}
	if strings.TrimSpace(base) == "" {
		base = BaseFor(p)
	} else {
		base = NormalizeBase(base, p.Base)
	}
	cli := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+"/models", nil)
	if err != nil {
		return err
	}
	if p.Protocol == "anthropic" {
		req.Header.Set("x-api-key", key)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	res, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect: %v", err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 8000))
	if res.StatusCode >= 400 {
		msg := strings.TrimSpace(string(b))
		var obj map[string]any
		if json.Unmarshal(b, &obj) == nil {
			if e, ok := obj["error"].(map[string]any); ok {
				if m, ok := e["message"].(string); ok {
					msg = m
				}
			}
		}
		if len([]rune(msg)) > 180 {
			msg = string([]rune(msg)[:180])
		}
		return fmt.Errorf("HTTP %d %s", res.StatusCode, msg)
	}
	return nil
}

func osGet(k string) string {
	return strings.TrimSpace(os.Getenv(k))
}
