// Package state reconciles the local bctl state before a command runs.
//
// EnsureReady is the single entry point: it checks config, auth token, and
// profile cache, and runs Init / Login / Sync inline as needed so that the
// caller (typically `bctl checkout`) can proceed without asking the user to
// run separate commands.
package state

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
)

// CacheMaxAge is how long a synced profile cache is considered fresh.
// After this, EnsureReady will refresh the cache on its own.
const CacheMaxAge = 24 * time.Hour

// Ready is the reconciled state returned by EnsureReady.
type Ready struct {
	Tenant   string
	Token    string
	Profiles map[string]config.Profile
}

// Callbacks lets the caller supply the interactive flows. This keeps the
// state package independent of cmd/ and avoids import cycles.
//
// Each callback is invoked only when the corresponding precondition is
// missing or stale. Callers that want non-interactive behavior (e.g. tests,
// CI) can pass functions that return errors immediately.
type Callbacks struct {
	// RunInit is called when no config file exists. It should populate the
	// config file (tenant at minimum) and return the loaded config.
	RunInit func(ctx context.Context) (*config.Config, error)

	// RunLogin is called when the token is missing or expired. It should
	// obtain a fresh token, store it, and return the stored token.
	RunLogin func(ctx context.Context, tenant string) (string, error)

	// RunSync is called when the profile cache is missing or stale. It
	// should fetch the latest profiles from the Britive API and persist
	// them to the profile cache.
	RunSync func(ctx context.Context, tenant, token string) (map[string]config.Profile, error)

	// TokenStore reads stored tokens for a tenant. If nil, the real OS
	// keychain is used via the config package. Tests can inject a mock
	// implementation so they do not depend on the host keychain.
	TokenStore TokenStore
}

// TokenStore abstracts the credential storage backend so the state package
// can be tested without the OS keychain.
type TokenStore interface {
	GetToken(tenant string) (string, error)
	GetTokenType(tenant string) string
	GetTokenExpiry(tenant string) int64
}

// keychainTokenStore adapts the real keychain-backed config helpers to the
// TokenStore interface.
type keychainTokenStore struct{}

func (keychainTokenStore) GetToken(tenant string) (string, error) {
	return config.GetToken(tenant)
}

func (keychainTokenStore) GetTokenType(tenant string) string {
	return config.GetTokenType(tenant)
}

func (keychainTokenStore) GetTokenExpiry(tenant string) int64 {
	return config.GetTokenExpiry(tenant)
}

// EnsureReady reconciles the local state and returns a Ready snapshot.
// The happy path (all state present and fresh) is <10ms: three os.Stat calls
// and a JWT exp check.
//
// Reconciliation order:
//  1. Migrate ~/.bctl -> XDG paths (one-time, idempotent)
//  2. Load config; run RunInit if missing/empty tenant
//  3. Load token from keychain; run RunLogin if missing
//  4. Decode JWT exp; run RunLogin if expired
//  5. Load profile cache; run RunSync if missing or older than CacheMaxAge
func EnsureReady(ctx context.Context, cb Callbacks) (*Ready, error) {
	// Step 1: one-time XDG migration.
	if _, err := config.MigrateLegacyDir(); err != nil {
		// Migration is best-effort -- fall through and try to continue.
		// We log by returning a warning in the message if this bites us later.
		_ = err
	}
	if err := config.EnsureXDGDirs(); err != nil {
		return nil, fmt.Errorf("ensuring XDG dirs: %w", err)
	}

	// Step 2: config + tenant.
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg == nil || cfg.Tenant == "" {
		if cb.RunInit == nil {
			return nil, errors.New("bctl is not configured: run 'bctl init'")
		}
		cfg, err = cb.RunInit(ctx)
		if err != nil {
			return nil, fmt.Errorf("init failed: %w", err)
		}
		if cfg == nil || cfg.Tenant == "" {
			return nil, errors.New("init did not set a tenant")
		}
	}

	// Step 3 + 4: token + expiry.
	store := cb.TokenStore
	if store == nil {
		store = keychainTokenStore{}
	}
	token, err := requireValidToken(ctx, cfg.Tenant, store, cb.RunLogin)
	if err != nil {
		return nil, err
	}

	// Step 5: profile cache. We prefer the on-disk cache file (written by
	// sync). If that's missing/stale, run sync. If the user has profiles in
	// their config.yaml but no cache file (e.g. upgraded from an older
	// version), seed the cache from config.
	profiles, err := loadOrSyncProfiles(ctx, cfg, token, cb.RunSync)
	if err != nil {
		return nil, err
	}

	return &Ready{
		Tenant:   cfg.Tenant,
		Token:    token,
		Profiles: profiles,
	}, nil
}

// requireValidToken loads the stored token for tenant and validates its
// expiry. If missing or expired, it runs RunLogin to obtain a fresh one.
func requireValidToken(ctx context.Context, tenant string, store TokenStore, runLogin func(context.Context, string) (string, error)) (string, error) {
	token, err := store.GetToken(tenant)
	if err != nil || token == "" {
		if runLogin == nil {
			return "", fmt.Errorf("%w: run 'bctl login'", britive.ErrNotLoggedIn)
		}
		return runLogin(ctx, tenant)
	}

	// Bearer tokens have a JWT exp claim we can decode locally. API tokens
	// don't (they're opaque strings); we only check Bearer expiry.
	if store.GetTokenType(tenant) == "Bearer" {
		storedExp := store.GetTokenExpiry(tenant)
		if storedExp == 0 {
			if jwtExp := britive.JWTExpiry(token); jwtExp > 0 {
				storedExp = jwtExp
			}
		}
		if storedExp > 0 && time.Now().Unix() >= storedExp {
			if runLogin == nil {
				return "", fmt.Errorf("%w: run 'bctl login'", britive.ErrTokenExpired)
			}
			return runLogin(ctx, tenant)
		}
	}

	return token, nil
}

// loadOrSyncProfiles returns the freshest set of profiles, running sync if
// the cache is missing or stale.
func loadOrSyncProfiles(ctx context.Context, cfg *config.Config, token string, runSync func(context.Context, string, string) (map[string]config.Profile, error)) (map[string]config.Profile, error) {
	cache, err := config.LoadProfilesCache()
	if err != nil && !errors.Is(err, config.ErrCacheMiss) {
		return nil, fmt.Errorf("loading profiles cache: %w", err)
	}

	if cache != nil && !cache.IsStale(CacheMaxAge) && len(cache.Profiles) > 0 {
		return cache.Profiles, nil
	}

	// Fall back: if sync is unavailable (non-interactive path), use
	// whatever is in config.yaml, even if empty.
	if runSync == nil {
		if len(cfg.Profiles) > 0 {
			return cfg.Profiles, nil
		}
		return nil, errors.New("no profile cache and no sync function: run 'bctl profiles sync'")
	}

	profiles, err := runSync(ctx, cfg.Tenant, token)
	if err != nil {
		return nil, fmt.Errorf("sync failed: %w", err)
	}
	return profiles, nil
}
