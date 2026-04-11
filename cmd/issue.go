package cmd

import (
	"context"
	"fmt"
	"runtime"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/issues"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/system"
	"github.com/smichalabs/britivectl/pkg/version"
	"github.com/spf13/cobra"
)

func newIssueCmd() *cobra.Command {
	issueCmd := &cobra.Command{
		Use:   "issue",
		Short: "File a bug report or feature request",
		Long: `Open a pre-filled GitHub issue in your browser to report a bug or request a feature.

bctl gathers local environment context (version, OS / arch, whether a
Britive tenant is configured) and pre-fills the issue body so you do not
have to type that information manually. The browser opens to the GitHub
new-issue page; you fill in the title and details and click "Submit new
issue". GitHub handles authentication via your existing browser session.

bctl never sees or stores a GitHub token of its own.`,
	}
	issueCmd.AddCommand(newIssueBugCmd())
	issueCmd.AddCommand(newIssueFeatureCmd())
	return issueCmd
}

func newIssueBugCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bug",
		Short: "Report a bug",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runIssue(cmd.Context(), "bug.yml")
		},
	}
}

func newIssueFeatureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "feature",
		Short: "Request a feature",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runIssue(cmd.Context(), "feature.yml")
		},
	}
}

// runIssue builds the new-issue URL with the requested template selected and
// the auto-collected environment block pre-filled, then launches the
// browser. On any browser launch failure, the URL is printed to stdout so
// the user can open it manually -- a UX fallback, not an error condition.
func runIssue(ctx context.Context, template string) error {
	body := issues.BuildEnvironmentBlock(currentEnvironment())
	issueURL := issues.BuildURL(template, body)

	output.Info("Opening browser to file an issue...")
	if err := system.OpenBrowser(ctx, issueURL); err != nil {
		output.Warning("Could not open browser automatically: %v", err)
		fmt.Println()
		fmt.Println("Open this URL manually in your browser:")
		fmt.Println("  " + issueURL)
		return nil
	}
	fmt.Println()
	fmt.Println("If the browser did not open, the URL is:")
	fmt.Println("  " + issueURL)
	return nil
}

// currentEnvironment collects the environment context that gets pre-filled
// into the issue body. It deliberately does NOT include the Britive tenant
// name because issues land on a public repo -- only a configured/not flag.
func currentEnvironment() issues.EnvironmentInfo {
	cfg, err := config.Load()
	tenantConfigured := err == nil && cfg.Tenant != ""

	return issues.EnvironmentInfo{
		BctlVersion:      version.Version,
		GOOS:             runtime.GOOS,
		GOARCH:           runtime.GOARCH,
		TenantConfigured: tenantConfigured,
	}
}
