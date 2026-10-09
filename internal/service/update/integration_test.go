// SPDX-License-Identifier: AGPL-3.0-or-later

package update

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"archive/tar"
	"github.com/craftsail/craftsail-growth/internal/model"
)

// Optional packaging smoke test: supply two real release executables for the
// current platform. It also runs in the release container as its non-root user.
func TestReleaseBinaryRoundTrip(t *testing.T) {
	oldPath, newPath := os.Getenv("CRAFTSAIL_UPDATE_TEST_OLD"), os.Getenv("CRAFTSAIL_UPDATE_TEST_NEW")
	if oldPath == "" || newPath == "" {
		t.Skip("set CRAFTSAIL_UPDATE_TEST_OLD and CRAFTSAIL_UPDATE_TEST_NEW to release executables")
	}
	ctx := context.Background()
	old, err := probeBinary(ctx, oldPath)
	if err != nil {
		t.Fatal(err)
	}
	next, err := probeBinary(ctx, newPath)
	if err != nil {
		t.Fatal(err)
	}
	if compare(next.Version, old.Version) <= 0 {
		t.Fatal("new executable must be newer")
	}
	exe := filepath.Join(t.TempDir(), "craftsail-growth")
	if err = copyAtomic(oldPath, exe); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	archive := fixtureArchive(t, "craftsail-growth", tar.TypeReg, raw)
	name := fmt.Sprintf("craftsail-growth_%s_%s_%s.tar.gz", next.Version, runtime.GOOS, runtime.GOARCH)
	f := &fakeSource{rows: []model.Release{{Tag: "v" + next.Version, Assets: []model.ReleaseAsset{{Name: name, URL: "archive"}, {Name: "checksums.txt", URL: "sums"}}}}, files: map[string][]byte{name: archive, "checksums.txt": []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), name))}}
	s := &Service{current: old, executable: exe, source: f, probe: probeBinary}
	if err = s.Apply(ctx, "v"+next.Version, false); err != nil {
		t.Fatal(err)
	}
	state, err := s.Check(ctx, false)
	if err != nil || state.PendingVersion != next.Version {
		t.Fatalf("pending %+v %v", state, err)
	}
	if got := diskVersion(t, s, exe); got != next.Version {
		t.Fatal(got)
	}
	s.current = next // new process reports its own embedded metadata after restart
	if err = s.Apply(ctx, "backup", true); err != nil {
		t.Fatal(err)
	}
	if got := diskVersion(t, s, exe); got != old.Version {
		t.Fatal(got)
	}
	if got := diskVersion(t, s, exe+".backup"); got != next.Version {
		t.Fatal(got)
	}
	t.Logf("real binary update and rollback: %s -> %s -> %s (uid %d)", old.Version, next.Version, old.Version, os.Getuid())
}
