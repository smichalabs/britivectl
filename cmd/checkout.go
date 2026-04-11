package cmd

import (
	"context"
	"fmt"

	"github.com/smichalabs/britivectl/internal/aws"
	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newCheckoutCmd() *cobra.Command {
	var (
		eks       bool
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "checkout <alias>",
		Short: "Check out a Britive profile",
		Long: `Check out a Britive profile to obtain temporary credentials.

Aliases are defined in ~/.bctl/config.yaml under the 'profiles' key.
The --output flag controls how credentials are presented:
  awscreds  Write to ~/.aws/credentials (default for AWS profiles)
  json      Print JSON to stdout
  env       Print export VAR=value lines for shell eval
  process   Print AWS credential_process JSON`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckout(cmd.Context(), args[0], eks, outputFmt)
		},
	}

	cmd.Flags().BoolVar(&eks, "eks", false, "also update kubeconfig for EKS clusters in this profile")
	cmd.Flags().StringVarP(&outputFmt, "output", "o", "", "output format: awscreds|json|env|process")
	return cmd
}

func runCheckout(ctx context.Context, alias string, eks bool, outFmt string) error {
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

	// Resolve alias to britive path
	profile, ok := cfg.Profiles[alias]
	if !ok {
		return fmt.Errorf("profile alias %q not found — run 'bctl profiles sync' to update", alias)
	}

	spin := output.NewSpinner(fmt.Sprintf("Checking out %s...", alias))
	spin.Start()

	if profile.ProfileID == "" || profile.EnvironmentID == "" {
		return fmt.Errorf("profile %q is missing API IDs — run 'bctl profiles sync' to update", alias)
	}

	client := newAPIClient(t, token)
	checkedOut, creds, err := client.Checkout(ctx, profile.ProfileID, profile.EnvironmentID)
	if err != nil {
		spin.Fail(fmt.Sprintf("Checkout failed: %v", err))
		return err
	}
	spin.Success(fmt.Sprintf("Checked out %s (expires: %s)", alias, checkedOut.Expiration))

	// Determine output format
	if outFmt == "" {
		outFmt = viper.GetString("output")
	}
	if outFmt == "" && profile.Cloud == "aws" {
		outFmt = "awscreds"
	}
	if outFmt == "" {
		outFmt = "json"
	}
	region := creds.Region
	if region == "" {
		region = profile.Region
	}
	if region == "" {
		region = cfg.DefaultRegion
	}

	switch outFmt {
	case "awscreds":
		awsProfile := profile.AWSProfile
		if awsProfile == "" {
			awsProfile = alias
		}
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
		fallthrough
	default:
		if err := output.PrintJSON(creds); err != nil {
			return err
		}
	}

	// Handle EKS
	if eks {
		awsProfile := profile.AWSProfile
		if awsProfile == "" {
			awsProfile = alias
		}
		for _, cluster := range profile.EKSClusters {
			spin2 := output.NewSpinner(fmt.Sprintf("Updating kubeconfig for %s...", cluster))
			spin2.Start()
			if err := aws.UpdateKubeconfig(cluster, region, awsProfile); err != nil {
				spin2.Fail(fmt.Sprintf("Failed to update kubeconfig for %s: %v", cluster, err))
			} else {
				spin2.Success(fmt.Sprintf("Updated kubeconfig for cluster %s", cluster))
			}
		}
	}

	return nil
}
