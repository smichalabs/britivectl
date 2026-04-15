package cmd

import (
	"context"
	"fmt"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/state"
)

// stateCallbacks wires the interactive init/login/sync flows to the state
// package's Callbacks interface. It is passed to state.EnsureReady so that
// `bctl checkout` can auto-recover from missing or stale state.
func stateCallbacks() state.Callbacks {
	return state.Callbacks{
		RunInit:  initCallback,
		RunLogin: loginCallback,
		RunSync:  syncCallback,
	}
}

// initCallback runs the existing interactive init wizard and returns the
// resulting config. Called by EnsureReady when the config file is missing or
// the tenant is unset.
func initCallback(_ context.Context) (*config.Config, error) {
	output.Info("No configuration found -- running 'bctl init' first.")
	if err := runInit(); err != nil {
		return nil, err
	}
	return config.Load()
}

// loginCallback runs the existing login flow using the stored auth method
// (browser SSO or API token) and returns the fresh token. Called by
// EnsureReady when the token is missing or the JWT exp claim is in the past.
func loginCallback(ctx context.Context, tenant string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", fmt.Errorf("loading config during login: %w", err)
	}

	method := cfg.Auth.Method
	if method == "" {
		method = "browser"
	}

	switch method {
	case "browser":
		output.Info("Session expired or missing -- launching browser login...")
		token, err := britive.AuthWithBrowser(ctx, tenant)
		if err != nil {
			return "", fmt.Errorf("browser login failed: %w", err)
		}
		if err := persistToken(tenant, token, "Bearer"); err != nil {
			return "", err
		}
		return token, nil

	case "token":
		// With an API token method, we cannot silently re-auth because the
		// token comes from the user. Tell them explicitly.
		return "", fmt.Errorf("api token missing or invalid: run 'bctl login --token <t>'")

	default:
		return "", fmt.Errorf("unknown auth method %q", method)
	}
}

// syncCallback runs the existing profile sync logic and writes the cache.
// Called by EnsureReady when the profile cache is missing or older than
// state.CacheMaxAge.
func syncCallback(ctx context.Context, tenant, token string) (map[string]config.Profile, error) {
	output.Info("Profile cache is stale -- syncing from Britive API...")

	client := newAPIClient(tenant, token)
	entries, err := client.ListAccess(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing access: %w", err)
	}

	profiles := buildProfileMap(entries)
	cache := &config.ProfilesCache{Profiles: profiles}
	if err := config.SaveProfilesCache(cache); err != nil {
		return nil, fmt.Errorf("saving profile cache: %w", err)
	}
	return profiles, nil
}

// persistToken writes a freshly obtained token (and its expiry, if JWT) to
// the OS keychain so that subsequent bctl invocations pick it up.
func persistToken(tenant, token, tokenType string) error {
	if err := config.SetToken(tenant, token); err != nil {
		return fmt.Errorf("storing token: %w", err)
	}
	if err := config.SetTokenType(tenant, tokenType); err != nil {
		return fmt.Errorf("storing token type: %w", err)
	}
	if exp := britive.JWTExpiry(token); exp > 0 {
		_ = config.SetTokenExpiry(tenant, exp)
	}
	return nil
}

// buildProfileMap flattens the Britive API response into alias -> Profile.
// Uses the same alias-generation rules as the interactive `profiles sync`
// command so both code paths produce identical results.
func buildProfileMap(entries []britive.AccessEntry) map[string]config.Profile {
	profiles := make(map[string]config.Profile, len(entries))
	for _, e := range entries {
		alias := pickAlias(profiles, e)
		profiles[alias] = config.Profile{
			ProfileID:     e.ProfileID,
			EnvironmentID: e.EnvironmentID,
			BritivePath:   e.AppName + "/" + e.EnvironmentName + "/" + e.ProfileName,
			Cloud:         e.Cloud,
		}
	}
	return profiles
}

// pickAlias picks the shortest alias that does not collide with an existing
// entry, walking a four-tier strategy:
//
//  1. ProfileName                            -- most concise, preferred when unique
//  2. ProfileName-EnvironmentName            -- disambiguate same profile across envs
//  3. AppName-ProfileName-EnvironmentName    -- disambiguate across applications
//  4. <tier 3>-N                             -- numeric suffix as a last resort
//
// Before this change, two apps with the same ProfileName and EnvironmentName
// (e.g. "admin" in "production") would silently overwrite each other in the
// map because collision handling only fell back to ProfileName+Environment and
// never considered AppName. Now every profile keeps its own slot.
func pickAlias(existing map[string]config.Profile, e britive.AccessEntry) string {
	candidates := []string{
		e.ProfileName,
		e.ProfileName + "-" + e.EnvironmentName,
		e.AppName + "-" + e.ProfileName + "-" + e.EnvironmentName,
	}
	for _, raw := range candidates {
		alias := sanitizeAlias(raw)
		if alias == "" {
			continue
		}
		if _, clash := existing[alias]; !clash {
			return alias
		}
	}

	// Tier 4: numeric suffix. Should be unreachable in practice since AppName
	// + ProfileName + EnvironmentName is unique per Britive access tuple, but
	// defend against API oddities so we never silently overwrite.
	base := sanitizeAlias(e.AppName + "-" + e.ProfileName + "-" + e.EnvironmentName)
	if base == "" {
		base = "profile"
	}
	for i := 2; ; i++ {
		alias := fmt.Sprintf("%s-%d", base, i)
		if _, clash := existing[alias]; !clash {
			return alias
		}
	}
}
