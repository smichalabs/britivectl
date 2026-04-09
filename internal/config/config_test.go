package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/smichalabs/britivectl/internal/config"
)

func TestConfigDirAndPath(t *testing.T) {
	dir := config.ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir() returned empty string")
	}
	path := config.ConfigPath()
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("ConfigPath() base = %q, want config.yaml", filepath.Base(path))
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp dir so we don't touch the real config.
	tmpDir := t.TempDir()

	// Patch config path via env is not straightforward; instead we test
	// the serialization round-trip by using a custom config path approach.
	// We call Save/Load indirectly by writing to a temp location.
	cfg := &config.Config{
		Tenant:        "test-tenant",
		DefaultRegion: "us-east-1",
		Auth:          config.AuthConfig{Method: "token"},
		Profiles: map[string]config.Profile{
			"dev": {
				BritivePath: "org/env/app/dev",
				AWSProfile:  "dev",
				Cloud:       "aws",
				Region:      "us-east-1",
				EKSClusters: []string{"cluster-1"},
			},
		},
	}

	// Save to a temp file using a real config path override via env var trick.
	// Since Save uses ConfigPath() which uses HOME, we'll override HOME.
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	if err := os.MkdirAll(filepath.Join(tmpDir, ".bctl"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Tenant != cfg.Tenant {
		t.Errorf("Tenant = %q, want %q", loaded.Tenant, cfg.Tenant)
	}
	if loaded.DefaultRegion != cfg.DefaultRegion {
		t.Errorf("DefaultRegion = %q, want %q", loaded.DefaultRegion, cfg.DefaultRegion)
	}
	if loaded.Auth.Method != cfg.Auth.Method {
		t.Errorf("Auth.Method = %q, want %q", loaded.Auth.Method, cfg.Auth.Method)
	}
	if len(loaded.Profiles) != 1 {
		t.Errorf("len(Profiles) = %d, want 1", len(loaded.Profiles))
	}
	dev, ok := loaded.Profiles["dev"]
	if !ok {
		t.Fatal("profile 'dev' not found after load")
	}
	if dev.AWSProfile != "dev" {
		t.Errorf("dev.AWSProfile = %q, want dev", dev.AWSProfile)
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// No config file exists — Load should return an empty config, not an error.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() with missing file returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	bctlDir := filepath.Join(tmpDir, ".bctl")
	if err := os.MkdirAll(bctlDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Write a malformed YAML file so viper returns a parse error.
	if err := os.WriteFile(filepath.Join(bctlDir, "config.yaml"), []byte("tenant: [unclosed"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for malformed config file, got nil")
	}
}

func TestSave_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Place a regular file where .bctl directory should be so MkdirAll fails.
	if err := os.WriteFile(filepath.Join(tmpDir, ".bctl"), []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := config.Save(&config.Config{})
	if err == nil {
		t.Fatal("expected error when .bctl exists as a file, got nil")
	}
}

func TestSave_CreateTempError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — cannot test permission denied errors")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	bctlDir := filepath.Join(tmpDir, ".bctl")
	if err := os.MkdirAll(bctlDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Make .bctl non-writable so CreateTemp fails.
	if err := os.Chmod(bctlDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(bctlDir, 0o755) })

	err := config.Save(&config.Config{})
	if err == nil {
		t.Fatal("expected error when .bctl is not writable, got nil")
	}
}

func TestSave_RenameError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	bctlDir := filepath.Join(tmpDir, ".bctl")
	if err := os.MkdirAll(bctlDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Create a directory at config.yaml path so Rename fails (can't overwrite dir with file).
	if err := os.MkdirAll(filepath.Join(bctlDir, "config.yaml"), 0o700); err != nil {
		t.Fatal(err)
	}

	err := config.Save(&config.Config{})
	if err == nil {
		t.Fatal("expected error when config.yaml is a directory, got nil")
	}
}
