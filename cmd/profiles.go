package cmd

import (
	"fmt"

	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newProfilesCmd() *cobra.Command {
	profilesCmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage Britive profiles",
		Long:  "List and sync Britive access profiles.",
	}

	profilesCmd.AddCommand(newProfilesListCmd())
	profilesCmd.AddCommand(newProfilesSyncCmd())
	return profilesCmd
}

func newProfilesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long:  "Display a table of Britive access profiles available to you.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesList()
		},
	}
}

func runProfilesList() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		output.Info("No profiles configured. Run 'bctl profiles sync' to fetch from API.")
		return nil
	}

	rows := make([][]string, 0, len(cfg.Profiles))
	for alias, p := range cfg.Profiles {
		rows = append(rows, []string{
			alias,
			p.BritivePath,
			p.Cloud,
			p.Region,
			p.AWSProfile,
		})
	}
	output.PrintTable([]string{"ALIAS", "BRITIVE PATH", "CLOUD", "REGION", "AWS PROFILE"}, rows)
	return nil
}

func newProfilesSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync profiles from Britive API",
		Long:  "Fetch available profiles from the Britive API and save them to local config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesSync()
		},
	}
}

func runProfilesSync() error {
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

	spin := output.NewSpinner("Syncing profiles from Britive API...")
	spin.Start()

	client := britive.NewClient(t, token)
	profiles, err := client.ListProfiles()
	if err != nil {
		spin.Fail(fmt.Sprintf("Failed to fetch profiles: %v", err))
		return err
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.Profile)
	}

	for _, p := range profiles {
		alias := p.Name
		cfg.Profiles[alias] = config.Profile{
			BritivePath: p.ID,
			Cloud:       "aws",
		}
	}

	if err := config.Save(cfg); err != nil {
		spin.Fail("Failed to save config")
		return fmt.Errorf("saving config: %w", err)
	}

	spin.Success(fmt.Sprintf("Synced %d profiles", len(profiles)))
	return nil
}
