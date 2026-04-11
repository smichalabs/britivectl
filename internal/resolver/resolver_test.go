package resolver

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/smichalabs/britivectl/internal/config"
)

func sampleProfiles() map[string]config.Profile {
	return map[string]config.Profile{
		"dev":            {BritivePath: "AWS/Sandbox/Developer", Cloud: "aws"},
		"prod":           {BritivePath: "AWS/Prod/ReadOnly", Cloud: "aws"},
		"gcp-prod":       {BritivePath: "GCP/prod/readonly", Cloud: "gcp"},
		"azure-contrib":  {BritivePath: "Azure/Sub/Contributor", Cloud: "azure"},
	}
}

func TestResolve_ExactMatch(t *testing.T) {
	m, err := Resolve(context.Background(), sampleProfiles(), "dev", bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if m.Alias != "dev" {
		t.Errorf("alias = %q, want dev", m.Alias)
	}
}

func TestResolve_SubstringMatchSingle(t *testing.T) {
	// "contrib" only matches the azure profile
	m, err := Resolve(context.Background(), sampleProfiles(), "contrib", bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if m.Alias != "azure-contrib" {
		t.Errorf("alias = %q, want azure-contrib", m.Alias)
	}
}

func TestResolve_SubstringMatchOnPath(t *testing.T) {
	// "Sandbox" only appears in the britive_path of the dev profile
	m, err := Resolve(context.Background(), sampleProfiles(), "Sandbox", bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if m.Alias != "dev" {
		t.Errorf("alias = %q, want dev", m.Alias)
	}
}

func TestResolve_FuzzyMatchFallback(t *testing.T) {
	// "gcd" matches "gcp-prod" via subsequence (g, c, <...> d) when no
	// substring matches
	profiles := map[string]config.Profile{
		"gcp-prod": {BritivePath: "x", Cloud: "gcp"},
	}
	m, err := Resolve(context.Background(), profiles, "gcd", bytes.NewReader(nil), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if m.Alias != "gcp-prod" {
		t.Errorf("alias = %q, want gcp-prod", m.Alias)
	}
}

func TestResolve_NoMatchErrors(t *testing.T) {
	_, err := Resolve(context.Background(), sampleProfiles(), "nonexistent-xyz", bytes.NewReader(nil), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected ErrNoMatch, got nil")
	}
	if !errors.Is(err, ErrNoMatch) {
		t.Errorf("errors.Is(err, ErrNoMatch) = false, want true; err = %v", err)
	}
}

func TestResolve_Ambiguous_UsesPicker(t *testing.T) {
	// "prod" matches both "prod" (alias) and "gcp-prod" (alias). But "prod"
	// is also an exact match for the "prod" alias -> wins.
	// Use "pro" to force ambiguity without an exact match.
	m, err := Resolve(context.Background(), sampleProfiles(), "pro", strings.NewReader("1\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	// Matches should be sorted alphabetically by alias: gcp-prod, prod
	if m.Alias != "gcp-prod" {
		t.Errorf("alias = %q, want gcp-prod (first in sorted picker)", m.Alias)
	}
}

func TestResolve_Picker_SecondChoice(t *testing.T) {
	m, err := Resolve(context.Background(), sampleProfiles(), "pro", strings.NewReader("2\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if m.Alias != "prod" {
		t.Errorf("alias = %q, want prod", m.Alias)
	}
}

func TestResolve_Picker_InvalidSelection(t *testing.T) {
	_, err := Resolve(context.Background(), sampleProfiles(), "pro", strings.NewReader("99\n"), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for out-of-range selection, got nil")
	}
}

func TestResolve_Picker_Canceled(t *testing.T) {
	_, err := Resolve(context.Background(), sampleProfiles(), "pro", strings.NewReader("q\n"), &bytes.Buffer{})
	if !errors.Is(err, ErrCanceled) {
		t.Errorf("err = %v, want ErrCanceled", err)
	}
}

func TestResolve_EmptyQuery_PickerOverAll(t *testing.T) {
	// Empty query with multiple profiles -> picker shows them all
	m, err := Resolve(context.Background(), sampleProfiles(), "", strings.NewReader("1\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if m.Alias != "azure-contrib" {
		t.Errorf("alias = %q, want azure-contrib (first alphabetically)", m.Alias)
	}
}

func TestResolve_NoProfiles(t *testing.T) {
	_, err := Resolve(context.Background(), nil, "", bytes.NewReader(nil), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for empty profile map, got nil")
	}
}

func TestIsSubsequence(t *testing.T) {
	cases := []struct {
		needle, haystack string
		want             bool
	}{
		{"", "anything", true},
		{"abc", "abc", true},
		{"abc", "aXbYcZ", true},
		{"abc", "acb", false},
		{"abc", "ab", false},
	}
	for _, tc := range cases {
		got := isSubsequence(tc.needle, tc.haystack)
		if got != tc.want {
			t.Errorf("isSubsequence(%q, %q) = %v, want %v", tc.needle, tc.haystack, got, tc.want)
		}
	}
}
