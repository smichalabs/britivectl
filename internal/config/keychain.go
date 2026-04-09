package config

import (
	"github.com/zalando/go-keyring"
)

const keychainService = "bctl"

// SetToken stores the API token for the given tenant in the OS keychain.
func SetToken(tenant, token string) error {
	return keyring.Set(keychainService, tenant, token)
}

// GetToken retrieves the API token for the given tenant from the OS keychain.
func GetToken(tenant string) (string, error) {
	return keyring.Get(keychainService, tenant)
}

// DeleteToken removes the API token for the given tenant from the OS keychain.
func DeleteToken(tenant string) error {
	return keyring.Delete(keychainService, tenant)
}
