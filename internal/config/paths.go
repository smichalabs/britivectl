package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// XDG paths follow https://specifications.freedesktop.org/basedir-spec/
//
//   Config  -> os.UserConfigDir() / bctl   (default: ~/.config/bctl on Linux, ~/Library/Application Support/bctl on macOS)
//   Cache   -> os.UserCacheDir()  / bctl   (default: ~/.cache/bctl on Linux, ~/Library/Caches/bctl on macOS)
//
// On first run after the upgrade, MigrateLegacyDir() moves anything under the
// old ~/.bctl/ directory into these XDG locations.

const (
	legacyDirName = ".bctl"
	appDirName    = "bctl"

	configFileName   = "config.yaml"
	profilesFileName = "profiles.json"
)

// legacyDir returns the pre-XDG directory (~/.bctl).
func legacyDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "." + appDirName
	}
	return filepath.Join(home, legacyDirName)
}

// xdgConfigDir returns the XDG config directory for bctl.
func xdgConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return legacyDir()
	}
	return filepath.Join(base, appDirName)
}

// xdgCacheDir returns the XDG cache directory for bctl.
func xdgCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return legacyDir()
	}
	return filepath.Join(base, appDirName)
}

// ConfigFilePath returns the absolute path to the config file.
func ConfigFilePath() string {
	return filepath.Join(xdgConfigDir(), configFileName)
}

// ProfilesCachePath returns the absolute path to the profiles cache file.
func ProfilesCachePath() string {
	return filepath.Join(xdgCacheDir(), profilesFileName)
}

// EnsureXDGDirs creates the XDG config and cache directories if they do not
// exist, with restrictive permissions.
func EnsureXDGDirs() error {
	for _, dir := range []string{xdgConfigDir(), xdgCacheDir()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	return nil
}

// MigrateLegacyDir checks for ~/.bctl/ and moves its contents to the XDG
// locations on first run. The legacy directory is removed after a successful
// migration. Returns (migrated, error) where migrated indicates whether any
// files were moved.
//
// Migration rules:
//   config.yaml  -> xdgConfigDir()/config.yaml
//   anything else -> xdgCacheDir()/<name>
//
// Existing files at the destination are NOT overwritten -- if the user has
// already run the new version and has a config, the legacy file is left in
// place and we return without error. This keeps the migration idempotent and
// non-destructive.
func MigrateLegacyDir() (bool, error) {
	legacy := legacyDir()

	info, err := os.Stat(legacy)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("checking legacy dir: %w", err)
	}
	if !info.IsDir() {
		return false, nil
	}

	if err := EnsureXDGDirs(); err != nil {
		return false, err
	}

	entries, err := os.ReadDir(legacy)
	if err != nil {
		return false, fmt.Errorf("reading legacy dir: %w", err)
	}

	var moved bool
	var migrationErrors []error
	for _, entry := range entries {
		src := filepath.Join(legacy, entry.Name())

		var dst string
		if entry.Name() == configFileName {
			dst = ConfigFilePath()
		} else {
			dst = filepath.Join(xdgCacheDir(), entry.Name())
		}

		if _, err := os.Stat(dst); err == nil {
			// Destination already exists -- do not overwrite.
			continue
		}

		if err := moveFile(src, dst); err != nil {
			migrationErrors = append(migrationErrors, fmt.Errorf("%s: %w", entry.Name(), err))
			continue
		}
		moved = true
	}

	if len(migrationErrors) > 0 {
		return moved, fmt.Errorf("migrating legacy dir: %w", errors.Join(migrationErrors...))
	}

	// Best-effort removal of the now-empty legacy directory.
	_ = os.Remove(legacy)

	return moved, nil
}

// moveFile copies src to dst atomically and removes the source on success.
// os.Rename is attempted first; if it fails (e.g. cross-device), we fall back
// to copy + remove.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src) //nolint:gosec // src comes from os.ReadDir on a directory we own
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying: %w", err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("removing source after copy: %w", err)
	}
	return nil
}
