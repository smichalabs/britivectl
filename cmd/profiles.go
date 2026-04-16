package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/smichalabs/britivectl/internal/aliases"
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
	var verbose bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long: `Display a table of Britive access profiles available to you.

By default the table shows alias, cloud, and Britive path -- the
fields that have a meaningful value for every profile. Pass --verbose
to also show region and AWS profile name overrides; those columns are
populated only for profiles you have customized in config.yaml.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfilesList(verbose)
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show region and AWS profile columns")
	return cmd
}

func runProfilesList(verbose bool) error {
	// Prefer the on-disk cache; fall back to legacy config.yaml profiles.
	cache, err := config.LoadProfilesCache()
	if err != nil && !errors.Is(err, config.ErrCacheMiss) {
		return fmt.Errorf("loading profile cache: %w", err)
	}

	var profiles map[string]config.Profile
	if cache != nil && len(cache.Profiles) > 0 {
		profiles = cache.Profiles
	} else {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		profiles = cfg.Profiles
	}

	if len(profiles) == 0 {
		output.Info("No profiles configured. Run 'bctl profiles sync' to fetch from API.")
		return nil
	}

	// Stable alphabetical order so repeated invocations show profiles in
	// the same row positions and tests stay deterministic.
	aliases := make([]string, 0, len(profiles))
	for alias := range profiles {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	rows := make([][]string, 0, len(profiles))
	for _, alias := range aliases {
		p := profiles[alias]
		row := []string{alias, p.Cloud, p.BritivePath}
		if verbose {
			row = append(row, dashIfEmpty(p.Region), awsProfileColumn(alias, p))
		}
		rows = append(rows, row)
	}

	headers := []string{"ALIAS", "CLOUD", "BRITIVE PATH"}
	if verbose {
		headers = append(headers, "REGION", "AWS PROFILE")
	}
	output.PrintTable(headers, rows)
	return nil
}

// dashIfEmpty returns "-" when s is empty so the table renders a clear
// "no value" cell instead of a confusing blank.
func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// awsProfileColumn returns the value shown in the AWS PROFILE column. For
// non-AWS profiles the field is meaningless, so we return "-" instead of an
// empty cell. For AWS profiles, if the user has not set an aws_profile
// override, we show the alias because that is what bctl actually uses at
// checkout time (see cmd/checkout.go).
func awsProfileColumn(alias string, p config.Profile) string {
	if !strings.EqualFold(p.Cloud, "aws") {
		return "-"
	}
	if p.AWSProfile != "" {
		return p.AWSProfile
	}
	return alias
}

func newProfilesSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync profiles from Britive API",
		Long:  "Fetch available profiles from the Britive API and save them to local config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesSync(cmd.Context())
		},
	}
}

func runProfilesSync(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	t := cfg.Tenant
	if v := viper.GetString("tenant"); v != "" {
		t = v
	}
	if t == "" {
		return fmt.Errorf("tenant not configured -- run 'bctl init' first")
	}

	token, err := requireToken(ctx, t)
	if err != nil {
		return err
	}

	spin := output.NewSpinner("Syncing profiles from Britive API...")
	spin.Start()

	client := newAPIClient(t, token)
	entries, err := client.ListAccess(ctx)
	if err != nil {
		spin.Fail(fmt.Sprintf("Failed to fetch profiles: %v", err))
		return err
	}

	profiles := aliases.BuildMap(entries)
	if err := config.SaveProfilesCache(&config.ProfilesCache{Profiles: profiles}); err != nil {
		spin.Fail("Failed to save profile cache")
		return fmt.Errorf("saving profile cache: %w", err)
	}

	spin.Success(fmt.Sprintf("Synced %d profiles", len(entries)))
	return nil
}
