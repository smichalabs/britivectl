package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smichalabs/britivectl/internal/config"
)

func setupXDG(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))
}

func TestLoadProfilesCache_Missing(t *testing.T) {
	setupXDG(t)
	cache, err := config.LoadProfilesCache("")
	if !errors.Is(err, config.ErrCacheMiss) {
		t.Fatalf("LoadProfilesCache() error = %v, want ErrCacheMiss", err)
	}
	if cache != nil {
		t.Errorf("expected nil cache on missing file, got %+v", cache)
	}
}

func TestSaveAndLoadProfilesCache(t *testing.T) {
	setupXDG(t)

	original := &config.ProfilesCache{
		Profiles: map[string]config.Profile{
			"dev": {BritivePath: "AWS/Dev/Admin", Cloud: "aws"},
		},
	}
	if err := config.SaveProfilesCache(original); err != nil {
		t.Fatalf("SaveProfilesCache() error = %v", err)
	}

	loaded, err := config.LoadProfilesCache("")
	if err != nil {
		t.Fatalf("LoadProfilesCache() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadProfilesCache() returned nil")
	}
	if loaded.SyncedAt.IsZero() {
		t.Error("SyncedAt should be set by SaveProfilesCache")
	}
	if _, ok := loaded.Profiles["dev"]; !ok {
		t.Error("dev profile not found in loaded cache")
	}
}

func TestSaveProfilesCache_Nil(t *testing.T) {
	setupXDG(t)
	if err := config.SaveProfilesCache(nil); err == nil {
		t.Error("expected error for nil cache, got nil")
	}
}

func TestLoadProfilesCache_Malformed(t *testing.T) {
	setupXDG(t)

	if err := os.MkdirAll(filepath.Dir(config.ProfilesCachePath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ProfilesCachePath(), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadProfilesCache("")
	if err == nil {
		t.Fatal("expected error for malformed cache, got nil")
	}
}

func TestSaveProfilesCache_CreateDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

	// Make the parent of XDG cache unwritable.
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(tmpDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(tmpDir, 0o755) })

	if err := config.SaveProfilesCache(&config.ProfilesCache{Profiles: map[string]config.Profile{}}); err == nil {
		t.Error("expected error with unwritable cache dir, got nil")
	}
}

func TestShouldAutoSync(t *testing.T) {
	fresh := &config.ProfilesCache{
		SyncedAt: time.Now().Add(-5 * time.Minute),
		Profiles: map[string]config.Profile{"dev": {}},
	}
	stale := &config.ProfilesCache{
		SyncedAt: time.Now().Add(-2 * time.Hour),
		Profiles: map[string]config.Profile{"dev": {}},
	}
	empty := &config.ProfilesCache{
		SyncedAt: time.Now(),
		Profiles: map[string]config.Profile{},
	}

	cases := []struct {
		name    string
		cache   *config.ProfilesCache
		refresh bool
		noSync  bool
		want    bool
	}{
		{"nil cache, default -> sync", nil, false, false, true},
		{"nil cache, --no-sync -> skip", nil, false, true, false},
		{"fresh cache, default -> skip", fresh, false, false, false},
		{"fresh cache, --refresh -> sync", fresh, true, false, true},
		{"fresh cache, --no-sync -> skip", fresh, false, true, false},
		{"stale cache, default -> sync", stale, false, false, true},
		{"stale cache, --no-sync wins", stale, false, true, false},
		{"empty cache, default -> sync", empty, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := config.ShouldAutoSync(tc.cache, tc.refresh, tc.noSync, 1*time.Hour)
			if got != tc.want {
				t.Errorf("ShouldAutoSync() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProfilesCache_IsStale(t *testing.T) {
	cases := []struct {
		name     string
		cache    *config.ProfilesCache
		maxAge   time.Duration
		wantTrue bool
	}{
		{"nil cache", nil, 1 * time.Hour, true},
		{"zero time", &config.ProfilesCache{}, 1 * time.Hour, true},
		{"fresh", &config.ProfilesCache{SyncedAt: time.Now().Add(-1 * time.Minute)}, 1 * time.Hour, false},
		{"expired", &config.ProfilesCache{SyncedAt: time.Now().Add(-2 * time.Hour)}, 1 * time.Hour, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cache.IsStale(tc.maxAge)
			if got != tc.wantTrue {
				t.Errorf("IsStale() = %v, want %v", got, tc.wantTrue)
			}
		})
	}
}
