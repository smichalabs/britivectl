package cmd

import (
	"fmt"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCheckinCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "checkin <alias>",
		Short: "Return a checked-out profile early",
		Long:  "Voluntarily return a Britive profile checkout before it expires.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckin(args[0])
		},
	}
}

func runCheckin(alias string) error {
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

	profile, ok := cfg.Profiles[alias]
	if !ok {
		return fmt.Errorf("profile alias %q not found in config", alias)
	}

	if profile.ProfileID == "" {
		return fmt.Errorf("profile %q is missing API IDs — run 'bctl profiles sync' to update", alias)
	}

	client := newAPIClient(t, token)

	// Find the active transaction for this profile
	sessions, err := client.MySessions()
	if err != nil {
		return fmt.Errorf("fetching active sessions: %w", err)
	}

	var transactionID string
	for _, s := range sessions {
		if s.CheckedIn == nil && s.PapID == profile.ProfileID {
			transactionID = s.TransactionID
			break
		}
	}
	if transactionID == "" {
		return fmt.Errorf("no active checkout found for %q", alias)
	}

	spin := output.NewSpinner(fmt.Sprintf("Checking in %s...", alias))
	spin.Start()

	if err := client.Checkin(transactionID); err != nil {
		spin.Fail(fmt.Sprintf("Checkin failed: %v", err))
		return err
	}

	spin.Success(fmt.Sprintf("Checked in %s successfully", alias))
	return nil
}
