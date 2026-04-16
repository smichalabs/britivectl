package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize bctl configuration",
		Long:  "Interactive wizard to set up bctl configuration including tenant and authentication method.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit()
		},
	}
}

func runInit() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to bctl! Let's set up your configuration.")
	fmt.Println()

	// Load existing config or start fresh
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	// Tenant
	fmt.Printf("Britive tenant name [%s]: ", cfg.Tenant)
	tenantInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading tenant: %w", err)
	}
	tenantInput = strings.TrimSpace(tenantInput)
	if tenantInput != "" {
		cfg.Tenant = tenantInput
	}
	if cfg.Tenant == "" {
		return fmt.Errorf("tenant is required")
	}

	// Auth method
	fmt.Printf("Authentication method (browser/token) [%s]: ", defaultStr(cfg.Auth.Method, "browser"))
	methodInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading auth method: %w", err)
	}
	methodInput = strings.TrimSpace(methodInput)
	if methodInput == "" {
		methodInput = defaultStr(cfg.Auth.Method, "browser")
	}
	if methodInput != "browser" && methodInput != "token" {
		return fmt.Errorf("invalid auth method %q: must be 'browser' or 'token'", methodInput)
	}
	cfg.Auth.Method = methodInput

	// Default region
	fmt.Printf("Default AWS region [%s]: ", defaultStr(cfg.DefaultRegion, "us-east-1"))
	regionInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading region: %w", err)
	}
	regionInput = strings.TrimSpace(regionInput)
	if regionInput != "" {
		cfg.DefaultRegion = regionInput
	}
	if cfg.DefaultRegion == "" {
		cfg.DefaultRegion = "us-east-1"
	}

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	output.Success("Configuration saved to %s", config.ConfigPath())
	return nil
}

func defaultStr(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}
