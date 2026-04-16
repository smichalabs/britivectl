// Package aliases derives shell-friendly, collision-free profile aliases from
// Britive access entries. It is the single source of truth used by both the
// interactive `bctl profiles sync` command and the automatic reconciliation
// flow inside `bctl checkout`, so both code paths produce identical maps.
package aliases

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
)

// Sanitize converts a profile-path-style string into a shell-friendly alias.
// The rules mirror what users can reasonably type on a command line and what
// persists cleanly into YAML: lowercase letters, digits, hyphen, underscore.
// Spaces, slashes, and dots collapse to hyphens; everything else is dropped.
// Leading and trailing hyphens are trimmed.
func Sanitize(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '/' || r == '.':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// BuildMap flattens a slice of Britive access entries into a stable
// alias -> Profile map. Collisions are resolved by Pick.
func BuildMap(entries []britive.AccessEntry) map[string]config.Profile {
	profiles := make(map[string]config.Profile, len(entries))
	for _, e := range entries {
		alias := Pick(profiles, e)
		profiles[alias] = config.Profile{
			ProfileID:     e.ProfileID,
			EnvironmentID: e.EnvironmentID,
			BritivePath:   e.AppName + "/" + e.EnvironmentName + "/" + e.ProfileName,
			Cloud:         e.Cloud,
		}
	}
	return profiles
}

// Pick returns the shortest alias for entry e that does not collide with any
// existing entry in the map. It walks a four-tier strategy:
//
//  1. ProfileName                            -- most concise, preferred when unique
//  2. ProfileName-EnvironmentName            -- disambiguate same profile across envs
//  3. AppName-ProfileName-EnvironmentName    -- disambiguate across applications
//  4. <tier 3>-N                             -- numeric suffix as a last resort
//
// Before this function existed, collision handling only fell back to
// ProfileName-EnvironmentName and never considered AppName, so two apps with
// the same profile name and environment silently overwrote each other.
func Pick(existing map[string]config.Profile, e britive.AccessEntry) string {
	candidates := []string{
		e.ProfileName,
		e.ProfileName + "-" + e.EnvironmentName,
		e.AppName + "-" + e.ProfileName + "-" + e.EnvironmentName,
	}
	for _, raw := range candidates {
		alias := Sanitize(raw)
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
	base := Sanitize(e.AppName + "-" + e.ProfileName + "-" + e.EnvironmentName)
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
