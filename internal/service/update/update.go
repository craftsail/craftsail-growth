// SPDX-License-Identifier: AGPL-3.0-or-later

// Package update implements the same operator-driven release/binary update flow
// as sub2api. Only stable releases from the build's repository are executable.
package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/craftsail/craftsail-growth/internal/model"
	"github.com/craftsail/craftsail-growth/internal/pkg/buildinfo"
	"github.com/craftsail/craftsail-growth/internal/repo"
)

var (
	ErrBusy        = errors.New("update.busy")
	ErrUnsupported = errors.New("update.unsupported")
	ErrPending     = errors.New("update.pending")
	ErrTarget      = errors.New("update.target")
	ErrDownload    = errors.New("update.download")
	ErrChecksum    = errors.New("update.checksum")
	ErrArchive     = errors.New("update.archive")
	ErrInstall     = errors.New("update.install")
	ErrRestart     = errors.New("update.restartUnavailable")
	ErrJobs        = errors.New("update.jobsRunning")
)
var stableVersion = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

func version(v string) ([3]uint64, bool) {
	var out [3]uint64
	m := stableVersion.FindStringSubmatch(v)
	if m == nil {
		return out, false
	}
	for i := range out {
		n, err := strconv.ParseUint(m[i+1], 10, 64)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
func compare(a, b string) int {
	av, _ := version(a)
	bv, _ := version(b)
	for i := range av {
		if av[i] > bv[i] {
			return 1
		}
		if av[i] < bv[i] {
			return -1
		}
	}
	return 0
}

type Source interface {
	List(context.Context) ([]model.Release, error)
	Download(context.Context, model.ReleaseAsset, io.Writer, int64) error
}
type State struct {
	Instance       string         `json:"instance"`
	Current        buildinfo.Info `json:"current"`
	Latest         *model.Release `json:"latest,omitempty"`
	HasUpdate      bool           `json:"has_update"`
	CanUpdate      bool           `json:"can_update"`
	CanRestart     bool           `json:"can_restart"`
	PendingVersion string         `json:"pending_version,omitempty"`
	BackupVersion  string         `json:"backup_version,omitempty"`
	Rollbacks      []string       `json:"rollbacks"`
	Warning        string         `json:"warning,omitempty"`
	Cached         bool           `json:"cached"`
	CheckedAt      string         `json:"checked_at,omitempty"`
}
type Service struct {
	instance   string
	current    buildinfo.Info
	source     Source
	executable string
	mu         sync.Mutex
	releases   []model.Release
	checked    time.Time
	attempted  time.Time
	lastError  error
	// probe is injectable only by same-package tests; production executes the
	// checksum-verified binary's side-effect-free metadata command.
	probe          func(context.Context, string) (buildinfo.Info, error)
	restart        func()
	prepareRestart func() error
	restarting     bool
}

func New(prepareRestart func() error) *Service {
	exe, _ := os.Executable()
	exe, _ = filepath.EvalSymlinks(exe)
	s := &Service{instance: strconv.FormatInt(time.Now().UnixNano(), 36), current: buildinfo.Current(), executable: exe, probe: probeBinary, prepareRestart: prepareRestart}
	s.source = &repo.Releases{Repository: s.current.Repository, Token: os.Getenv("CRAFTSAIL_GROWTH_UPDATE_GITHUB_TOKEN")}
	// Like sub2api, restart exits successfully and relies on the supervisor.
	// Explicit opt-in is needed for native installations, avoiding a dead service.
	if runtime.GOOS == "linux" && (os.Getpid() == 1 || os.Getenv("CRAFTSAIL_GROWTH_RESTART_ON_EXIT") == "1") {
		s.restart = func() { time.Sleep(time.Second); os.Exit(0) }
	}
	return s
}
func (s *Service) supported() bool {
	_, valid := version(s.current.Version)
	return s.current.BuildType == "release" && valid && s.executable != "" && supportedPlatform() && writable(filepath.Dir(s.executable))
}
func (s *Service) list(ctx context.Context, force bool) (bool, error) {
	if !force && s.lastError != nil && time.Since(s.attempted) < time.Minute {
		return len(s.releases) > 0, s.lastError
	}
	if !force && s.lastError == nil && !s.checked.IsZero() && time.Since(s.checked) < 20*time.Minute {
		return true, nil
	}
	rows, err := s.source.List(ctx)
	s.attempted = time.Now()
	s.lastError = err
	if err != nil {
		return false, err
	}
	stable := make([]model.Release, 0, len(rows))
	for _, r := range rows {
		if _, ok := version(r.Tag); ok && !r.Draft && !r.Prerelease {
			stable = append(stable, r)
		}
	}
	if len(stable) == 0 {
		s.lastError = repo.ErrNoRelease
		return false, repo.ErrNoRelease
	}
	sort.Slice(stable, func(i, j int) bool { return compare(stable[i].Tag, stable[j].Tag) > 0 })
	s.releases = stable
	s.checked = time.Now()
	return false, nil
}
func assetFor(r model.Release) (model.ReleaseAsset, model.ReleaseAsset, bool) {
	name := "craftsail-growth_" + strings.TrimPrefix(r.Tag, "v") + "_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	var archive, sums model.ReleaseAsset
	for _, a := range r.Assets {
		if a.Name == name {
			archive = a
		}
		if a.Name == "checksums.txt" {
			sums = a
		}
	}
	return archive, sums, archive.URL != "" && sums.URL != ""
}
func (s *Service) disk(ctx context.Context) (pending, backup string) {
	if s.current.BuildType != "release" {
		return
	}
	if info, err := s.probe(ctx, s.executable); err == nil && info.Repository == s.current.Repository && (compare(info.Version, s.current.Version) != 0 || info.Commit != s.current.Commit) {
		pending = info.Version
	}
	if info, err := s.probe(ctx, s.executable+".backup"); err == nil && info.Repository == s.current.Repository && info.BuildType == "release" {
		if _, ok := version(info.Version); ok {
			backup = info.Version
		}
	}
	return
}
func (s *Service) Check(ctx context.Context, force bool) (State, error) {
	if !s.mu.TryLock() {
		return State{}, ErrBusy
	}
	defer s.mu.Unlock()
	out := State{Instance: s.instance, Current: s.current, CanUpdate: s.supported(), CanRestart: s.restart != nil, Rollbacks: []string{}}
	cached, err := s.list(ctx, force)
	out.Cached = cached
	if err != nil {
		out.Warning = "update.checkFailed"
		out.Cached = len(s.releases) > 0
		if errors.Is(err, repo.ErrNoRelease) {
			out.Warning = "update.noRelease"
		}
	}
	if !s.checked.IsZero() {
		out.CheckedAt = s.checked.UTC().Format(time.RFC3339)
	}
	if len(s.releases) > 0 {
		r := s.releases[0]
		out.Latest = &r
		_, valid := version(s.current.Version)
		out.HasUpdate = valid && compare(r.Tag, s.current.Version) > 0
		if _, _, ok := assetFor(r); !ok {
			if out.Warning == "" {
				out.Warning = "update.noAsset"
			}
		}
		for _, r := range s.releases {
			if _, _, ok := assetFor(r); ok && compare(r.Tag, s.current.Version) < 0 && len(out.Rollbacks) < 3 {
				out.Rollbacks = append(out.Rollbacks, r.Tag)
			}
		}
	}
	out.PendingVersion, out.BackupVersion = s.disk(ctx)
	return out, nil
}
func (s *Service) operation(fn func() error) error {
	if !s.mu.TryLock() {
		return ErrBusy
	}
	defer s.mu.Unlock()
	if s.restarting {
		return ErrBusy
	}
	if !s.supported() {
		return ErrUnsupported
	}
	unlock, err := lockFile(s.executable + ".update.lock")
	if err != nil {
		return ErrBusy
	}
	defer unlock()
	return fn()
}

// Apply re-fetches releases and validates the requested tag, preventing silent
// upgrades to a version the administrator has not reviewed.
func (s *Service) Apply(ctx context.Context, target string, rollback bool) error {
	return s.operation(func() error {
		pending, _ := s.disk(ctx)
		if pending != "" && !rollback {
			return ErrPending
		}
		if rollback && target == "backup" {
			return s.restore(ctx)
		}
		if _, err := s.list(ctx, true); err != nil {
			return fmt.Errorf("%w: %v", ErrDownload, err)
		}
		for i, r := range s.releases {
			if r.Tag != target {
				continue
			}
			allowed := !rollback && i == 0 && compare(r.Tag, s.current.Version) > 0
			if rollback && compare(r.Tag, s.current.Version) < 0 {
				count := 0
				for _, older := range s.releases {
					if _, _, ok := assetFor(older); ok && compare(older.Tag, s.current.Version) < 0 {
						count++
						if older.Tag == target && count <= 3 {
							allowed = true
						}
					}
				}
			}
			if !allowed {
				return ErrTarget
			}
			archive, sums, ok := assetFor(r)
			if !ok {
				return ErrTarget
			}
			return s.downloadAndInstall(ctx, r.Tag, archive, sums)
		}
		return ErrTarget
	})
}
func (s *Service) Restart() error {
	if !s.mu.TryLock() {
		return ErrBusy
	}
	defer s.mu.Unlock()
	if s.restarting {
		return ErrBusy
	}
	if s.restart == nil {
		return ErrRestart
	}
	unlock, err := lockFile(s.executable + ".update.lock")
	if err != nil {
		return ErrBusy
	}
	if s.prepareRestart != nil {
		if err := s.prepareRestart(); err != nil {
			unlock()
			return ErrJobs
		}
	}
	s.restarting = true
	go func() { defer unlock(); s.restart() }()
	return nil
}
