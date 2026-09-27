// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigCopiesExample(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, "default.example.toml")
	target := filepath.Join(dir, "default.toml")
	if err := os.WriteFile(example, []byte("[auth]\nuser = \"admin\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := ensureConfig(target)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected config to be created from example")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "[auth]\nuser = \"admin\"\n" {
		t.Fatalf("copied content = %q", got)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config should be 0600, got %v", info.Mode().Perm())
	}
}

func TestEnsureConfigKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "default.toml")
	if err := os.WriteFile(target, []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "default.example.toml"), []byte("example"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := ensureConfig(target)
	if err != nil || created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "mine" {
		t.Fatalf("existing config was overwritten: %q", got)
	}
}

func TestEnsureConfigWithoutExample(t *testing.T) {
	dir := t.TempDir()
	if _, err := ensureConfig(filepath.Join(dir, "default.toml")); err == nil {
		t.Fatal("expected an error when neither config nor example exists")
	}
}
