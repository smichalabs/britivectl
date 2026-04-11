package britive

import (
	"errors"
	"fmt"
	"testing"
)

// Verify each sentinel error is a distinct, non-nil value.
func TestSentinelErrors_Distinct(t *testing.T) {
	sentinels := []error{
		ErrNotLoggedIn,
		ErrUnauthorized,
		ErrTokenExpired,
		ErrCheckoutTimeout,
		ErrAuthTimeout,
		ErrUnsupportedPlatform,
		ErrProfileNotFound,
	}
	for i, e := range sentinels {
		if e == nil {
			t.Errorf("sentinel[%d] is nil", i)
		}
		for j, other := range sentinels {
			if i != j && errors.Is(e, other) {
				t.Errorf("sentinel[%d] should not match sentinel[%d]", i, j)
			}
		}
	}
}

// Verify wrapped sentinel errors are detectable via errors.Is.
func TestSentinelErrors_WrappingDetection(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		sentinel error
	}{
		{"not logged in", fmt.Errorf("%w: run 'bctl login'", ErrNotLoggedIn), ErrNotLoggedIn},
		{"unauthorized", fmt.Errorf("%w: check your token", ErrUnauthorized), ErrUnauthorized},
		{"checkout timeout", fmt.Errorf("%w: %w", ErrCheckoutTimeout, errors.New("deadline exceeded")), ErrCheckoutTimeout},
		{"auth timeout", fmt.Errorf("%w: context canceled", ErrAuthTimeout), ErrAuthTimeout},
		{"unsupported platform", fmt.Errorf("%w: plan9", ErrUnsupportedPlatform), ErrUnsupportedPlatform},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, tc.sentinel) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tc.err, tc.sentinel)
			}
		})
	}
}
