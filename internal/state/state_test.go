package state

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
)

// fakeTokenStore is an in-memory TokenStore used by state tests so they do
// not touch the OS keychain. The keychain is deliberately avoided because
// go-keyring on macOS can prompt or fail in headless test environments.
type fakeTokenStore struct {
	token     string
	tokenType string
	expiry    int64
	getErr    error
}

func (f fakeTokenStore) GetToken(_ string) (string, error) { return f.token, f.getErr }
func (f fakeTokenStore) GetTokenType(_ string) string      { return f.tokenType }
func (f fakeTokenStore) GetTokenExpiry(_ string) int64     { return f.expiry }

// setupTestHome re-roots HOME and XDG so that config/cache files land in
// a tempdir instead of the real user home.
func setupTestHome(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))
}

// makeJWT builds a minimal JWT whose exp claim is the given unix timestamp.
func makeJWT(t *testing.T, exp int64) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(map[string]int64{"exp": exp})
	if err != nil {
		t.Fatal(err)
	}
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".fakesig"
}

func TestEnsureReady_HappyPath_CacheHit(t *testing.T) {
	setupTestHome(t)

	// Seed config
	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	// Seed fresh profile cache
	fresh := &config.ProfilesCache{
		SyncedAt: time.Now().UTC().Add(-1 * time.Hour),
		Profiles: map[string]config.Profile{
			"dev": {BritivePath: "AWS/Dev/Admin", Cloud: "aws"},
		},
	}
	if err := config.SaveProfilesCache(fresh); err != nil {
		t.Fatal(err)
	}

	token := makeJWT(t, time.Now().Add(1*time.Hour).Unix())
	cb := Callbacks{
		TokenStore: fakeTokenStore{
			token:     token,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(token),
		},
		RunInit: func(_ context.Context) (*config.Config, error) {
			t.Error("RunInit should not be called on happy path")
			return nil, errors.New("should not be called")
		},
		RunLogin: func(_ context.Context, _ string) (string, error) {
			t.Error("RunLogin should not be called on happy path")
			return "", errors.New("should not be called")
		},
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			t.Error("RunSync should not be called on happy path")
			return nil, errors.New("should not be called")
		},
	}

	ready, err := EnsureReady(context.Background(), cb)
	if err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if ready.Tenant != "acme" {
		t.Errorf("Tenant = %q, want acme", ready.Tenant)
	}
	if ready.Token != token {
		t.Error("token not propagated")
	}
	if _, ok := ready.Profiles["dev"]; !ok {
		t.Error("profiles not propagated")
	}
}

func TestEnsureReady_NoConfig_RunsInit(t *testing.T) {
	setupTestHome(t)

	token := makeJWT(t, time.Now().Add(1*time.Hour).Unix())
	var initCalled bool
	cb := Callbacks{
		TokenStore: fakeTokenStore{
			token:     token,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(token),
		},
		RunInit: func(_ context.Context) (*config.Config, error) {
			initCalled = true
			cfg := &config.Config{Tenant: "new-tenant"}
			if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
				return nil, err
			}
			if err := config.Save(cfg); err != nil {
				return nil, err
			}
			return cfg, nil
		},
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			return map[string]config.Profile{"p1": {Cloud: "aws"}}, nil
		},
	}

	if _, err := EnsureReady(context.Background(), cb); err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if !initCalled {
		t.Error("RunInit was not called when config was missing")
	}
}

func TestEnsureReady_ExpiredToken_RunsLogin(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	expired := makeJWT(t, time.Now().Add(-1*time.Hour).Unix())
	var loginCalled bool
	cb := Callbacks{
		TokenStore: fakeTokenStore{
			token:     expired,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(expired),
		},
		RunLogin: func(_ context.Context, tenant string) (string, error) {
			loginCalled = true
			if tenant != "acme" {
				t.Errorf("login tenant = %q, want acme", tenant)
			}
			return "fresh-token", nil
		},
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			return map[string]config.Profile{"p": {Cloud: "aws"}}, nil
		},
	}

	ready, err := EnsureReady(context.Background(), cb)
	if err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if !loginCalled {
		t.Error("RunLogin was not called for expired token")
	}
	if ready.Token != "fresh-token" {
		t.Errorf("token = %q, want fresh-token", ready.Token)
	}
}

func TestEnsureReady_StaleCache_RunsSync(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	token := makeJWT(t, time.Now().Add(1*time.Hour).Unix())

	// Seed a stale cache directly (bypassing SaveProfilesCache which
	// overwrites SyncedAt with time.Now).
	if err := os.MkdirAll(filepath.Dir(config.ProfilesCachePath()), 0o700); err != nil {
		t.Fatal(err)
	}
	staleJSON := []byte(`{
  "syncedAt": "2020-01-01T00:00:00Z",
  "profiles": {
    "old": {"cloud": "aws"}
  }
}`)
	if err := os.WriteFile(config.ProfilesCachePath(), staleJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	var syncCalled bool
	cb := Callbacks{
		TokenStore: fakeTokenStore{
			token:     token,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(token),
		},
		RunLogin: func(_ context.Context, _ string) (string, error) { return token, nil },
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			syncCalled = true
			return map[string]config.Profile{"new": {Cloud: "aws"}}, nil
		},
	}

	ready, err := EnsureReady(context.Background(), cb)
	if err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if !syncCalled {
		t.Error("RunSync was not called for stale cache")
	}
	if _, ok := ready.Profiles["new"]; !ok {
		t.Error("fresh profiles not returned")
	}
}

