package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/smichalabs/britivectl/internal/aws"
	"github.com/smichalabs/britivectl/internal/britive"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/resolver"
	"github.com/smichalabs/britivectl/internal/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCheckoutCmd() *cobra.Command {
	var (
		eks       bool
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
  1. Exact alias match (e.g. 'dev')
  2. Substring match on alias or Britive path (e.g. 'sandbox')
  3. Fuzzy match as a last resort

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
			return runCheckout(cmd.Context(), query, eks, outputFmt)
		},
	}

	cmd.Flags().BoolVar(&eks, "eks", false, "also update kubeconfig for EKS clusters in this profile")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "", "output format: awscreds|json|env|process")
	return cmd
}

func runCheckout(ctx context.Context, query string, eks bool, outFmt string) error {
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

	// 3. Non-AWS profiles: friendly "coming soon" message instead of a crash.
	// This is an intentional feature gap, not an error -- print and exit 0.
	if match.Profile.Cloud != "aws" {
		printComingSoon(match)
		return nil
	}

	// 4. Checkout via the Britive API.
	if match.Profile.ProfileID == "" || match.Profile.EnvironmentID == "" {
		return fmt.Errorf("profile %q is missing API IDs -- run 'bctl profiles sync' to update", match.Alias)
	}

	spin := output.NewSpinner(fmt.Sprintf("Checking out %s...", match.Alias))
	spin.Start()

	client := newAPIClient(ready.Tenant, ready.Token)
	checkedOut, creds, err := client.Checkout(ctx, match.Profile.ProfileID, match.Profile.EnvironmentID)
	if err != nil {
		spin.Fail(fmt.Sprintf("Checkout failed: %v", err))
		return err
	}
	spin.Success(fmt.Sprintf("Checked out %s (expires: %s)", match.Alias, checkedOut.Expiration))

	// 5. Inject credentials locally.
	if err := injectAWS(match, creds, outFmt); err != nil {
		return err
	}

	// 6. Optional EKS kubeconfig update.
	if eks {
		return connectEKS(ctx, match, creds)
	}
	return nil
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
