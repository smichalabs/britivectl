package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/99designs/keyring"
)

const (
	keychainService     = "bctl"
	credentialsDirName  = "credentials"
	filePassphraseConst = "bctl-file-backend"
)

// openKeyring opens the OS keychain (or a file fallback on systems without
// one). Backends are tried in priority order: native OS stores first, then a
// file-based fallback so headless Linux and WSL still work without a desktop
// session. Setting BCTL_KEYRING_BACKEND=file forces the file backend.
func openKeyring() (keyring.Keyring, error) {
	fileDir := filepath.Join(xdgConfigDir(), credentialsDirName)
	if err := os.MkdirAll(fileDir, 0o700); err != nil {
		return nil, fmt.Errorf("creating credentials dir: %w", err)
	}

	allowed := []keyring.BackendType{
		keyring.KeychainBackend,
		keyring.WinCredBackend,
		keyring.SecretServiceBackend,
		keyring.KWalletBackend,
		keyring.FileBackend,
	}
	if os.Getenv("BCTL_KEYRING_BACKEND") == "file" {
		allowed = []keyring.BackendType{keyring.FileBackend}
	}

	return keyring.Open(keyring.Config{
		ServiceName:              keychainService,
		AllowedBackends:          allowed,
		KeychainTrustApplication: true,
		KeychainSynchronizable:   false,
		LibSecretCollectionName:  keychainService,
		KWalletAppID:             keychainService,
		KWalletFolder:            keychainService,
		WinCredPrefix:            keychainService,
		FileDir:                  fileDir,
		FilePasswordFunc:         keyring.FixedStringPrompt(filePassphraseConst),
	})
}

func keyName(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}

// SetToken stores the auth token for the given tenant.
func SetToken(tenant, token string) error {
	r, err := openKeyring()
	if err != nil {
		return err
	}
	return r.Set(keyring.Item{
		Key:   keyName(tenant, "token"),
		Data:  []byte(token),
		Label: keychainService + ": token (" + tenant + ")",
	})
}

// GetToken retrieves the auth token for the given tenant.
func GetToken(tenant string) (string, error) {
	r, err := openKeyring()
	if err != nil {
		return "", err
	}
	item, err := r.Get(keyName(tenant, "token"))
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

// DeleteToken removes the auth token for the given tenant.
func DeleteToken(tenant string) error {
	r, err := openKeyring()
	if err != nil {
		return err
	}
	if err := r.Remove(keyName(tenant, "token")); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}
	return nil
}

// SetTokenType stores the token type ("TOKEN" or "Bearer").
func SetTokenType(tenant, tokenType string) error {
	r, err := openKeyring()
	if err != nil {
		return err
	}
	return r.Set(keyring.Item{
		Key:   keyName(tenant, "type"),
		Data:  []byte(tokenType),
		Label: keychainService + ": token type (" + tenant + ")",
	})
}

// GetTokenType retrieves the token type for the given tenant.
// Defaults to "TOKEN" if not set (backwards compatibility with API tokens).
func GetTokenType(tenant string) string {
	r, err := openKeyring()
	if err != nil {
		return "TOKEN"
	}
	item, err := r.Get(keyName(tenant, "type"))
	if err != nil || len(item.Data) == 0 {
		return "TOKEN"
	}
	return string(item.Data)
}

// DeleteTokenType removes the token type entry.
func DeleteTokenType(tenant string) error {
	r, err := openKeyring()
	if err != nil {
		return err
	}
	if err := r.Remove(keyName(tenant, "type")); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}
	return nil
}

// SetTokenExpiry stores the token expiry as a Unix timestamp string.
func SetTokenExpiry(tenant string, unixSecs int64) error {
	r, err := openKeyring()
	if err != nil {
		return err
	}
	return r.Set(keyring.Item{
		Key:   keyName(tenant, "expiry"),
		Data:  []byte(fmt.Sprintf("%d", unixSecs)),
		Label: keychainService + ": token expiry (" + tenant + ")",
	})
}

// GetTokenExpiry returns the stored token expiry as a Unix timestamp, or 0 if not set.
func GetTokenExpiry(tenant string) int64 {
	r, err := openKeyring()
	if err != nil {
		return 0
	}
	item, err := r.Get(keyName(tenant, "expiry"))
	if err != nil || len(item.Data) == 0 {
		return 0
	}
	var ts int64
	if _, err := fmt.Sscan(string(item.Data), &ts); err != nil {
		return 0
	}
	return ts
}

// DeleteTokenExpiry removes the expiry entry.
func DeleteTokenExpiry(tenant string) error {
	r, err := openKeyring()
	if err != nil {
		return err
	}
	if err := r.Remove(keyName(tenant, "expiry")); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return err
	}
	return nil
}
