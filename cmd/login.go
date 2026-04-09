package cmd

import (
	"fmt"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newLoginCmd() *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Britive",
		Long: `Authenticate with the Britive platform.

Use --token for API token authentication, or omit it for browser-based SSO.
The token is stored securely in your OS keychain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(token)
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Britive API token (skips browser SSO)")
	return cmd
}

func runLogin(token string) error {
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

	var finalToken string

	if token != "" {
		// Token auth
		output.Info("Validating token for tenant %s...", t)
		if err := britive.AuthWithToken(t, token); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		finalToken = token
	} else {
		// Browser SSO
		output.Info("Starting browser-based authentication for tenant %s...", t)
		tok, err := britive.AuthWithBrowser(t)
		if err != nil {
			return fmt.Errorf("browser authentication failed: %w", err)
		}
		finalToken = tok
	}

	// Store token in keychain
	if err := config.SetToken(t, finalToken); err != nil {
		return fmt.Errorf("storing token in keychain: %w", err)
	}

	output.Success("Logged in to tenant %s", t)
	return nil
}
