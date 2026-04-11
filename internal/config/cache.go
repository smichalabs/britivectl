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
	// SyncedAt is the wall-clock time when the cache was last populated.
	SyncedAt time.Time `json:"syncedAt"`
	// Profiles is keyed by alias.
	Profiles map[string]Profile `json:"profiles"`
}

// LoadProfilesCache reads the profile cache from disk. Returns (nil, nil) if
// the cache file does not yet exist -- callers should treat that as a cache
// miss and trigger a sync.
func LoadProfilesCache() (*ProfilesCache, error) {
	path := ProfilesCachePath()
	data, err := os.ReadFile(path) //nolint:gosec // path is under our controlled cache dir
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading profiles cache: %w", err)
	}

	var cache ProfilesCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing profiles cache: %w", err)
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
