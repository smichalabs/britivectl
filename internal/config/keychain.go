package config

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const keychainService = "bctl"

// SetToken stores the auth token for the given tenant in the OS keychain.
func SetToken(tenant, token string) error {
	return keyring.Set(keychainService, tenant, token)
}

// GetToken retrieves the auth token for the given tenant from the OS keychain.
func GetToken(tenant string) (string, error) {
	return keyring.Get(keychainService, tenant)
}

// DeleteToken removes the auth token for the given tenant from the OS keychain.
func DeleteToken(tenant string) error {
	return keyring.Delete(keychainService, tenant)
}

// SetTokenType stores the token type ("TOKEN" or "Bearer") in the OS keychain.
func SetTokenType(tenant, tokenType string) error {
	return keyring.Set(keychainService, tenant+":type", tokenType)
}

// GetTokenType retrieves the token type for the given tenant.
// Defaults to "TOKEN" if not set (backwards compatibility with API tokens).
func GetTokenType(tenant string) string {
	t, err := keyring.Get(keychainService, tenant+":type")
	if err != nil || t == "" {
		return "TOKEN"
	}
	return t
}

// DeleteTokenType removes the token type entry from the OS keychain.
func DeleteTokenType(tenant string) error {
	return keyring.Delete(keychainService, tenant+":type")
}

// SetTokenExpiry stores the token expiry as a Unix timestamp string.
func SetTokenExpiry(tenant string, unixSecs int64) error {
	return keyring.Set(keychainService, tenant+":expiry", fmt.Sprintf("%d", unixSecs))
}

// GetTokenExpiry returns the stored token expiry as a Unix timestamp, or 0 if not set.
func GetTokenExpiry(tenant string) int64 {
	s, err := keyring.Get(keychainService, tenant+":expiry")
	if err != nil || s == "" {
		return 0
	}
	var ts int64
	_, _ = fmt.Sscan(s, &ts)
	return ts
}

// DeleteTokenExpiry removes the expiry entry from the OS keychain.
func DeleteTokenExpiry(tenant string) error {
	return keyring.Delete(keychainService, tenant+":expiry")
}
