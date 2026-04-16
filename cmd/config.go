package cmd

import (
	"fmt"
	"strings"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// validConfigKeys lists every configuration key accepted by `config set` and
// `config unset`. Kept in one place so help text, validation, and the switch
// statements stay in sync.
var validConfigKeys = []string{"tenant", "default_region", "auth.method"}

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set configuration values",
		Long:  fmt.Sprintf("Read and write bctl configuration values stored at %s.", config.ConfigFilePath()),
	}

	configCmd.AddCommand(newConfigGetCmd())
	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigUnsetCmd())
	configCmd.AddCommand(newConfigViewCmd())
	configCmd.AddCommand(newConfigPathCmd())
	return configCmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := viper.GetString(key)
			if val == "" {
				output.Warning("Key %q is not set", key)
				return nil
			}
			fmt.Println(val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value and persist it to the config file.

Valid keys:
  tenant          Britive tenant name (subdomain only, e.g. "acme")
  default_region  Default AWS region (e.g. "us-east-1")
  auth.method     Authentication method: browser or token`,
		Example: `  bctl config set tenant acme
  bctl config set default_region us-west-2
  bctl config set auth.method browser`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("requires key and value arguments -- run 'bctl config set --help' for valid keys")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := args[1]

			cfg, err := config.Load()
			if err != nil {
				cfg = &config.Config{}
			}

			// Tenant input like https://acme.britive-app.com/ is sanitized
			// before persistence so we do not save a double-URL.
			if strings.EqualFold(key, "tenant") {
				cleaned := britive.SanitizeTenant(val)
				if cleaned != val {
					output.Info("Using tenant %q (stripped URL scheme and britive-app.com suffix)", cleaned)
				}
				val = cleaned
			}
			if err := applyConfigKey(cfg, key, val); err != nil {
				return err
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			output.Success("Set %s = %s", key, val)
			return nil
		},
	}
}

// applyConfigKey mutates cfg to apply val to the named key. Returns an error
// for unknown keys listing the valid ones.
func applyConfigKey(cfg *config.Config, key, val string) error {
	switch strings.ToLower(key) {
	case "tenant":
		cfg.Tenant = val
	case "default_region", "region":
		cfg.DefaultRegion = val
	case "auth.method":
		cfg.Auth.Method = val
	default:
		return fmt.Errorf("unknown config key %q -- valid keys: %s", key, strings.Join(validConfigKeys, ", "))
	}
	return nil
}

func newConfigUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a configuration value",
		Long: `Clear a configuration value by resetting it to its zero value and saving.

Valid keys:
  tenant          Britive tenant name
  default_region  Default AWS region
  auth.method     Authentication method`,
		Example: `  bctl config unset tenant
  bctl config unset default_region`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("requires exactly one key argument -- run 'bctl config unset --help' for valid keys")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if err := applyConfigKey(cfg, key, ""); err != nil {
				return err
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			output.Success("Unset %s", key)
			return nil
		},
	}
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Display the current configuration",
		Long:  "Print the current configuration in YAML format -- matches the on-disk config file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			return output.PrintYAML(cfg)
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the resolved paths for config and cache files",
		Long:  "Print the platform-specific locations bctl uses for the config file and the profile cache.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Config: %s\n", config.ConfigFilePath())
			fmt.Printf("Cache:  %s\n", config.ProfilesCachePath())
			return nil
		},
	}
}
