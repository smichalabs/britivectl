package aliases_test

import (
	"testing"

	"github.com/smichalabs/britivectl/internal/aliases"
	"github.com/smichalabs/britivectl/internal/britive"
)

func TestSanitize(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Admin", "admin"},
		{"aws-admin-prod", "aws-admin-prod"},
		{"Some Profile Name", "some-profile-name"},
		{"App/Env/Profile", "app-env-profile"},
		{"v1.2.3", "v1-2-3"},
		{"weird!@#chars", "weirdchars"},
		{"--leading-and-trailing--", "leading-and-trailing"},
		{"", ""},
	}
	for _, c := range cases {
		if got := aliases.Sanitize(c.in); got != c.want {
			t.Errorf("Sanitize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildMap_FourTierCollision(t *testing.T) {
	entries := []britive.AccessEntry{
		{AppName: "Alpha", ProfileName: "Admin", EnvironmentName: "Prod", ProfileID: "p1", EnvironmentID: "e1"},
		{AppName: "Beta", ProfileName: "Admin", EnvironmentName: "Prod", ProfileID: "p2", EnvironmentID: "e2"},
		{AppName: "Gamma", ProfileName: "Admin", EnvironmentName: "Prod", ProfileID: "p3", EnvironmentID: "e3"},
		{AppName: "Delta", ProfileName: "ReadOnly", EnvironmentName: "Dev", ProfileID: "p4", EnvironmentID: "e4"},
		{AppName: "Delta", ProfileName: "ReadOnly", EnvironmentName: "Staging", ProfileID: "p5", EnvironmentID: "e5"},
	}

	got := aliases.BuildMap(entries)

	if len(got) != len(entries) {
		t.Fatalf("expected %d profiles after dedup, got %d: %v", len(entries), len(got), mapKeys(got))
	}

	want := []string{
		"admin",             // Alpha, tier 1
		"admin-prod",        // Beta, tier 2
		"gamma-admin-prod",  // Gamma, tier 3
		"readonly",          // Delta/Dev, tier 1
		"readonly-staging", // Delta/Staging, tier 2
	}
	for _, alias := range want {
		if _, ok := got[alias]; !ok {
			t.Errorf("expected alias %q in map, keys were %v", alias, mapKeys(got))
		}
	}
}

func TestBuildMap_NumericFallback(t *testing.T) {
	entries := []britive.AccessEntry{
		{AppName: "A", ProfileName: "X", EnvironmentName: "Y", ProfileID: "p1", EnvironmentID: "e1"},
		{AppName: "A", ProfileName: "X", EnvironmentName: "Y", ProfileID: "p2", EnvironmentID: "e2"},
	}

	got := aliases.BuildMap(entries)

	if len(got) != 2 {
		t.Fatalf("expected both entries to survive via numeric fallback, got %d: %v", len(got), mapKeys(got))
	}
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
