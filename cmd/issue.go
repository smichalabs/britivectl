package cmd

import (
	"context"
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/smichalabs/britivectl/internal/config"
	"github.com/smichalabs/britivectl/internal/output"
	"github.com/smichalabs/britivectl/internal/system"
	"github.com/smichalabs/britivectl/pkg/version"
	"github.com/spf13/cobra"
)

// issueRepo is where bug reports and feature requests are filed. The source
// repo (smichalabs/britivectl) is private, so issues live on the public
// releases repo where any bctl user can reach them without needing access to
// closed source.
const issueRepo = "smichalabs/britivectl-releases"

// issueNewURL is the GitHub "new issue" endpoint that accepts query params
// for template selection and body pre-fill.
const issueNewURL = "https://github.com/" + issueRepo + "/issues/new"

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
// the auto-collected environment block pre-filled, then launches the browser.
// On any browser launch failure, the URL is printed to stdout so the user can
// open it manually -- this is a UX fallback, not an error condition.
func runIssue(ctx context.Context, template string) error {
	body := buildEnvironmentBlock()
	issueURL := buildIssueURL(template, body)

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

// buildIssueURL constructs the new-issue URL with the template and body
// query parameters set. Split out from runIssue so it can be unit-tested
// without launching a browser.
func buildIssueURL(template, body string) string {
	q := url.Values{}
	q.Set("template", template)
	q.Set("body", body)
	return issueNewURL + "?" + q.Encode()
}

// buildEnvironmentBlock returns a markdown snippet describing the local
// environment, intended to be pre-filled into the issue body. The block
// reveals only non-sensitive information: bctl version, OS / arch, and
// whether a Britive tenant is configured (the actual tenant name is
// deliberately omitted because issues are filed on a public repo).
func buildEnvironmentBlock() string {
	var sb strings.Builder
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("**Environment** (auto-collected by `bctl issue`)\n\n")
	sb.WriteString(fmt.Sprintf("- bctl version: `%s`\n", version.Version))
	sb.WriteString(fmt.Sprintf("- OS / arch: `%s/%s`\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("- Britive tenant: `%s`\n", tenantConfiguredLabel()))
	return sb.String()
}

// tenantConfiguredLabel reports whether a Britive tenant is set in the local
// config without exposing the tenant name itself. Issues land on a public
// repo, so the actual tenant string never leaves the user's machine.
func tenantConfiguredLabel() string {
	cfg, err := config.Load()
	if err != nil || cfg.Tenant == "" {
		return "not configured"
	}
	return "configured"
}
