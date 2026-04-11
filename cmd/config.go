package cmd

import (
	"fmt"
	"strings"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get and set configuration values",
		Long:  "Read and write bctl configuration values in ~/.config/bctl/config.yaml.",
	}

	configCmd.AddCommand(newConfigGetCmd())
	configCmd.AddCommand(newConfigSetCmd())
	configCmd.AddCommand(newConfigViewCmd())
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
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val := args[1]

			cfg, err := config.Load()
			if err != nil {
				cfg = &config.Config{}
			}

			// Apply the key-value to the config struct
			switch strings.ToLower(key) {
			case "tenant":
				cfg.Tenant = val
			case "default_region", "region":
				cfg.DefaultRegion = val
			case "auth.method":
				cfg.Auth.Method = val
			default:
				return fmt.Errorf("unknown config key %q — valid keys: tenant, default_region, auth.method", key)
			}

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			output.Success("Set %s = %s", key, val)
			return nil
		},
	}
}

func newConfigViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view",
		Short: "Display the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			return output.PrintJSON(cfg)
		},
	}
}
