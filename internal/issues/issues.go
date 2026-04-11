// Package issues builds GitHub issue URLs and the auto-collected environment
// block that bctl pre-fills into bug reports and feature requests. The
// package is deliberately small and pure -- no I/O, no global state -- so
// the URL construction and body assembly can be unit tested in isolation
// from the cobra command wiring in cmd/issue.go.
package issues

import (
	"fmt"
	"net/url"
	"strings"
)

// Repo is the GitHub repository where bctl issues are filed. The bctl source
// repo is private, so issues live on the public releases repo where any user
// can reach them without needing source access.
const Repo = "smichalabs/britivectl-releases"

// NewURL is the GitHub "new issue" endpoint that accepts query parameters
// for template selection and body pre-fill.
const NewURL = "https://github.com/" + Repo + "/issues/new"

// BuildURL constructs the new-issue URL with the requested template and body
// pre-filled via query parameters. The body is URL-encoded so multi-line
// markdown (including newlines and backticks) round-trips correctly.
func BuildURL(template, body string) string {
	q := url.Values{}
	q.Set("template", template)
	q.Set("body", body)
	return NewURL + "?" + q.Encode()
}

// EnvironmentInfo is the data BuildEnvironmentBlock needs to render the
// auto-collected environment section of an issue body. It is passed in by
// the caller (cmd/issue.go) so this package does not have to depend on the
// version, runtime, or config packages -- which would make it harder to
// test in isolation.
type EnvironmentInfo struct {
	BctlVersion      string
	GOOS             string
	GOARCH           string
	TenantConfigured bool
}

// BuildEnvironmentBlock returns a markdown snippet describing the local
// environment, intended to be pre-filled into the issue body. The block
// reveals only non-sensitive information: bctl version, OS/arch, and a
// configured/not-configured flag for the Britive tenant. The actual tenant
// name is deliberately NOT included because issues land on a public repo.
func BuildEnvironmentBlock(info EnvironmentInfo) string {
	var sb strings.Builder
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("**Environment** (auto-collected by `bctl issue`)\n\n")
	fmt.Fprintf(&sb, "- bctl version: `%s`\n", info.BctlVersion)
	fmt.Fprintf(&sb, "- OS / arch: `%s/%s`\n", info.GOOS, info.GOARCH)
	fmt.Fprintf(&sb, "- Britive tenant: `%s`\n", tenantLabel(info.TenantConfigured))
	return sb.String()
}

func tenantLabel(configured bool) string {
	if configured {
		return "configured"
	}
	return "not configured"
}
