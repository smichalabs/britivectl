package aws

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// AWSCredentials holds temporary AWS credential values.
type AWSCredentials struct { //nolint:revive // name predates this lint rule; renaming would be a breaking change
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
}

// credentialsPath returns the path to ~/.aws/credentials.
func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".aws", "credentials"), nil
}

// WriteCredentials writes AWS credentials to ~/.aws/credentials atomically.
func WriteCredentials(profile string, creds AWSCredentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}

	// Ensure ~/.aws directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating .aws dir: %w", err)
	}

	// Load existing credentials file or create new one
	var cfg *ini.File
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		cfg = ini.Empty()
	} else {
		cfg, err = ini.Load(path)
		if err != nil {
			return fmt.Errorf("loading credentials file: %w", err)
		}
	}

	// Set the profile section
	sec, err := cfg.NewSection(profile)
	if err != nil {
		// Section may already exist; get it instead
		sec = cfg.Section(profile)
	}

	sec.Key("aws_access_key_id").SetValue(creds.AccessKeyID)
	sec.Key("aws_secret_access_key").SetValue(creds.SecretAccessKey)
	if creds.SessionToken != "" {
		sec.Key("aws_session_token").SetValue(creds.SessionToken)
	}
	if creds.Region != "" {
		sec.Key("region").SetValue(creds.Region)
	}

	// Write atomically via temp file + rename
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "credentials-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp credentials file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := cfg.WriteTo(tmpFile); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing credentials: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replacing credentials file: %w", err)
	}

	return nil
}
