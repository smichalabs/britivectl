package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check bctl environment and dependencies",
		Long:  "Run a series of health checks to ensure bctl is correctly configured and all dependencies are available.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
}

type check struct {
	name string
	fn   func() (string, error)
}

func runDoctor() error {
	green := color.New(color.FgGreen, color.Bold)
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	fmt.Println("bctl doctor — checking your environment")

	checks := []check{
		{
			name: "Config file exists",
			fn: func() (string, error) {
				path := config.ConfigPath()
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return "", fmt.Errorf("not found at %s — run 'bctl init'", path)
				}
				return config.ConfigPath(), nil
			},
		},
		{
			name: "Tenant is configured",
			fn: func() (string, error) {
				cfg, err := config.Load()
				if err != nil {
					return "", fmt.Errorf("could not load config: %w", err)
				}
				if cfg.Tenant == "" {
					return "", fmt.Errorf("not set — run 'bctl config set tenant <name>'")
				}
				return cfg.Tenant, nil
			},
		},
		{
			name: "Token in keychain",
			fn: func() (string, error) {
				cfg, err := config.Load()
				if err != nil {
					return "", fmt.Errorf("could not load config")
				}
				if cfg.Tenant == "" {
					return "", fmt.Errorf("tenant not set")
				}
				tok, err := config.GetToken(cfg.Tenant)
				if err != nil || tok == "" {
					return "", fmt.Errorf("no token stored — run 'bctl login'")
				}
				return "found", nil
			},
		},
		{
			name: "Britive API reachable",
			fn: func() (string, error) {
				cfg, err := config.Load()
				if err != nil || cfg.Tenant == "" {
					return "", fmt.Errorf("skipped (tenant not configured)")
				}
				url := fmt.Sprintf("https://%s.britive-app.com/api/v1/health", cfg.Tenant)
				client := &http.Client{Timeout: 10 * time.Second}
				resp, err := client.Get(url)
				if err != nil {
					return "", fmt.Errorf("unreachable: %w", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode >= 500 {
					return "", fmt.Errorf("API returned %d", resp.StatusCode)
				}
				return fmt.Sprintf("HTTP %d", resp.StatusCode), nil
			},
		},
		{
			name: "aws CLI available",
			fn: func() (string, error) {
				path, err := exec.LookPath("aws")
				if err != nil {
					return "", fmt.Errorf("not found in PATH — install AWS CLI")
				}
				return path, nil
			},
		},
		{
			name: "kubectl available",
			fn: func() (string, error) {
				path, err := exec.LookPath("kubectl")
				if err != nil {
					return "", fmt.Errorf("not found in PATH — install kubectl for EKS operations")
				}
				return path, nil
			},
		},
	}

	allOK := true
	for _, c := range checks {
		detail, err := c.fn()
		if err != nil {
			red.Printf("  ✗ %s\n", c.name)
			yellow.Printf("    %v\n", err)
			allOK = false
		} else {
			green.Printf("  ✓ %s", c.name)
			if detail != "" {
				fmt.Printf(" (%s)", detail)
			}
			fmt.Println()
		}
	}

	fmt.Println()
	if allOK {
		green.Println("All checks passed!")
	} else {
		yellow.Println("Some checks failed. See above for details.")
	}
	return nil
}
