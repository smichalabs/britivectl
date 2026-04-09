package config_test

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/smichalabs/britivectl/internal/config"
)

func skipIfNotDarwin(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("keychain tests require macOS")
	}
}

func uniqueTenant() string {
	return fmt.Sprintf("bctl-test-%d", time.Now().UnixNano())
}

func TestSetGetDeleteToken(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	t.Cleanup(func() {
		_ = config.DeleteToken(tenant)
	})

	const token = "my-secret-token"

	if err := config.SetToken(tenant, token); err != nil {
		t.Fatalf("SetToken() error: %v", err)
	}

	got, err := config.GetToken(tenant)
	if err != nil {
		t.Fatalf("GetToken() after set, error: %v", err)
	}
	if got != token {
		t.Errorf("GetToken() = %q, want %q", got, token)
	}

	if err := config.DeleteToken(tenant); err != nil {
		t.Fatalf("DeleteToken() error: %v", err)
	}

	got, err = config.GetToken(tenant)
	if err == nil && got != "" {
		t.Errorf("GetToken() after delete = %q, want error or empty", got)
	}
}

func TestGetToken_NotSet(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	// No cleanup needed — nothing was set.

	got, err := config.GetToken(tenant)
	if err == nil {
		t.Errorf("GetToken() for unset tenant returned no error, got %q", got)
	}
}

func TestSetGetDeleteTokenType(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	t.Cleanup(func() {
		_ = config.DeleteTokenType(tenant)
	})

	const tokenType = "Bearer"

	if err := config.SetTokenType(tenant, tokenType); err != nil {
		t.Fatalf("SetTokenType() error: %v", err)
	}

	got := config.GetTokenType(tenant)
	if got != tokenType {
		t.Errorf("GetTokenType() = %q, want %q", got, tokenType)
	}

	if err := config.DeleteTokenType(tenant); err != nil {
		t.Fatalf("DeleteTokenType() error: %v", err)
	}

	got = config.GetTokenType(tenant)
	if got != "TOKEN" {
		t.Errorf("GetTokenType() after delete = %q, want default %q", got, "TOKEN")
	}
}

func TestGetTokenType_Default(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	// Nothing set — expect the default.

	got := config.GetTokenType(tenant)
	if got != "TOKEN" {
		t.Errorf("GetTokenType() for unset tenant = %q, want default %q", got, "TOKEN")
	}
}

func TestSetGetDeleteTokenExpiry(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	t.Cleanup(func() {
		_ = config.DeleteTokenExpiry(tenant)
	})

	const expiry int64 = 1234567890

	if err := config.SetTokenExpiry(tenant, expiry); err != nil {
		t.Fatalf("SetTokenExpiry() error: %v", err)
	}

	got := config.GetTokenExpiry(tenant)
	if got != expiry {
		t.Errorf("GetTokenExpiry() = %d, want %d", got, expiry)
	}

	if err := config.DeleteTokenExpiry(tenant); err != nil {
		t.Fatalf("DeleteTokenExpiry() error: %v", err)
	}

	got = config.GetTokenExpiry(tenant)
	if got != 0 {
		t.Errorf("GetTokenExpiry() after delete = %d, want 0", got)
	}
}

func TestGetTokenExpiry_NotSet(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	// Nothing set — expect zero.

	got := config.GetTokenExpiry(tenant)
	if got != 0 {
		t.Errorf("GetTokenExpiry() for unset tenant = %d, want 0", got)
	}
}

func TestTokenExpiry_ZeroValue(t *testing.T) {
	skipIfNotDarwin(t)
	tenant := uniqueTenant()
	t.Cleanup(func() {
		_ = config.DeleteTokenExpiry(tenant)
	})

	// Setting expiry to 1 (smallest non-zero value) and verifying round-trip.
	const expiry int64 = 1

	if err := config.SetTokenExpiry(tenant, expiry); err != nil {
		t.Fatalf("SetTokenExpiry(1) error: %v", err)
	}

	got := config.GetTokenExpiry(tenant)
	if got != expiry {
		t.Errorf("GetTokenExpiry() = %d, want %d", got, expiry)
	}
}
