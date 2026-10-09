// SPDX-License-Identifier: AGPL-3.0-or-later

package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/buildinfo"
)

type fakeSource struct {
	rows      []model.Release
	files     map[string][]byte
	err       error
	calls     int
	downloads int
}

func (f *fakeSource) List(context.Context) ([]model.Release, error) { f.calls++; return f.rows, f.err }
func (f *fakeSource) Download(_ context.Context, a model.ReleaseAsset, w io.Writer, _ int64) error {
	f.downloads++
	if f.err != nil {
		return f.err
	}
	_, err := w.Write(f.files[a.Name])
	return err
}
func metadata(v string) buildinfo.Info {
	return buildinfo.Info{Version: v, Commit: "commit-" + v, BuildType: "release", Repository: "craftsail/craftsail-growth"}
}
func fixtureArchive(t *testing.T, name string, kind byte, body []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	h := &tar.Header{Name: name, Mode: 0755, Typeflag: kind, Size: int64(len(body))}
	if kind == tar.TypeSymlink {
		h.Linkname = "/tmp/target"
		h.Size = 0
		body = nil
	}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func fixture(t *testing.T) (*Service, *fakeSource) {
	t.Helper()
	if !supportedPlatform() {
		t.Skip("binary replacement only on Unix")
	}
	exe := filepath.Join(t.TempDir(), "craftsail-growth")
	info := metadata("1.0.0")
	raw, _ := json.Marshal(info)
	if err := os.WriteFile(exe, raw, 0755); err != nil {
		t.Fatal(err)
	}
	f := &fakeSource{files: map[string][]byte{}}
	for _, v := range []string{"1.0.0", "2.0.0", "0.9.0", "0.8.0", "0.7.0", "0.6.0"} {
		name := fmt.Sprintf("craftsail-growth_%s_%s_%s.tar.gz", v, runtime.GOOS, runtime.GOARCH)
		body, _ := json.Marshal(metadata(v))
		a := fixtureArchive(t, "craftsail-growth", tar.TypeReg, body)
		f.files[name] = a
		f.rows = append(f.rows, model.Release{Tag: "v" + v, Assets: []model.ReleaseAsset{{Name: name, URL: "archive"}, {Name: "checksums.txt", URL: "sums"}}})
	}
	sums := ""
	for name, data := range f.files {
		sums += fmt.Sprintf("%x  %s\n", sha256.Sum256(data), name)
	}
	f.files["checksums.txt"] = []byte(sums)
	s := &Service{current: info, executable: exe, source: f, instance: "test", probe: func(_ context.Context, p string) (buildinfo.Info, error) {
		var v buildinfo.Info
		b, err := os.ReadFile(p)
		if err != nil {
			return v, err
		}
		err = json.Unmarshal(b, &v)
		return v, err
	}}
	return s, f
}
func diskVersion(t *testing.T, s *Service, file string) string {
	t.Helper()
	v, err := s.probe(context.Background(), file)
	if err != nil {
		t.Fatal(err)
	}
	return v.Version
}
func TestUpdateRestartRollback(t *testing.T) {
	s, f := fixture(t)
	ctx := context.Background()
	state, err := s.Check(ctx, false)
	if err != nil || !state.HasUpdate || !state.CanUpdate || len(state.Rollbacks) != 3 {
		t.Fatalf("check %+v %v", state, err)
	}
	if state.Latest.Tag != "v2.0.0" {
		t.Fatal(state.Latest.Tag)
	}
	if _, err = s.Check(ctx, false); err != nil || f.calls != 1 {
		t.Fatalf("cache: %v calls=%d", err, f.calls)
	}
	if err = s.Apply(ctx, "v2.0.0", false); err != nil {
		t.Fatal(err)
	}
	if got := diskVersion(t, s, s.executable); got != "2.0.0" {
		t.Fatal(got)
	}
	if got := diskVersion(t, s, s.executable+".backup"); got != "1.0.0" {
		t.Fatal(got)
	}
	state, _ = s.Check(ctx, false)
	if state.PendingVersion != "2.0.0" || state.Current.Version != "1.0.0" {
		t.Fatalf("pending %+v", state)
	}
	if err = s.Apply(ctx, "v2.0.0", false); !errors.Is(err, ErrPending) {
		t.Fatal(err)
	}
	// A second service instance recovers pending state from disk, not RAM.
	other := &Service{current: metadata("2.0.0"), executable: s.executable, source: s.source, probe: s.probe, instance: "restarted"}
	state, _ = other.Check(ctx, false)
	if state.PendingVersion != "" || state.BackupVersion != "1.0.0" {
		t.Fatalf("after restart %+v", state)
	}
	f.err = errors.New("offline")
	if err = other.Apply(ctx, "backup", true); err != nil {
		t.Fatal(err)
	}
	if got := diskVersion(t, s, s.executable); got != "1.0.0" {
		t.Fatal(got)
	}
	if got := diskVersion(t, s, s.executable+".backup"); got != "2.0.0" {
		t.Fatal(got)
	}
}
func TestRejectedReleaseNeverReplacesExecutable(t *testing.T) {
	for _, kind := range []string{"hash", "missing", "duplicate", "traversal", "symlink", "wrong-version", "wrong-repository", "malformed", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			s, f := fixture(t)
			ctx := context.Background()
			a, _, _ := assetFor(f.rows[1])
			raw := f.files[a.Name]
			switch kind {
			case "hash":
				f.files[a.Name] = append(raw, byte(1))
			case "missing":
				f.files["checksums.txt"] = []byte("unrelated")
			case "duplicate":
				f.files["checksums.txt"] = append(f.files["checksums.txt"], f.files["checksums.txt"]...)
			case "traversal":
				raw = fixtureArchive(t, "../craftsail-growth", tar.TypeReg, []byte("bad"))
			case "symlink":
				raw = fixtureArchive(t, "craftsail-growth", tar.TypeSymlink, nil)
			case "wrong-version":
				b, _ := json.Marshal(metadata("5.0.0"))
				raw = fixtureArchive(t, "craftsail-growth", tar.TypeReg, b)
			case "wrong-repository":
				v := metadata("2.0.0")
				v.Repository = "attacker/repo"
				b, _ := json.Marshal(v)
				raw = fixtureArchive(t, "craftsail-growth", tar.TypeReg, b)
			case "malformed":
				raw = []byte("not gzip")
			case "cancelled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			if kind != "hash" && kind != "missing" && kind != "duplicate" {
				f.files[a.Name] = raw
				f.files["checksums.txt"] = []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(raw), a.Name))
			}
			if err := s.Apply(ctx, "v2.0.0", false); err == nil {
				t.Fatal("unsafe release accepted")
			}
			if got := diskVersion(t, s, s.executable); got != "1.0.0" {
				t.Fatal(got)
			}
			if _, err := os.Stat(s.executable + ".backup"); !os.IsNotExist(err) {
				t.Fatalf("backup changed on validation failure: %v", err)
			}
		})
	}
}
func TestSourceBuildAndVersionAllowlist(t *testing.T) {
	s, f := fixture(t)
	ctx := context.Background()
	for _, target := range []string{"v0.6.0", "v9.9.9", "../../evil", "https://evil.test/program"} {
		if err := s.Apply(ctx, target, true); !errors.Is(err, ErrTarget) {
			t.Fatalf("%s %v", target, err)
		}
	}
	if err := s.Apply(ctx, "v0.9.0", true); err != nil {
		t.Fatal(err)
	}
	if got := diskVersion(t, s, s.executable); got != "0.9.0" {
		t.Fatal(got)
	}
	s.current.BuildType = "source"
	before := f.downloads
	if err := s.Apply(ctx, "v2.0.0", false); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if f.downloads != before {
		t.Fatal("source build downloaded executable")
	}
	for _, v := range []string{"v1.0.0-beta.1", "01.2.3", "v1.0.0+build", "999999999999999999999.0.0", ""} {
		if _, ok := version(v); ok {
			t.Fatal(v)
		}
	}
	if compare("v1.10.0", "1.9.99") != 1 {
		t.Fatal("lexical comparison")
	}
}
func TestCacheFailureIsVisible(t *testing.T) {
	s, f := fixture(t)
	ctx := context.Background()
	_, _ = s.Check(ctx, false)
	f.err = errors.New("rate limit")
	state, err := s.Check(ctx, true)
	if err != nil || state.Warning != "update.checkFailed" || !state.Cached || state.Latest == nil {
		t.Fatalf("%+v %v", state, err)
	}
	if err := s.Apply(ctx, "v2.0.0", false); !errors.Is(err, ErrDownload) {
		t.Fatal(err)
	}
}
func TestOperationsSerializedAndRestartGuard(t *testing.T) {
	s, _ := fixture(t)
	ctx := context.Background()
	unlock, err := lockFile(s.executable + ".update.lock")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Apply(ctx, "v2.0.0", false); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	unlock()
	s.mu.Lock()
	if _, err = s.Check(ctx, false); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	s.mu.Unlock()
	if err = s.Restart(); !errors.Is(err, ErrRestart) {
		t.Fatal(err)
	}
	restarted := make(chan struct{})
	release := make(chan struct{})
	s.restart = func() { close(restarted); <-release }
	s.prepareRestart = func() error { return errors.New("running jobs") }
	if err = s.Restart(); !errors.Is(err, ErrJobs) {
		t.Fatal(err)
	}
	s.prepareRestart = func() error { return nil }
	if err = s.Restart(); err != nil {
		t.Fatal(err)
	}
	<-restarted
	if err = s.Apply(ctx, "v2.0.0", false); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	if err = s.Restart(); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
	close(release)
}
func TestExtractRejectsMissingAndOversizedBinary(t *testing.T) {
	b := fixtureArchive(t, "readme.txt", tar.TypeReg, []byte("readme"))
	if err := extract(bytes.NewReader(b), filepath.Join(t.TempDir(), "bin")); err == nil {
		t.Fatal("missing executable accepted")
	}
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "craftsail-growth", Mode: 0755, Size: maxBinary + 1})
	_ = tw.Close()
	_ = gz.Close()
	if err := extract(bytes.NewReader(raw.Bytes()), filepath.Join(t.TempDir(), "bin")); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatal(err)
	}
}

func TestMetadataBufferCannotBypassLimitViaReadFrom(t *testing.T) {
	var out = boundedBuffer{limit: 4}
	_, err := io.Copy(&out, io.LimitReader(strings.NewReader("oversized output"), 100))
	if err == nil || len(out.Bytes()) > 4 {
		t.Fatal("metadata output limit bypassed")
	}
}
