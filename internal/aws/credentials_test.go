package aws_test

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/ini.v1"

	bctlaws "github.com/smichalabs/britivectl/internal/aws"
)

func credentialsFilePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
}

func TestWriteCredentials_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := bctlaws.AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "test-session-token-example",
		Region:          "us-east-1",
	}

	if err := bctlaws.WriteCredentials("test-profile", creds); err != nil {
		t.Fatalf("WriteCredentials() error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	sec := cfg.Section("test-profile")
	if sec == nil {
		t.Fatal("section 'test-profile' not found")
	}

	check := func(key, want string) {
		t.Helper()
		got := sec.Key(key).String()
		if got != want {
			t.Errorf("key %q = %q, want %q", key, got, want)
		}
	}

	check("aws_access_key_id", creds.AccessKeyID)
	check("aws_secret_access_key", creds.SecretAccessKey)
	check("aws_session_token", creds.SessionToken)
	check("region", creds.Region)
}

func TestWriteCredentials_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	credsA := bctlaws.AWSCredentials{
		AccessKeyID:     "KEY_A",
		SecretAccessKey: "SECRET_A",
	}
	credsB := bctlaws.AWSCredentials{
		AccessKeyID:     "KEY_B",
		SecretAccessKey: "SECRET_B",
	}

	if err := bctlaws.WriteCredentials("profile-a", credsA); err != nil {
		t.Fatalf("WriteCredentials(profile-a) error = %v", err)
	}
	if err := bctlaws.WriteCredentials("profile-b", credsB); err != nil {
		t.Fatalf("WriteCredentials(profile-b) error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	if !cfg.HasSection("profile-a") {
		t.Error("section 'profile-a' not found after writing both profiles")
	}
	if !cfg.HasSection("profile-b") {
		t.Error("section 'profile-b' not found after writing both profiles")
	}

	if got := cfg.Section("profile-a").Key("aws_access_key_id").String(); got != "KEY_A" {
		t.Errorf("profile-a aws_access_key_id = %q, want KEY_A", got)
	}
	if got := cfg.Section("profile-b").Key("aws_access_key_id").String(); got != "KEY_B" {
		t.Errorf("profile-b aws_access_key_id = %q, want KEY_B", got)
	}
}

func TestWriteCredentials_UpdateSameProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	first := bctlaws.AWSCredentials{
		AccessKeyID:     "FIRST_KEY",
		SecretAccessKey: "FIRST_SECRET",
	}
	second := bctlaws.AWSCredentials{
		AccessKeyID:     "SECOND_KEY",
		SecretAccessKey: "SECOND_SECRET",
	}

	if err := bctlaws.WriteCredentials("my-profile", first); err != nil {
		t.Fatalf("WriteCredentials() first write error = %v", err)
	}
	if err := bctlaws.WriteCredentials("my-profile", second); err != nil {
		t.Fatalf("WriteCredentials() second write error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	sec := cfg.Section("my-profile")
	if got := sec.Key("aws_access_key_id").String(); got != "SECOND_KEY" {
		t.Errorf("aws_access_key_id = %q, want SECOND_KEY (second write should win)", got)
	}
	if got := sec.Key("aws_secret_access_key").String(); got != "SECOND_SECRET" {
		t.Errorf("aws_secret_access_key = %q, want SECOND_SECRET", got)
	}
}

func TestWriteCredentials_WithSessionToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
		SessionToken:    "TOKEN123",
	}

	if err := bctlaws.WriteCredentials("token-profile", creds); err != nil {
		t.Fatalf("WriteCredentials() error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	sec := cfg.Section("token-profile")
	if !sec.HasKey("aws_session_token") {
		t.Error("expected aws_session_token key to exist when SessionToken is non-empty")
	}
	if got := sec.Key("aws_session_token").String(); got != "TOKEN123" {
		t.Errorf("aws_session_token = %q, want TOKEN123", got)
	}
}

func TestWriteCredentials_NoSessionToken(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
		SessionToken:    "",
	}

	if err := bctlaws.WriteCredentials("no-token-profile", creds); err != nil {
		t.Fatalf("WriteCredentials() error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	sec := cfg.Section("no-token-profile")
	if sec.HasKey("aws_session_token") {
		t.Error("expected aws_session_token key to be absent when SessionToken is empty")
	}
}

func TestWriteCredentials_WithRegion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
		Region:          "eu-west-1",
	}

	if err := bctlaws.WriteCredentials("region-profile", creds); err != nil {
		t.Fatalf("WriteCredentials() error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	sec := cfg.Section("region-profile")
	if !sec.HasKey("region") {
		t.Error("expected region key to exist when Region is non-empty")
	}
	if got := sec.Key("region").String(); got != "eu-west-1" {
		t.Errorf("region = %q, want eu-west-1", got)
	}
}

func TestWriteCredentials_NoRegion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	creds := bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
		Region:          "",
	}

	if err := bctlaws.WriteCredentials("no-region-profile", creds); err != nil {
		t.Fatalf("WriteCredentials() error = %v", err)
	}

	path := credentialsFilePath(t)
	cfg, err := ini.Load(path)
	if err != nil {
		t.Fatalf("ini.Load() error = %v", err)
	}

	sec := cfg.Section("no-region-profile")
	if sec.HasKey("region") {
		t.Error("expected region key to be absent when Region is empty")
	}
}

func TestWriteCredentials_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Place a regular file at ~/.aws so MkdirAll fails.
	if err := os.WriteFile(filepath.Join(tmpDir, ".aws"), []byte("blocker"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := bctlaws.WriteCredentials("profile", bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
	})
	if err == nil {
		t.Fatal("expected error when .aws exists as a file, got nil")
	}
}

func TestWriteCredentials_IniLoadError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	awsDir := filepath.Join(tmpDir, ".aws")
	if err := os.MkdirAll(awsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Create a directory at credentials path so ini.Load fails (cannot read dir as ini).
	if err := os.MkdirAll(filepath.Join(awsDir, "credentials"), 0o700); err != nil {
		t.Fatal(err)
	}

	err := bctlaws.WriteCredentials("profile", bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
	})
	if err == nil {
		t.Fatal("expected error when credentials is a directory, got nil")
	}
}

func TestWriteCredentials_CreateTempError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — cannot test permission denied errors")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	awsDir := filepath.Join(tmpDir, ".aws")
	if err := os.MkdirAll(awsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Make .aws non-writable so CreateTemp fails.
	if err := os.Chmod(awsDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(awsDir, 0o755) })

	err := bctlaws.WriteCredentials("profile", bctlaws.AWSCredentials{
		AccessKeyID:     "KEY",
		SecretAccessKey: "SECRET",
	})
	if err == nil {
		t.Fatal("expected error when .aws is not writable, got nil")
	}
}
