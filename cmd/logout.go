package cmd

import (
	"fmt"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out from Britive",
		Long:  "Remove stored credentials from the OS keychain and clear the local session.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout()
		},
	}
}

func runLogout() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	t := cfg.Tenant
	if v := viper.GetString("tenant"); v != "" {
		t = v
	}
	if t == "" {
		return fmt.Errorf("tenant not configured — run 'bctl init' first")
	}

	if err := config.DeleteToken(t); err != nil {
		output.Warning("Could not remove token from keychain: %v", err)
	} else {
		output.Success("Removed token from keychain for tenant %s", t)
	}

	output.Success("Logged out successfully")
	return nil
}
