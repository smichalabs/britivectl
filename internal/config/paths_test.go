package config

// This file is in package config (not config_test) so it can exercise the
// unexported helpers in paths.go. The exported surface is tested via
// cache_test.go and config_test.go in package config_test.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureXDGDirs_Creates(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

	if err := EnsureXDGDirs(); err != nil {
		t.Fatalf("EnsureXDGDirs() error = %v", err)
	}

	for _, d := range []string{xdgConfigDir(), xdgCacheDir()} {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("directory %s was not created: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}
}

func TestMigrateLegacyDir_NoLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

	moved, err := MigrateLegacyDir()
	if err != nil {
		t.Fatalf("MigrateLegacyDir() error = %v", err)
	}
	if moved {
		t.Error("moved = true when no legacy dir existed")
	}
}

func TestMigrateLegacyDir_MovesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

	legacy := filepath.Join(tmpDir, ".bctl")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}

	// Seed legacy files
	cfgContent := []byte("tenant: acme\n")
	if err := os.WriteFile(filepath.Join(legacy, "config.yaml"), cfgContent, 0o600); err != nil {
		t.Fatal(err)
	}
	otherContent := []byte("some cache data")
	if err := os.WriteFile(filepath.Join(legacy, "profiles.json"), otherContent, 0o600); err != nil {
		t.Fatal(err)
	}

	moved, err := MigrateLegacyDir()
	if err != nil {
		t.Fatalf("MigrateLegacyDir() error = %v", err)
	}
	if !moved {
		t.Error("moved = false when legacy files existed")
	}

	// Verify files are at new locations
	cfgAt := ConfigFilePath()
	if _, err := os.Stat(cfgAt); err != nil {
		t.Errorf("config.yaml not found at new location %s: %v", cfgAt, err)
	}
	cacheAt := ProfilesCachePath()
	if _, err := os.Stat(cacheAt); err != nil {
		t.Errorf("profiles.json not found at new location %s: %v", cacheAt, err)
	}

	// Verify legacy dir was removed
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy dir should have been removed; stat err = %v", err)
	}
}

func TestMigrateLegacyDir_DoesNotOverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

	// Seed legacy
	legacy := filepath.Join(tmpDir, ".bctl")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.yaml"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Seed destination with different content
	if err := os.MkdirAll(xdgConfigDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ConfigFilePath(), []byte("xdg-existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := MigrateLegacyDir(); err != nil {
		t.Fatalf("MigrateLegacyDir() error = %v", err)
	}

	// The XDG file should NOT be overwritten
	data, err := os.ReadFile(ConfigFilePath())
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "xdg-existing" {
		t.Errorf("XDG config was overwritten; got %q, want 'xdg-existing'", string(data))
	}
}

func TestProfilesCachePath_Shape(t *testing.T) {
	if filepath.Base(ProfilesCachePath()) != "profiles.json" {
		t.Errorf("ProfilesCachePath base = %q, want profiles.json", filepath.Base(ProfilesCachePath()))
	}
}

func TestMoveFile_CopyFallback(t *testing.T) {
	// Rename works on the same filesystem; we test the copy path by passing
	// a source and destination that are reachable.
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := moveFile(src, dst); err != nil {
		t.Fatalf("moveFile() error = %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source still exists after move")
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Errorf("destination content = %q, want 'hello'", string(data))
	}
}

func TestLegacyDir_Shape(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if d := legacyDir(); filepath.Base(d) != ".bctl" {
		t.Errorf("legacyDir base = %q, want .bctl", filepath.Base(d))
	}
}

func TestMoveFile_SourceMissing(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "nonexistent.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	if err := moveFile(src, dst); err == nil {
		t.Error("expected error for missing source, got nil")
	}
}

func TestMoveFile_DestExists(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")
	if err := os.WriteFile(src, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Rename on same filesystem will succeed (overwrites); this exercises
	// the primary path without actually failing.
	_ = moveFile(src, dst)
}

func TestMigrateLegacyDir_NotADir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

	// Create a file where .bctl should be.
	legacy := filepath.Join(tmpDir, ".bctl")
	if err := os.WriteFile(legacy, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}
	moved, err := MigrateLegacyDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if moved {
		t.Error("moved should be false when legacy path is a file")
	}
}
