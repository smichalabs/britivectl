package cmd

import (
	"fmt"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show active profile checkouts",
		Long:  "Display a table of currently active Britive profile checkouts with their expiry times.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
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

	token, err := config.GetToken(t)
	if err != nil {
		return fmt.Errorf("not logged in — run 'bctl login' first")
	}

	client := britive.NewClient(t, token)
	sessions, err := client.MySessions()
	if err != nil {
		return fmt.Errorf("fetching sessions: %w", err)
	}

	if len(sessions) == 0 {
		output.Info("No active checkouts.")
		return nil
	}

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		rows = append(rows, []string{
			s.ProfileName,
			s.Status,
			s.CreatedAt,
			s.ExpiresAt,
		})
	}
	output.PrintTable([]string{"PROFILE", "STATUS", "CREATED", "EXPIRES"}, rows)
	return nil
}
