package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/smichalabs/britivectl/internal/aliases"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// listCacheMaxAge is how long 'bctl profiles list' trusts the cache before
// doing a background sync. Shorter than state.CacheMaxAge (24h) because
// 'list' is specifically how users discover new profiles they were added to
// recently -- a 24h stale window would make it feel broken.
const listCacheMaxAge = 1 * time.Hour

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
	var (
		verbose bool
		refresh bool
		noSync  bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long: `Display a table of Britive access profiles available to you.

By default the table shows alias, cloud, and Britive path -- the
fields that have a meaningful value for every profile. Pass --verbose
to also show region and AWS profile name overrides; those columns are
populated only for profiles you have customized in config.yaml.

If the cache is older than one hour, 'bctl profiles list' transparently
syncs from the Britive API as part of the same command (you will see a
short spinner). Users who were added to a new profile during the day do
not have to run 'bctl profiles sync' manually. Pass --refresh to force a
sync even on a fresh cache, or --no-sync to always use the cache as-is
(handy when offline).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if refresh && noSync {
				return fmt.Errorf("--refresh and --no-sync are mutually exclusive")
			}
			return runProfilesList(cmd.Context(), verbose, refresh, noSync)
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "show region and AWS profile columns")
	cmd.Flags().BoolVarP(&refresh, "refresh", "r", false, "force a sync from the Britive API before listing")
	cmd.Flags().BoolVar(&noSync, "no-sync", false, "never sync; use the cached profiles as-is")
	return cmd
}

func runProfilesList(ctx context.Context, verbose, refresh, noSync bool) error {
	// Prefer the on-disk cache; fall back to legacy config.yaml profiles.
	// Pass an empty tenant so the list surface still works when config is
	// mid-setup -- tenant mismatch protection is enforced by the checkout and
	// sync flows where it actually matters for correctness.
	cache, err := config.LoadProfilesCache("")
	if err != nil && !errors.Is(err, config.ErrCacheMiss) {
		return fmt.Errorf("loading profile cache: %w", err)
	}

	if config.ShouldAutoSync(cache, refresh, noSync, listCacheMaxAge) {
		synced, syncErr := syncProfilesForList(ctx, refresh)
		switch {
		case syncErr == nil:
			// Sync succeeded -- fall through with the freshly written cache.
			cache = &config.ProfilesCache{Profiles: synced}
		case refresh:
			// Explicit --refresh requested and it failed -- surface that to
			// the user rather than silently falling back to a stale cache.
			return syncErr
		default:
			// Best-effort auto-sync failed. Warn and carry on with whatever
			// we have so the user can still see their profiles offline.
			output.Warning("auto-sync failed, showing cached profiles: %v", syncErr)
		}
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
	aliasList := make([]string, 0, len(profiles))
	for alias := range profiles {
		aliasList = append(aliasList, alias)
	}
	sort.Strings(aliasList)

	rows := make([][]string, 0, len(profiles))
	for _, alias := range aliasList {
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

// syncProfilesForList resolves tenant + token and fetches a fresh snapshot
// from the Britive API, writing it to the cache. Returns the new profile
// map. The explicit flag controls user-facing messaging only -- the sync
// itself is the same work either way.
func syncProfilesForList(ctx context.Context, explicit bool) (map[string]config.Profile, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	t := cfg.Tenant
	if v := viper.GetString("tenant"); v != "" {
		t = v
	}
	if t == "" {
		// No tenant configured -- cannot sync. An explicit --refresh should
		// surface this; an auto-sync should silently skip so the list command
		// still works during first-time setup.
		return nil, errors.New("tenant not configured -- run 'bctl init' first")
	}

	token, err := requireToken(ctx, t)
	if err != nil {
		return nil, err
	}

	message := "Refreshing profiles from Britive API..."
	if !explicit {
		message = "Profile cache is stale -- syncing from Britive API..."
	}
	spin := output.NewSpinner(message)
	spin.Start()

	client := newAPIClient(t, token)
	entries, err := client.ListAccess(ctx)
	if err != nil {
		spin.Fail(fmt.Sprintf("Failed to fetch profiles: %v", err))
		return nil, err
	}

	profiles := aliases.BuildMap(entries)
	if err := config.SaveProfilesCache(&config.ProfilesCache{Tenant: t, Profiles: profiles}); err != nil {
		spin.Fail("Failed to save profile cache")
		return nil, fmt.Errorf("saving profile cache: %w", err)
	}

	spin.Success(fmt.Sprintf("Synced %d profiles", len(entries)))
	return profiles, nil
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
	if err := config.SaveProfilesCache(&config.ProfilesCache{Tenant: t, Profiles: profiles}); err != nil {
		spin.Fail("Failed to save profile cache")
		return fmt.Errorf("saving profile cache: %w", err)
	}

	spin.Success(fmt.Sprintf("Synced %d profiles", len(entries)))
	return nil
}
