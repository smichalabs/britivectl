package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/smichalabs/britivectl/internal/aws"
	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/resolver"
	"github.com/smichalabs/britivectl/internal/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// freshnessBuffer is how much head room we require on cached credentials
// before we trust them. Anything closer to expiry than this triggers a
// fresh checkout so downstream tools do not get half-dead credentials.
const freshnessBuffer = 5 * time.Minute

// errUnsupportedCloud signals that the resolved profile is for a cloud bctl
// does not yet support (GCP, Azure). The checkout command catches this to
// suppress cobra's default error print -- printComingSoon already rendered a
// user-friendly message -- while still exiting non-zero so scripts can tell
// the checkout did not produce credentials.
var errUnsupportedCloud = errors.New("profile cloud is not supported")

func newCheckoutCmd() *cobra.Command {
	var (
		eks       bool
		force     bool
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "checkout [alias]",
		Short: "Check out a Britive profile (auto-reconciles state)",
		Long: `Check out a Britive profile and obtain temporary cloud credentials.

This command is a one-stop orchestrator: it will init, login, and sync
profiles on demand if anything is missing. The alias is optional -- if
omitted, you'll get an interactive picker.

Matching rules for the alias:
  1. Exact alias match (e.g. 'aws-admin-prod')
  2. Substring match on alias or Britive path (e.g. 'sandbox')
  3. Fuzzy match as a last resort

If credentials for the profile were already checked out and have at
least 5 minutes of life left, bctl skips the Britive API entirely and
reports the existing expiry. Pass --force to refresh anyway.

Output formats (--output / -o):
  awscreds  Write to ~/.aws/credentials (default for AWS profiles)
  json      Print JSON to stdout
  env       Print export VAR=value lines for shell eval
  process   Print AWS credential_process JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) == 1 {
				query = args[0]
			}
			err := runCheckout(cmd.Context(), query, eks, force, outputFmt)
			if errors.Is(err, errUnsupportedCloud) {
				// Message already printed by printComingSoon; suppress cobra's
				// duplicate "Error: ..." line but keep the non-zero exit.
				cmd.SilenceErrors = true
			}
			return err
		},
	}

	cmd.Flags().BoolVar(&eks, "eks", false, "also update kubeconfig for EKS clusters in this profile")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "refresh credentials even if existing ones are still valid")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "", "output format: awscreds|json|env|process")
	return cmd
}

func runCheckout(ctx context.Context, query string, eks, force bool, outFmt string) error {
	// 1. Reconcile state: config, token, profile cache.
	ready, err := state.EnsureReady(ctx, stateCallbacks())
	if err != nil {
		return err
	}

	// 2. Resolve the user's query to a single profile.
	match, err := resolver.Resolve(ctx, ready.Profiles, query, os.Stdin, os.Stdout)
	if err != nil {
		if errors.Is(err, resolver.ErrCanceled) {
			output.Info("Canceled.")
			return nil
		}
		return err
	}

	// 3a. EKS was explicitly requested but the profile is not AWS. EKS is an
	// AWS service, so this can never work -- fail fast with the specific
	// error rather than the generic "coming soon" message.
	if eks {
		if err := requireAWSForEKS(match.Alias, match.Profile); err != nil {
			return err
		}
	}

	// 3b. Non-AWS profiles: print the friendly "coming soon" message and exit
	// non-zero. Scripts that wrap bctl need the exit code to reflect reality
	// -- "no credentials were produced" must not look like success.
	if match.Profile.Cloud != "aws" {
		printComingSoon(match)
		return errUnsupportedCloud
	}

	// 4. Skip-if-fresh: if a previous checkout is still valid, do not bother
	// hitting the Britive API again. The user can pass --force to override.
	// This is suppressed for output formats that need fresh credentials in
	// stdout (env, process, json) -- those callers want the actual values
	// printed, not a "still valid" message.
	if !force && outFmtWritesAWSCredsFile(outFmt) {
		if cached, err := config.LoadCheckoutState(match.Alias); err == nil && cached.IsFresh(freshnessBuffer) {
			output.Success("%s is already checked out (expires in %s)", match.Alias, formatDuration(cached.Remaining()))
			fmt.Println("Use --force to refresh now.")
			if eks {
				// Even on a fresh-cache hit we still want kubeconfig to be
				// up to date in case clusters changed since the last run.
				return connectEKSFromProfile(ctx, match)
			}
			return nil
		}
	}

	// 5. Checkout via the Britive API.
	if match.Profile.ProfileID == "" || match.Profile.EnvironmentID == "" {
		return fmt.Errorf("profile %q is missing API IDs -- run 'bctl profiles sync' to update", match.Alias)
	}

	client := newAPIClient(ready.Tenant, ready.Token)

	// 5a. Check Britive for an already-active session before attempting a new
	// checkout. This handles the case where the user already has credentials
	// live on the Britive side but no local cache (e.g. same profile used from
	// a different project directory or a different machine). Without this,
	// Britive returns HTTP 400 "already checked out" and the user sees a
	// confusing error for what should be a success.
	if !force {
		existing, err := findActiveSession(ctx, client, match.Profile.ProfileID)
		switch {
		case err == nil:
			return reuseExistingSession(ctx, client, match, existing, eks, outFmt)
		case errors.Is(err, errNoActiveSession):
			// No active session; fall through to a fresh checkout.
		default:
			// MySessions failed for another reason. Fall through and let the
			// fresh checkout surface a real error if the API is broken.
		}
	}

	spin := output.NewSpinner(fmt.Sprintf("Checking out %s...", match.Alias))
	spin.Start()

	checkedOut, creds, err := client.Checkout(ctx, match.Profile.ProfileID, match.Profile.EnvironmentID)
	if err != nil {
		spin.Fail(fmt.Sprintf("Checkout failed: %v", err))
		return err
	}
	spin.Success(fmt.Sprintf("Checked out %s (expires: %s)", match.Alias, checkedOut.Expiration))

	// 6. Persist the freshness state for next time.
	if err := saveCheckoutState(match.Alias, checkedOut.TransactionID, checkedOut.Expiration); err != nil {
		// Non-fatal -- the credentials are valid even if we cannot record
		// the cache. Print a warning so the user can see what happened.
		output.Warning("could not save checkout cache: %v", err)
	}

	// 7. Inject credentials locally.
	if err := injectAWS(match, creds, outFmt); err != nil {
		return err
	}

	// 8. Optional EKS kubeconfig update.
	if eks {
		return connectEKS(ctx, match, creds)
	}
	return nil
}

// errNoActiveSession indicates findActiveSession looked at all of the user's
// current sessions and none of them matched the requested profile. Distinct
// from a real API failure -- the caller treats this as "do a fresh checkout"
// rather than as an error to surface.
var errNoActiveSession = errors.New("no active session for profile")

// findActiveSession returns the active Britive session for the given profile
// ID. Returns errNoActiveSession (wrapped via errors.Is) if the user has no
// active checkout for this profile, or the API error if MySessions failed.
// Never returns (nil, nil) so callers always have a definite signal.
func findActiveSession(ctx context.Context, client *britive.Client, profileID string) (*britive.CheckedOutProfile, error) {
	sessions, err := client.MySessions(ctx)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		s := &sessions[i]
		if s.CheckedIn == nil && s.PapID == profileID {
			return s, nil
		}
	}
	return nil, errNoActiveSession
}

// reuseExistingSession fetches credentials for an already-active checkout and
// injects them locally without creating a new checkout on the Britive side.
// Britive does not allow two concurrent checkouts for the same profile, so
// when credentials are already live we grab them and move on.
func reuseExistingSession(ctx context.Context, client *britive.Client, match resolver.Match, existing *britive.CheckedOutProfile, eks bool, outFmt string) error {
	creds, err := client.GetCredentials(ctx, existing.TransactionID)
	if err != nil {
		return fmt.Errorf("fetching credentials for existing checkout: %w", err)
	}

	output.Success("Reusing existing checkout for %s (expires: %s)", match.Alias, existing.Expiration)

	if err := saveCheckoutState(match.Alias, existing.TransactionID, existing.Expiration); err != nil {
		output.Warning("could not save checkout cache: %v", err)
	}

	if err := injectAWS(match, creds, outFmt); err != nil {
		return err
	}

	if eks {
		return connectEKS(ctx, match, creds)
	}
	return nil
}

// outFmtWritesAWSCredsFile reports whether the requested output format
// puts credentials into ~/.aws/credentials (the only case where the
// skip-if-fresh shortcut is correct). Other formats need to print the live
// values to stdout, which means we must actually call Britive.
func outFmtWritesAWSCredsFile(outFmt string) bool {
	if outFmt == "" {
		outFmt = viper.GetString("output")
	}
	if outFmt == "" {
		outFmt = "awscreds"
	}
	return outFmt == "awscreds"
}

// saveCheckoutState persists the just-completed checkout so that subsequent
// invocations can skip the Britive API while the credentials are still
// valid. The expiration string comes from the Britive API; if it cannot be
// parsed, the cache is skipped (the checkout itself still succeeded).
func saveCheckoutState(alias, txnID, expiration string) error {
	expiresAt, err := time.Parse(time.RFC3339, expiration)
	if err != nil {
		// Britive sometimes uses subtly different formats. Try a couple
		// of common ones before giving up.
		for _, layout := range []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.000Z",
			time.RFC3339Nano,
		} {
			if t, e := time.Parse(layout, expiration); e == nil {
				expiresAt = t
				err = nil
				break
			}
		}
	}
	if err != nil {
		return fmt.Errorf("parsing expiration %q: %w", expiration, err)
	}

	return config.SaveCheckoutState(&config.CheckoutState{
		Alias:         alias,
		TransactionID: txnID,
		CheckedOutAt:  time.Now().UTC(),
		ExpiresAt:     expiresAt.UTC(),
	})
}

// formatDuration renders a duration in a way humans actually read at a
// glance: "3h 47m", "12m", "30s". Anything sub-second collapses to "<1s".
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) - hours*60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// connectEKSFromProfile updates kubeconfig using whatever profile/region the
// user has configured locally. Used on the skip-if-fresh path where we did
// not just check out new credentials and therefore have no live region.
func connectEKSFromProfile(ctx context.Context, match resolver.Match) error {
	if len(match.Profile.EKSClusters) == 0 {
		return nil
	}
	awsProfile := match.Profile.AWSProfile
	if awsProfile == "" {
		awsProfile = match.Alias
	}
	region := match.Profile.Region

	for _, cluster := range match.Profile.EKSClusters {
		spin := output.NewSpinner(fmt.Sprintf("Updating kubeconfig for %s...", cluster))
		spin.Start()
		if err := aws.UpdateKubeconfig(ctx, cluster, region, awsProfile); err != nil {
			spin.Fail(fmt.Sprintf("Failed to update kubeconfig for %s: %v", cluster, err))
		} else {
			spin.Success(fmt.Sprintf("Updated kubeconfig for cluster %s", cluster))
		}
	}
	return nil
}

// requireAWSForEKS validates that a profile is an AWS profile before any EKS
// kubeconfig work is attempted. Returns nil for AWS profiles. For anything
// else (GCP, Azure, blank cloud), prints a clear explanation and returns an
// error so the caller exits non-zero.
//
// EKS is an AWS service. There is no equivalent thing to do for GCP or Azure
// profiles, so we fail fast with a useful message rather than calling the
// Britive API only to fail at `aws eks update-kubeconfig` later.
func requireAWSForEKS(alias string, profile config.Profile) error {
	if strings.EqualFold(profile.Cloud, "aws") {
		return nil
	}
	cloud := profile.Cloud
	if cloud == "" {
		cloud = "non-AWS"
	}

	output.Error("EKS only works with AWS profiles. %q is a %s profile.", alias, cloud)
	fmt.Println()
	fmt.Printf("  alias:        %s\n", alias)
	fmt.Printf("  britive path: %s\n", profile.BritivePath)
	fmt.Printf("  cloud:        %s\n", cloud)
	fmt.Println()
	fmt.Println("EKS clusters are an AWS service. Pick an AWS profile, or run")
	fmt.Println("'bctl checkout' without the --eks flag.")
	return fmt.Errorf("EKS requires an AWS profile, got %s", cloud)
}

// printComingSoon prints a friendly message explaining that a cloud other
// than AWS is not yet implemented. The profile was still resolved correctly
// -- only local credential injection is missing. Called for GCP and Azure
// profiles; returns no error because this is an intentional feature gap.
func printComingSoon(match resolver.Match) {
	cloud := match.Profile.Cloud
	if cloud == "" {
		cloud = "unknown"
	}

	output.Info("Profile %q resolved to a %s profile.", match.Alias, cloud)
	fmt.Println()
	fmt.Printf("  alias:        %s\n", match.Alias)
	fmt.Printf("  britive path: %s\n", match.Profile.BritivePath)
	fmt.Printf("  cloud:        %s\n", cloud)
	fmt.Println()
	output.Warning("%s credential injection is coming soon.", cloud)
	fmt.Println("bctl currently injects only AWS credentials. GCP and Azure support")
	fmt.Println("is on the roadmap -- see https://smichalabs.dev/utils/bctl/")
}

// injectAWS writes the checkout credentials to the location dictated by the
// requested output format.
func injectAWS(match resolver.Match, creds *britive.Credentials, outFmt string) error {
	if outFmt == "" {
		outFmt = viper.GetString("output")
	}
	if outFmt == "" {
		outFmt = "awscreds"
	}

	cfg, _ := config.Load()
	region := creds.Region
	if region == "" {
		region = match.Profile.Region
	}
	if region == "" && cfg != nil {
		region = cfg.DefaultRegion
	}

	awsProfile := match.Profile.AWSProfile
	if awsProfile == "" {
		awsProfile = match.Alias
	}

	switch outFmt {
	case "awscreds":
		if err := aws.WriteCredentials(awsProfile, aws.AWSCredentials{
			AccessKeyID:     creds.AccessKeyID,
			SecretAccessKey: creds.SecretAccessKey,
			SessionToken:    creds.SessionToken,
			Region:          region,
		}); err != nil {
			return fmt.Errorf("writing AWS credentials: %w", err)
		}
		output.Success("Credentials written to ~/.aws/credentials [%s]", awsProfile)
	case "env":
		output.PrintEnv(map[string]string{
			"AWS_ACCESS_KEY_ID":     creds.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY": creds.SecretAccessKey,
			"AWS_SESSION_TOKEN":     creds.SessionToken,
			"AWS_DEFAULT_REGION":    region,
		})
	case "process":
		output.PrintAWSCredsProcess(map[string]string{
			"AccessKeyId":     creds.AccessKeyID,
			"SecretAccessKey": creds.SecretAccessKey,
			"SessionToken":    creds.SessionToken,
			"Expiration":      creds.Expiration,
		})
	case "json":
		if err := output.PrintJSON(creds); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output format %q", outFmt)
	}
	return nil
}

// connectEKS updates kubeconfig for every EKS cluster listed on the profile.
// Errors are reported per-cluster but do not stop processing of the next.
func connectEKS(ctx context.Context, match resolver.Match, creds *britive.Credentials) error {
	if len(match.Profile.EKSClusters) == 0 {
		return nil
	}

	awsProfile := match.Profile.AWSProfile
	if awsProfile == "" {
		awsProfile = match.Alias
	}
	region := creds.Region
	if region == "" {
		region = match.Profile.Region
	}

	for _, cluster := range match.Profile.EKSClusters {
		spin := output.NewSpinner(fmt.Sprintf("Updating kubeconfig for %s...", cluster))
		spin.Start()
		if err := aws.UpdateKubeconfig(ctx, cluster, region, awsProfile); err != nil {
			spin.Fail(fmt.Sprintf("Failed to update kubeconfig for %s: %v", cluster, err))
		} else {
			spin.Success(fmt.Sprintf("Updated kubeconfig for cluster %s", cluster))
		}
	}
	return nil
}
