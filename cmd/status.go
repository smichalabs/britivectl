package cmd

import (
	"fmt"
	"strings"

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

	token, err := requireToken(t)
	if err != nil {
		return fmt.Errorf("not logged in — run 'bctl login' first")
	}

	client := newAPIClient(t, token)
	sessions, err := client.MySessions()
	if err != nil {
		return fmt.Errorf("fetching sessions: %w", err)
	}

	if len(sessions) == 0 {
		output.Info("No active checkouts.")
		return nil
	}

	// Build a reverse lookup: profileId → alias from local config
	aliasLookup := make(map[string]string) // profileId → alias
	for alias, p := range cfg.Profiles {
		aliasLookup[p.ProfileID] = alias
	}

	rows := make([][]string, 0, len(sessions))
	for _, s := range sessions {
		alias := aliasLookup[s.PapID]
		if alias == "" {
			alias = s.PapID // fallback to raw ID
		}
		expiry := strings.Replace(s.Expiration, "T", " ", 1)
		expiry = strings.TrimSuffix(expiry, "Z") + " UTC"
		rows = append(rows, []string{
			alias,
			s.Status,
			expiry,
		})
	}
	output.PrintTable([]string{"PROFILE", "STATUS", "EXPIRES"}, rows)
	return nil
}
