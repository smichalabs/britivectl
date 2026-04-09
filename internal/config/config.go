package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds all bctl configuration.
type Config struct {
	Tenant        string             `mapstructure:"tenant"         yaml:"tenant"`
	DefaultRegion string             `mapstructure:"default_region" yaml:"default_region"`
	Auth          AuthConfig         `mapstructure:"auth"           yaml:"auth"`
	Profiles      map[string]Profile `mapstructure:"profiles"       yaml:"profiles"`
}

// AuthConfig holds authentication method configuration.
type AuthConfig struct {
	Method string `mapstructure:"method" yaml:"method"` // browser | token
}

// Profile is a named Britive access profile.
type Profile struct {
	BritivePath string   `mapstructure:"britive_path" yaml:"britive_path"`
	AWSProfile  string   `mapstructure:"aws_profile"  yaml:"aws_profile"`
	Cloud       string   `mapstructure:"cloud"        yaml:"cloud"`
	Region      string   `mapstructure:"region"       yaml:"region"`
	EKSClusters []string `mapstructure:"eks_clusters" yaml:"eks_clusters"`
}

// ConfigDir returns ~/.bctl.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bctl"
	}
	return filepath.Join(home, ".bctl")
}

// ConfigPath returns ~/.bctl/config.yaml.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Load reads configuration from disk and environment variables.
func Load() (*Config, error) {
	v := viper.New()

	// Environment variable bindings
	v.SetEnvPrefix("BCTL")
	v.AutomaticEnv()
	_ = v.BindEnv("tenant", "BCTL_TENANT")
	_ = v.BindEnv("auth.method", "BCTL_TOKEN")
	_ = v.BindEnv("default_region", "BCTL_REGION")

	// Config file
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			// If the file simply doesn't exist, that is fine — return empty config.
			if _, statErr := os.Stat(ConfigPath()); !os.IsNotExist(statErr) {
				return nil, fmt.Errorf("reading config: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	return &cfg, nil
}

// Save writes cfg to ~/.bctl/config.yaml atomically.
func Save(cfg *Config) error {
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	// Write to temp file in same dir, then rename (atomic on same filesystem).
	tmpFile, err := os.CreateTemp(ConfigDir(), "config-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err = tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, ConfigPath()); err != nil {
		return fmt.Errorf("renaming config file: %w", err)
	}
	return nil
}
