package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// ProfilesCache is the persisted snapshot of profiles available to the user.
// It lives in xdgCacheDir() (not the config file) so that sync is cheap to
// invalidate and does not churn the user's config on every refresh.
type ProfilesCache struct {
	// Tenant records which Britive tenant this cache was synced from. Present
	// so that switching tenants via --tenant cannot silently hand stale
	// profile IDs from tenant A to tenant B.
	Tenant string `json:"tenant,omitempty"`
	// SyncedAt is the wall-clock time when the cache was last populated.
	SyncedAt time.Time `json:"syncedAt"`
	// Profiles is keyed by alias.
	Profiles map[string]Profile `json:"profiles"`
}

// ErrCacheMiss is returned by LoadProfilesCache when the cache file does
// not yet exist. Callers should treat it as a signal to run a sync rather
// than a hard failure.
var ErrCacheMiss = errors.New("profile cache does not exist")

// LoadProfilesCache reads the profile cache from disk. Returns ErrCacheMiss
// if the file has not been written yet or if the cached tenant does not match
// the one the caller is asking about -- both signal "do a fresh sync".
//
// Pass an empty tenant to skip tenant validation (used by surfaces like
// `bctl profiles list` where the current tenant cannot always be resolved,
// for instance before init has completed). Callers that know the tenant
// should always pass it so a --tenant switch forces a re-sync rather than
// silently returning the previous tenant's profile IDs.
func LoadProfilesCache(tenant string) (*ProfilesCache, error) {
	path := ProfilesCachePath()
	data, err := os.ReadFile(path) //nolint:gosec // path is under our controlled cache dir
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("reading profiles cache: %w", err)
	}

	var cache ProfilesCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing profiles cache: %w", err)
	}

	if tenant != "" && cache.Tenant != "" && cache.Tenant != tenant {
		return nil, ErrCacheMiss
	}
	return &cache, nil
}

// SaveProfilesCache writes the given cache to disk atomically.
func SaveProfilesCache(cache *ProfilesCache) error {
	if cache == nil {
		return errors.New("cache is nil")
	}
	if err := EnsureXDGDirs(); err != nil {
		return err
	}

	cache.SyncedAt = time.Now().UTC()
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling profiles cache: %w", err)
	}

	path := ProfilesCachePath()
	tmp, err := os.CreateTemp(xdgCacheDir(), "profiles-*.json")
	if err != nil {
		return fmt.Errorf("creating temp cache file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp cache file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp cache file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("chmod temp cache file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming cache file: %w", err)
	}
	return nil
}

// IsStale returns true if the cache is older than maxAge or has no SyncedAt.
func (c *ProfilesCache) IsStale(maxAge time.Duration) bool {
	if c == nil || c.SyncedAt.IsZero() {
		return true
	}
	return time.Since(c.SyncedAt) > maxAge
}
