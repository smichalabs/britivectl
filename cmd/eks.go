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

func newEKSCmd() *cobra.Command {
	eksCmd := &cobra.Command{
		Use:   "eks",
		Short: "EKS cluster operations",
		Long:  "Connect to Amazon EKS clusters via Britive JIT access.",
	}

	eksCmd.AddCommand(newEKSConnectCmd())
	return eksCmd
}

func newEKSConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <alias>",
		Short: "Checkout profile and update kubeconfig for EKS",
		Long:  "Check out a Britive profile and update your local kubeconfig for all associated EKS clusters.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEKSConnect(cmd.Context(), args[0])
		},
	}
}

func runEKSConnect(ctx context.Context, alias string) error {
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

	// Look up the profile and validate it locally BEFORE acquiring a token.
	// requireToken can pop a browser window for SSO re-auth, and we do not
	// want to interrupt the user just to tell them they picked the wrong
	// profile.
	profile, ok := cfg.Profiles[alias]
	if !ok {
		return fmt.Errorf("profile alias %q not found -- run 'bctl profiles sync' first", alias)
	}

	// EKS is an AWS service. Reject non-AWS profiles up front with a clear
	// message rather than letting the command run and fail later.
	if err := requireAWSForEKS(alias, profile); err != nil {
		return err
	}

	token, err := requireToken(ctx, t)
	if err != nil {
		return err
	}

	// Checkout
	spin := output.NewSpinner(fmt.Sprintf("Checking out %s...", alias))
	spin.Start()

	if profile.ProfileID == "" || profile.EnvironmentID == "" {
		return fmt.Errorf("profile %q is missing API IDs — run 'bctl profiles sync' to update", alias)
	}

	client := newAPIClient(t, token)
	_, creds, err := client.Checkout(ctx, profile.ProfileID, profile.EnvironmentID)
	if err != nil {
		spin.Fail(fmt.Sprintf("Checkout failed: %v", err))
		return err
	}
	spin.Success(fmt.Sprintf("Checked out %s", alias))

	// Write credentials
	awsProfile := profile.AWSProfile
	if awsProfile == "" {
		awsProfile = alias
	}
	region := creds.Region
	if region == "" {
		region = profile.Region
	}
	if region == "" {
		region = cfg.DefaultRegion
	}

	if err := aws.WriteCredentials(awsProfile, aws.AWSCredentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Region:          region,
	}); err != nil {
		return fmt.Errorf("writing AWS credentials: %w", err)
	}

	// Update kubeconfig for each cluster. If none are configured, discover
	// what is available in the account/region.
	clusters := profile.EKSClusters
	if len(clusters) == 0 {
		discovered, discoverErr := discoverEKSClusters(ctx, region, awsProfile)
		if discoverErr != nil {
			return discoverErr
		}
		clusters = discovered
	}
	for _, cluster := range clusters {
		spin2 := output.NewSpinner(fmt.Sprintf("Updating kubeconfig for %s...", cluster))
		spin2.Start()
		if err := aws.UpdateKubeconfig(ctx, cluster, region, awsProfile); err != nil {
			spin2.Fail(fmt.Sprintf("Failed: %v", err))
			output.Warning("Continuing despite error on cluster %s", cluster)
		} else {
			spin2.Success(fmt.Sprintf("kubeconfig updated for cluster %s", cluster))
		}
	}

	return nil
}
