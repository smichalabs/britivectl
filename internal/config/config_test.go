package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/smichalabs/britivectl/internal/config"
)

// testConfigDir returns the directory where config.ConfigPath() will write,
// given that HOME has been re-rooted to tmpDir. We reuse the production
// discovery by calling os.UserConfigDir after the test has set HOME.
func testConfigDir(t *testing.T) string {
	t.Helper()
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir: %v", err)
	}
	return filepath.Join(base, "bctl")
}

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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, ".cache"))

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

	if err := os.MkdirAll(testConfigDir(t), 0o700); err != nil {
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
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

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
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	dir := testConfigDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Write a malformed YAML file so viper returns a parse error.
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("tenant: [unclosed"), 0o600); err != nil {
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
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Place a regular file where the config directory should be so MkdirAll fails.
	if err := os.MkdirAll(filepath.Dir(testConfigDir(t)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testConfigDir(t), []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := config.Save(&config.Config{})
	if err == nil {
		t.Fatal("expected error when config dir exists as a file, got nil")
	}
}

func TestSave_CreateTempError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root -- cannot test permission denied errors")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	dir := testConfigDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := config.Save(&config.Config{})
	if err == nil {
		t.Fatal("expected error when config dir is not writable, got nil")
	}
}

func TestSave_RenameError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	dir := testConfigDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Create a directory at config.yaml path so Rename fails (can't overwrite dir with file).
	if err := os.MkdirAll(filepath.Join(dir, "config.yaml"), 0o700); err != nil {
		t.Fatal(err)
	}

	err := config.Save(&config.Config{})
	if err == nil {
		t.Fatal("expected error when config.yaml is a directory, got nil")
	}
}

func TestApplyProfileOverrides(t *testing.T) {
	base := map[string]config.Profile{
		// API-sourced entry with no user fields set.
		"ctp_aws-dev-nonprod": {
			ProfileID:     "p1",
			EnvironmentID: "e1",
			BritivePath:   "AWS/Dev/Admin",
			Cloud:         "aws",
		},
		// API entry the user does not override -- must pass through untouched.
		"ctp_aws-prod": {ProfileID: "p2", Cloud: "aws"},
	}
	overrides := map[string]config.Profile{
		"ctp_aws-dev-nonprod": {AWSProfile: "default", Region: "us-east-1"},
		// Profile defined only in config.yaml -- must be added as-is.
		"manual-only": {Cloud: "aws", BritivePath: "AWS/Manual"},
	}

	got := config.ApplyProfileOverrides(base, overrides)

	dev := got["ctp_aws-dev-nonprod"]
	if dev.AWSProfile != "default" || dev.Region != "us-east-1" {
		t.Errorf("override not applied: %+v", dev)
	}
	if dev.ProfileID != "p1" || dev.EnvironmentID != "e1" || dev.Cloud != "aws" {
		t.Errorf("API fields lost after overlay: %+v", dev)
	}
	if prod := got["ctp_aws-prod"]; prod.ProfileID != "p2" || prod.AWSProfile != "" {
		t.Errorf("non-overridden profile changed: %+v", prod)
	}
	if manual, ok := got["manual-only"]; !ok || manual.BritivePath != "AWS/Manual" {
		t.Errorf("config-only profile not added: %+v", manual)
	}
}

func TestApplyProfileOverrides_NoOverrides(t *testing.T) {
	base := map[string]config.Profile{"a": {Cloud: "aws"}}
	if got := config.ApplyProfileOverrides(base, nil); len(got) != 1 || got["a"].Cloud != "aws" {
		t.Errorf("nil overrides should return base unchanged, got %+v", got)
	}
}
