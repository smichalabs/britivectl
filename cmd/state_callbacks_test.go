package cmd

import (
	"testing"

	"github.com/smichalabs/britivectl/internal/britive"
)

func TestPickAlias_ThreeTierCollision(t *testing.T) {
	entries := []britive.AccessEntry{
		{AppName: "Alpha", ProfileName: "Admin", EnvironmentName: "Prod", ProfileID: "p1", EnvironmentID: "e1"},
		{AppName: "Beta", ProfileName: "Admin", EnvironmentName: "Prod", ProfileID: "p2", EnvironmentID: "e2"},
		{AppName: "Gamma", ProfileName: "Admin", EnvironmentName: "Prod", ProfileID: "p3", EnvironmentID: "e3"},
		{AppName: "Delta", ProfileName: "ReadOnly", EnvironmentName: "Dev", ProfileID: "p4", EnvironmentID: "e4"},
		{AppName: "Delta", ProfileName: "ReadOnly", EnvironmentName: "Staging", ProfileID: "p5", EnvironmentID: "e5"},
	}

	got := buildProfileMap(entries)

	if len(got) != len(entries) {
		t.Fatalf("expected %d profiles after dedup, got %d: %v", len(entries), len(got), mapKeys(got))
	}

	want := []string{
		"admin",                     // Alpha, tier 1
		"admin-prod",                // Beta, tier 2
		"gamma-admin-prod",          // Gamma, tier 3
		"readonly",                  // Delta/Dev, tier 1
		"readonly-staging",          // Delta/Staging, tier 2 (Dev already took ReadOnly)
	}
	for _, alias := range want {
		if _, ok := got[alias]; !ok {
			t.Errorf("expected alias %q in map, keys were %v", alias, mapKeys(got))
		}
	}
}

func TestPickAlias_NumericFallback(t *testing.T) {
	// Two entries that sanitize to identical AppName+ProfileName+EnvironmentName
	// should still both land in the map via a numeric suffix rather than one
	// silently overwriting the other.
	entries := []britive.AccessEntry{
		{AppName: "A", ProfileName: "X", EnvironmentName: "Y", ProfileID: "p1", EnvironmentID: "e1"},
		{AppName: "A", ProfileName: "X", EnvironmentName: "Y", ProfileID: "p2", EnvironmentID: "e2"},
	}

	got := buildProfileMap(entries)

	if len(got) != 2 {
		t.Fatalf("expected both entries to survive, got %d: %v", len(got), mapKeys(got))
	}
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
