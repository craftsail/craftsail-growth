// SPDX-License-Identifier: AGPL-3.0-or-later

package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/buildinfo"
)

const maxBinary int64 = 256 << 20
const maxArchive int64 = 256 << 20

type boundedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.limit-b.buffer.Len() {
		return 0, fmt.Errorf("metadata too large")
	}
	return b.buffer.Write(p)
}
func (b *boundedBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *boundedBuffer) String() string { return b.buffer.String() }

func probeBinary(ctx context.Context, file string) (buildinfo.Info, error) {
	var info buildinfo.Info
	st, err := os.Lstat(file)
	if err != nil || !st.Mode().IsRegular() {
		return info, fmt.Errorf("not a regular executable")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, file, "--version-json")
	var out = boundedBuffer{limit: 4096}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return info, err
	}
	err = json.Unmarshal(out.Bytes(), &info)
	return info, err
}
func (s *Service) downloadAndInstall(ctx context.Context, tag string, asset, sums model.ReleaseAsset) error {
	var b = boundedBuffer{limit: 1 << 20}
	if err := s.source.Download(ctx, sums, &b, 1<<20); err != nil {
		return fmt.Errorf("%w: %v", ErrDownload, err)
	}
	expected := ""
	for _, line := range strings.Split(b.String(), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == asset.Name {
			if expected != "" {
				return ErrChecksum
			}
			expected = strings.ToLower(f[0])
		}
	}
	if hash, err := hex.DecodeString(expected); err != nil || len(hash) != sha256.Size {
		return ErrChecksum
	}
	dir, err := os.MkdirTemp(filepath.Dir(s.executable), ".craftsail-update-")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInstall, err)
	}
	defer os.RemoveAll(dir)
	archive, err := os.OpenFile(filepath.Join(dir, "release.tar.gz"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return ErrInstall
	}
	defer archive.Close()
	digest := sha256.New()
	if err := s.source.Download(ctx, asset, io.MultiWriter(archive, digest), maxArchive); err != nil {
		return fmt.Errorf("%w: %v", ErrDownload, err)
	}
	if hex.EncodeToString(digest.Sum(nil)) != expected {
		return ErrChecksum
	}
	if _, err := archive.Seek(0, 0); err != nil {
		return ErrArchive
	}
	candidate := filepath.Join(dir, "craftsail-growth")
	if err := extract(archive, candidate); err != nil {
		return fmt.Errorf("%w: %v", ErrArchive, err)
	}
	info, err := s.probe(ctx, candidate)
	if err != nil || info.BuildType != "release" || info.Repository != s.current.Repository || strings.TrimPrefix(info.Version, "v") != strings.TrimPrefix(tag, "v") {
		return ErrArchive
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.install(candidate)
}
func extract(src io.Reader, dest string) error {
	gz, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	defer gz.Close()
	// Bound total uncompressed input, including irrelevant entries/padding.
	tr := tar.NewReader(io.LimitReader(gz, maxBinary+(16<<20)))
	found := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		clean := path.Clean(h.Name)
		if strings.Contains(h.Name, "\\") || path.IsAbs(h.Name) || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("unsafe archive path")
		}
		if h.Typeflag == tar.TypeDir {
			continue
		}
		if h.Typeflag != tar.TypeReg || h.Size < 0 || h.Size > maxBinary {
			return fmt.Errorf("unsupported archive entry")
		}
		if path.Base(clean) != "craftsail-growth" {
			continue
		}
		if found {
			return fmt.Errorf("duplicate executable")
		}
		found = true
		f, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(f, tr, h.Size)
		syncErr := f.Sync()
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if !found {
		return fmt.Errorf("no executable in archive")
	}
	return nil
}

// install preserves a synced copy before one atomic rename over the executable.
// There is never a window where the launch path is missing.
func (s *Service) install(candidate string) error {
	if err := copyAtomic(s.executable, s.executable+".backup"); err != nil {
		return fmt.Errorf("%w: %v", ErrInstall, err)
	}
	st, err := os.Stat(s.executable)
	if err != nil {
		return ErrInstall
	}
	if err := os.Chmod(candidate, st.Mode().Perm()&0777); err != nil {
		return ErrInstall
	}
	if err := os.Rename(candidate, s.executable); err != nil {
		return fmt.Errorf("%w: %v", ErrInstall, err)
	}
	syncDir(filepath.Dir(s.executable))
	return nil
}
func copyAtomic(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	st, err := input.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() > maxBinary {
		return fmt.Errorf("invalid executable")
	}
	output, err := os.CreateTemp(filepath.Dir(dst), ".craftsail-copy-")
	if err != nil {
		return err
	}
	name := output.Name()
	defer os.Remove(name)
	defer output.Close()
	if _, err = io.Copy(output, io.LimitReader(input, maxBinary+1)); err != nil {
		return err
	}
	if err = output.Chmod(st.Mode().Perm() & 0777); err != nil {
		return err
	}
	if err = output.Sync(); err != nil {
		return err
	}
	if err = output.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, dst); err != nil {
		return err
	}
	syncDir(filepath.Dir(dst))
	return nil
}
func syncDir(dir string) {
	if f, err := os.Open(dir); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}
}
func (s *Service) restore(ctx context.Context) error {
	file := s.executable + ".backup"
	info, err := s.probe(ctx, file)
	if _, ok := version(info.Version); err != nil || !ok || info.BuildType != "release" || info.Repository != s.current.Repository {
		return ErrTarget
	}
	// Stage before install replaces the backup, allowing a second rollback to undo.
	dir, err := os.MkdirTemp(filepath.Dir(s.executable), ".craftsail-rollback-")
	if err != nil {
		return ErrInstall
	}
	defer os.RemoveAll(dir)
	candidate := filepath.Join(dir, "craftsail-growth")
	if err := copyAtomic(file, candidate); err != nil {
		return ErrInstall
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.install(candidate)
}