func TestEnsureReady_NoInitCallback_Errors(t *testing.T) {
	setupTestHome(t)

	_, err := EnsureReady(context.Background(), Callbacks{
		TokenStore: fakeTokenStore{},
	})
	if err == nil {
		t.Fatal("expected error when no callbacks provided and no config exists, got nil")
	}
}

func TestEnsureReady_MissingToken_RunsLogin(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	var loginCalled bool
	cb := Callbacks{
		TokenStore: fakeTokenStore{}, // empty token
		RunLogin: func(_ context.Context, _ string) (string, error) {
			loginCalled = true
			return "new", nil
		},
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			return map[string]config.Profile{"p": {Cloud: "aws"}}, nil
		},
	}

	if _, err := EnsureReady(context.Background(), cb); err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if !loginCalled {
		t.Error("RunLogin was not called for missing token")
	}
}

func TestEnsureReady_ExpiredTokenNoCallback_ReturnsSentinel(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	expired := makeJWT(t, time.Now().Add(-1*time.Hour).Unix())
	_, err := EnsureReady(context.Background(), Callbacks{
		TokenStore: fakeTokenStore{
			token:     expired,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(expired),
		},
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			return nil, errors.New("unreachable")
		},
	})
	if err == nil || !errors.Is(err, britive.ErrTokenExpired) {
		t.Errorf("err = %v, want wrapped ErrTokenExpired", err)
	}
}

func TestEnsureReady_MissingTokenNoCallback_ReturnsSentinel(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	_, err := EnsureReady(context.Background(), Callbacks{
		TokenStore: fakeTokenStore{},
	})
	if err == nil || !errors.Is(err, britive.ErrNotLoggedIn) {
		t.Errorf("err = %v, want wrapped ErrNotLoggedIn", err)
	}
}

// TestKeychainTokenStore exercises the thin adapter that wraps the real
// keychain. On platforms where the keychain is unavailable (Linux CI without
// dbus, etc.) the methods simply return errors/zero values, which is fine:
// we're verifying the wrapper compiles and delegates, not that the keychain
// itself works.
func TestKeychainTokenStore(t *testing.T) {
	store := keychainTokenStore{}
	tenant := "nonexistent-tenant-" + t.Name()
	// We don't assert on the return values -- the keychain may or may not
	// have the tenant cached. We just need to exercise the code paths.
	_, _ = store.GetToken(tenant)
	_ = store.GetTokenType(tenant)
	_ = store.GetTokenExpiry(tenant)
}

// TestLoadOrSyncProfiles_NoSyncNoCache exercises the fallback path where
// neither a cache nor a sync function is available. EnsureReady should
// return the in-config profiles instead of failing.
func TestLoadOrSyncProfiles_FallbackToConfigProfiles(t *testing.T) {
	setupTestHome(t)

	cfg := &config.Config{
		Tenant: "acme",
		Profiles: map[string]config.Profile{
			"legacy": {Cloud: "aws", BritivePath: "AWS/Legacy"},
		},
	}
	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	token := makeJWT(t, time.Now().Add(1*time.Hour).Unix())
	cb := Callbacks{
		TokenStore: fakeTokenStore{
			token:     token,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(token),
		},
		// No RunSync: should fall back to config.yaml profiles
	}

	ready, err := EnsureReady(context.Background(), cb)
	if err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if _, ok := ready.Profiles["legacy"]; !ok {
		t.Error("legacy profile from config not returned")
	}
}

// TestLoadOrSyncProfiles_NoSyncNoCacheNoProfiles tests the failure case
// where absolutely nothing is available.
func TestEnsureReady_NothingAvailable_Errors(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	token := makeJWT(t, time.Now().Add(1*time.Hour).Unix())
	_, err := EnsureReady(context.Background(), Callbacks{
		TokenStore: fakeTokenStore{
			token:     token,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(token),
		},
		// No RunSync and no cache file and no profiles in config
	})
	if err == nil {
		t.Fatal("expected error when neither cache nor sync nor config profiles exist, got nil")
	}
}

func TestEnsureReady_FreshCacheWithConfig_UsesCache(t *testing.T) {
	setupTestHome(t)

	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(&config.Config{Tenant: "acme"}); err != nil {
		t.Fatal(err)
	}

	cache := &config.ProfilesCache{
		SyncedAt: time.Now().UTC().Add(-1 * time.Minute),
		Profiles: map[string]config.Profile{"dev": {Cloud: "aws"}},
	}
	if err := config.SaveProfilesCache(cache); err != nil {
		t.Fatal(err)
	}

	token := makeJWT(t, time.Now().Add(1*time.Hour).Unix())
	var syncCalled bool
	cb := Callbacks{
		TokenStore: fakeTokenStore{
			token:     token,
			tokenType: "Bearer",
			expiry:    britive.JWTExpiry(token),
		},
		RunSync: func(_ context.Context, _, _ string) (map[string]config.Profile, error) {
			syncCalled = true
			return nil, errors.New("should not be called: cache is fresh")
		},
	}

	if _, err := EnsureReady(context.Background(), cb); err != nil {
		t.Fatalf("EnsureReady() error = %v", err)
	}
	if syncCalled {
		t.Error("RunSync should not have been called when cache is fresh")
	}
}
