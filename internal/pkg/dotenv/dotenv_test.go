// SPDX-License-Identifier: AGPL-3.0-or-later

package dotenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndLoadToml(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("DEEPSEEK_API_KEY")
	os.Unsetenv("CRAFTSAIL_GROWTH_USER")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(TomlPath(dir), []byte("[mysql.default]\ndsn = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, map[string]string{"DEEPSEEK_API_KEY": "sk-abc", "CRAFTSAIL_GROWTH_USER": "root"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(TomlPath(dir))
	s := string(b)
	if !strings.Contains(s, "[mysql.default]") || !strings.Contains(s, "DEEPSEEK_API_KEY") || !strings.Contains(s, "user =") {
		t.Fatalf("%s", s)
	}
	os.Unsetenv("DEEPSEEK_API_KEY")
	os.Unsetenv("CRAFTSAIL_GROWTH_USER")
	if err := Load(dir); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DEEPSEEK_API_KEY") != "sk-abc" {
		t.Fatalf("key %s", os.Getenv("DEEPSEEK_API_KEY"))
	}
	if os.Getenv("CRAFTSAIL_GROWTH_USER") != "root" {
		t.Fatalf("user %s", os.Getenv("CRAFTSAIL_GROWTH_USER"))
	}
}

func TestWriteUpdateAndClear(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("DEEPSEEK_API_KEY")
	if err := Write(dir, map[string]string{"DEEPSEEK_API_KEY": "sk-abc", "GLM_MODEL": "keep-me"}); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, map[string]string{"DEEPSEEK_API_KEY": "sk-xyz"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(TomlPath(dir))
	s := string(b)
	if !strings.Contains(s, "sk-xyz") || !strings.Contains(s, "keep-me") {
		t.Fatalf("%s", s)
	}
	if err := Write(dir, map[string]string{"DEEPSEEK_API_KEY": ""}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		t.Fatal("env not cleared")
	}
}

func TestRejectUnknownKey(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, map[string]string{"PATH": "/tmp"}); err == nil {
		t.Fatal("expected reject")
	}
}

func TestLoadDoesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, map[string]string{"OPENAI_API_KEY": "fromfile"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "fromenv")
	if err := Load(dir); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("OPENAI_API_KEY") != "fromenv" {
		t.Fatalf("%s", os.Getenv("OPENAI_API_KEY"))
	}
}

func TestWriteGoogleHTTPProxy(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("GOOGLE_HTTP_PROXY")
	if err := Write(dir, map[string]string{"GOOGLE_HTTP_PROXY": "http://127.0.0.1:7890"}); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("GOOGLE_HTTP_PROXY")
	if err := Load(dir); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("GOOGLE_HTTP_PROXY") != "http://127.0.0.1:7890" {
		t.Fatalf("%q", os.Getenv("GOOGLE_HTTP_PROXY"))
	}
}

func TestWriteGoogleSAJSON(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("GOOGLE_SA_JSON")
	raw := `{"type":"service_account","client_email":"sa@x.iam.gserviceaccount.com"}`
	if err := Write(dir, map[string]string{"GOOGLE_SA_JSON": raw}); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("GOOGLE_SA_JSON")
	if err := Load(dir); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("GOOGLE_SA_JSON") != raw {
		t.Fatalf("got %q", os.Getenv("GOOGLE_SA_JSON"))
	}
}

func TestLoadRawRetentionFromKeys(t *testing.T) {
	dir := t.TempDir()
	os.Unsetenv("GOOGLE_RAW_RETENTION_DAYS")
	defer os.Unsetenv("GOOGLE_RAW_RETENTION_DAYS")
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(TomlPath(dir), []byte("[keys]\nGOOGLE_RAW_RETENTION_DAYS = \"90\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Load(dir); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("GOOGLE_RAW_RETENTION_DAYS"); got != "90" {
		t.Fatalf("GOOGLE_RAW_RETENTION_DAYS = %q, want 90", got)
	}
}
