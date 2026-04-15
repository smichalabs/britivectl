package cmd

import (
	"testing"
	"time"

	"github.com/smichalabs/britivectl/internal/config"
)

func TestShouldSyncForList(t *testing.T) {
	freshCache := &config.ProfilesCache{
		SyncedAt: time.Now().Add(-5 * time.Minute),
		Profiles: map[string]config.Profile{"dev": {}},
	}
	staleCache := &config.ProfilesCache{
		SyncedAt: time.Now().Add(-2 * time.Hour),
		Profiles: map[string]config.Profile{"dev": {}},
	}
	emptyCache := &config.ProfilesCache{
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
		{"no cache, default -> sync", nil, false, false, true},
		{"no cache, --no-sync -> skip", nil, false, true, false},
		{"fresh cache, default -> skip", freshCache, false, false, false},
		{"fresh cache, --refresh -> sync", freshCache, true, false, true},
		{"fresh cache, --no-sync -> skip", freshCache, false, true, false},
		{"stale cache, default -> sync", staleCache, false, false, true},
		{"stale cache, --no-sync wins", staleCache, false, true, false},
		{"empty cache, default -> sync", emptyCache, false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSyncForList(tc.cache, tc.refresh, tc.noSync)
			if got != tc.want {
				t.Errorf("shouldSyncForList() = %v, want %v", got, tc.want)
			}
		})
	}
}
