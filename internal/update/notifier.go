package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/smichalabs/britivectl/internal/config"
)

// DefaultTTL is how long the cached "latest version" entry is trusted before
// the notifier triggers a fresh GitHub API call. Picked to keep the network
// footprint to at most one call per laptop per day.
const DefaultTTL = 24 * time.Hour

// fetchTimeout caps the goroutine that talks to GitHub so a slow or
// unreachable network never delays the user's command perceptibly.
const fetchTimeout = 2 * time.Second

// Cache is the on-disk record the notifier reads to decide whether to nudge
// the user. Stored at config.UpdateCheckCachePath().
type Cache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

// Notifier coordinates update-check caching, opportunistic refresh, and the
// one-line notice printed to the user. All collaborators are pluggable so
// tests can drive every code path without touching the real filesystem,
// network, or wall clock.
type Notifier struct {
	CacheFile  string
	CurrentVer string
	TTL        time.Duration
	Now        func() time.Time
	Fetch      func(ctx context.Context) (string, error)
	Env        func(string) string
}

// DefaultNotifier wires a Notifier to real I/O for the running bctl process.
// The Fetch hook calls CheckLatest, which queries the same release repo
// install.sh uses, so a "newer version available" decision matches what a
// fresh curl install would pick up.
func DefaultNotifier(currentVer string) *Notifier {
	return &Notifier{
		CacheFile:  config.UpdateCheckCachePath(),
		CurrentVer: currentVer,
		TTL:        DefaultTTL,
		Now:        time.Now,
		Fetch: func(ctx context.Context) (string, error) {
			latest, _, err := CheckLatest(ctx, currentVer)
			if err != nil {
				return "", err
			}
			return latest, nil
		},
		Env: os.Getenv,
	}
}

// shouldSkip reports whether the notifier should make no network calls and
// print no notice. Honors a disable env var, common CI markers, and dev /
// pre-1.0 builds where users typically do not want a noisy nudge.
func (n *Notifier) shouldSkip() bool {
	if n.Env("BCTL_NO_UPDATE_CHECK") != "" {
		return true
	}
	if n.Env("CI") != "" || n.Env("GITHUB_ACTIONS") != "" {
		return true
	}
	cur := strings.TrimPrefix(n.CurrentVer, "v")
	if cur == "" || cur == "dev" || strings.Contains(cur, "-dev") || strings.HasPrefix(cur, "0.0.0") {
		return true
	}
	return false
}

// RefreshIfStale starts a goroutine that refreshes the cache file when it is
// missing or older than TTL. Returns a channel that is closed once the
// goroutine finishes (or never started), so callers can wait briefly for it
// before exiting if they want the next bctl run to see the fresh value.
//
// Always non-blocking. Errors are swallowed -- the next run will simply use
// the stale cache (or skip the notice) and try again later.
func (n *Notifier) RefreshIfStale(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	if n.shouldSkip() {
		close(done)
		return done
	}
	cache, _ := loadCache(n.CacheFile)
	if cache != nil && n.Now().Sub(cache.CheckedAt) < n.TTL {
		close(done)
		return done
	}

	go func() {
		defer close(done)
		fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
		defer cancel()
		latest, err := n.Fetch(fetchCtx)
		if err != nil || latest == "" {
			return
		}
		_ = saveCache(n.CacheFile, &Cache{CheckedAt: n.Now(), LatestVersion: latest})
	}()
	return done
}

// MaybePrintNotice writes a one-line update nudge to w when the cached latest
// version is newer than CurrentVer and isTTY is true. Returns whether a notice
// was printed. Never makes a network call.
//
// The TTY gate is the caller's job to pass in -- the notifier does not assume
// w is os.Stderr or any other specific writer, so tests can capture output.
func (n *Notifier) MaybePrintNotice(w io.Writer, isTTY bool) bool {
	if n.shouldSkip() || !isTTY {
		return false
	}
	cache, err := loadCache(n.CacheFile)
	if err != nil || cache == nil || cache.LatestVersion == "" {
		return false
	}
	if !versionIsNewer(cache.LatestVersion, n.CurrentVer) {
		return false
	}
	fmt.Fprintf(w, "\nA new bctl release is available: v%s (you have v%s). Run `bctl update` to upgrade.\n",
		strings.TrimPrefix(cache.LatestVersion, "v"),
		strings.TrimPrefix(n.CurrentVer, "v"),
	)
	return true
}

// versionIsNewer compares two version strings (with or without a leading "v")
// using simple string equality. Reports true when latest is non-empty and
// differs from current. We deliberately do not parse semver here: GitHub's
// "latest release" tag is the source of truth, and any difference means the
// upstream has moved on.
func versionIsNewer(latest, current string) bool {
	l := strings.TrimPrefix(latest, "v")
	c := strings.TrimPrefix(current, "v")
	return l != "" && l != c
}

// loadCache reads the on-disk notifier cache. A missing file is not an error
// -- it just means the notifier has never run on this machine.
func loadCache(path string) (*Cache, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is from config.UpdateCheckCachePath, not user input
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// saveCache writes the notifier cache atomically: write to a temp file in the
// same directory, then rename. This avoids a torn read if the process is
// interrupted partway through the write.
func saveCache(path string, c *Cache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
