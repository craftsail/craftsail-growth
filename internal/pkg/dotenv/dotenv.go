// SPDX-License-Identifier: AGPL-3.0-or-later

package dotenv

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var allowed = map[string]bool{
	"ZHIPUAI_API_KEY": true, "ARK_API_KEY": true, "DEEPSEEK_API_KEY": true,
	"MOONSHOT_API_KEY": true, "MINIMAX_API_KEY": true,
	"GEMINI_API_KEY": true, "OPENAI_API_KEY": true, "ANTHROPIC_API_KEY": true,
	"XAI_API_KEY": true, "PERPLEXITY_API_KEY": true,
	"GLM_BASE": true, "ARK_BASE": true, "DEEPSEEK_BASE": true,
	"MOONSHOT_BASE": true, "MINIMAX_BASE": true, "GEMINI_BASE": true,
	"OPENAI_BASE": true, "ANTHROPIC_BASE": true, "GROK_BASE": true, "PERPLEXITY_BASE": true,
	"GLM_MODEL": true, "ARK_MODEL": true, "DEEPSEEK_MODEL": true,
	"MOONSHOT_MODEL": true, "MINIMAX_MODEL": true, "GEMINI_MODEL": true,
	"OPENAI_MODEL": true, "ANTHROPIC_MODEL": true, "GROK_MODEL": true, "PERPLEXITY_MODEL": true,
	"CRAFTSAIL_GROWTH_TOKEN": true, "CRAFTSAIL_GROWTH_USER": true, "CRAFTSAIL_GROWTH_PASSWORD": true,
	"GOOGLE_SA_JSON": true, "GOOGLE_HTTP_PROXY": true,
	"GOOGLE_OAUTH_CLIENT_ID": true, "GOOGLE_OAUTH_CLIENT_SECRET": true, "GOOGLE_REFRESH_TOKEN": true, "GOOGLE_RAW_RETENTION_DAYS": true,
}

var authField = map[string]string{
	"CRAFTSAIL_GROWTH_USER":     "user",
	"CRAFTSAIL_GROWTH_PASSWORD": "password",
	"CRAFTSAIL_GROWTH_TOKEN":    "token",
}

func Get(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func FindRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for d := wd; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return wd
		}
	}
}

func TomlPath(root string) string {
	return filepath.Join(root, "config", "default.toml")
}

func Load(root string) error {
	if err := loadFile(TomlPath(root)); err != nil {
		return err
	}
	return loadEnvFile(filepath.Join(root, ".env"))
}

func loadEnvFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") || !strings.Contains(ln, "=") {
			continue
		}
		ln = strings.TrimPrefix(ln, "export ")
		k, v, _ := strings.Cut(ln, "=")
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if allowed[k] && os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
	return nil
}

func loadFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	sec := ""
	for _, ln := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(ln)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			sec = strings.TrimSpace(trim[1 : len(trim)-1])
			continue
		}
		k, v, ok := parseAssign(trim)
		if !ok {
			continue
		}
		envKey := ""
		if sec == "keys" && allowed[k] {
			envKey = k
		}
		if sec == "auth" {
			for ek, field := range authField {
				if field == k {
					envKey = ek
					break
				}
			}
		}
		if envKey == "" {
			continue
		}
		if os.Getenv(envKey) == "" {
			_ = os.Setenv(envKey, v)
		}
	}
	return nil
}

func Write(root string, updates map[string]string) error {
	for k := range updates {
		if !allowed[k] {
			return fmt.Errorf("config key %s cannot be written", k)
		}
	}
	path := TomlPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw := ""
	if b, err := os.ReadFile(path); err == nil {
		raw = string(b)
	} else if !os.IsNotExist(err) {
		return err
	}
	secs := splitSections(raw)
	for k, v := range updates {
		if field, ok := authField[k]; ok {
			secs = setKV(secs, "auth", field, v)
		} else {
			secs = setKV(secs, "keys", k, v)
		}
		if v != "" {
			_ = os.Setenv(k, v)
		} else {
			_ = os.Unsetenv(k)
		}
	}
	body := joinSections(secs)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	return nil
}

func parseAssign(ln string) (string, string, bool) {
	i := strings.IndexByte(ln, '=')
	if i <= 0 {
		return "", "", false
	}
	k := strings.TrimSpace(ln[:i])
	v := strings.TrimSpace(ln[i+1:])
	if k == "" {
		return "", "", false
	}
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		if u, err := strconv.Unquote(v); err == nil {
			v = u
		} else {
			v = v[1 : len(v)-1]
		}
	}
	return k, v, true
}

type section struct {
	name  string
	lines []string
}

func splitSections(raw string) []section {
	if raw == "" {
		return nil
	}
	var out []section
	cur := section{}
	for _, ln := range strings.Split(strings.TrimRight(raw, "\n"), "\n") {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			out = append(out, cur)
			cur = section{name: strings.TrimSpace(trim[1 : len(trim)-1])}
			continue
		}
		cur.lines = append(cur.lines, ln)
	}
	out = append(out, cur)
	return out
}

func setKV(secs []section, name, key, val string) []section {
	found := false
	assign := key + " = " + strconv.Quote(val)
	for i := range secs {
		if secs[i].name != name {
			continue
		}
		found = true
		replaced := false
		prefix := key + " "
		prefixEq := key + "="
		var lines []string
		for _, ln := range secs[i].lines {
			t := strings.TrimSpace(ln)
			if strings.HasPrefix(t, prefix) || strings.HasPrefix(t, prefixEq) {
				if !replaced {
					lines = append(lines, assign)
					replaced = true
				}
				continue
			}
			lines = append(lines, ln)
		}
		if !replaced {
			lines = append(lines, assign)
		}
		secs[i].lines = lines
		return secs
	}
	if !found {
		secs = append(secs, section{name: name, lines: []string{assign}})
	}
	return secs
}

func joinSections(secs []section) string {
	var b strings.Builder
	for i, s := range secs {
		if s.name != "" {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString("[" + s.name + "]\n")
		}
		for _, ln := range s.lines {
			b.WriteString(ln)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
