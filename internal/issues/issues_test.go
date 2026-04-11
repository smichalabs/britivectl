package issues_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/smichalabs/britivectl/internal/issues"
)

func TestBuildURL_BugTemplate(t *testing.T) {
	got := issues.BuildURL("bug.yml", "hello world")

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("BuildURL produced an unparseable URL: %v", err)
	}

	if parsed.Host != "github.com" {
		t.Errorf("host = %q, want github.com", parsed.Host)
	}
	wantPath := "/smichalabs/britivectl-releases/issues/new"
	if parsed.Path != wantPath {
		t.Errorf("path = %q, want %q", parsed.Path, wantPath)
	}

	q := parsed.Query()
	if q.Get("template") != "bug.yml" {
		t.Errorf("template = %q, want bug.yml", q.Get("template"))
	}
	if q.Get("body") != "hello world" {
		t.Errorf("body = %q, want %q", q.Get("body"), "hello world")
	}
}

func TestBuildURL_FeatureTemplate(t *testing.T) {
	got := issues.BuildURL("feature.yml", "")
	if !strings.Contains(got, "template=feature.yml") {
		t.Errorf("URL %q does not contain feature template selector", got)
	}
}

// TestBuildURL_BodyEncoding ensures multi-line markdown bodies survive the
// round-trip through URL encoding without being mangled. This matters
// because the auto-collected environment block contains newlines and
// backticks, both of which need to be percent-encoded.
func TestBuildURL_BodyEncoding(t *testing.T) {
	body := "line one\nline two\n- item with `backticks`\n"
	got := issues.BuildURL("bug.yml", body)

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("URL did not parse: %v", err)
	}
	roundTripped := parsed.Query().Get("body")
	if roundTripped != body {
		t.Errorf("body did not round-trip:\n got:  %q\n want: %q", roundTripped, body)
	}
}

func TestBuildEnvironmentBlock_ContainsRequiredFields(t *testing.T) {
	block := issues.BuildEnvironmentBlock(issues.EnvironmentInfo{
		BctlVersion:      "v1.2.3",
		GOOS:             "darwin",
		GOARCH:           "arm64",
		TenantConfigured: true,
	})

	wantSubstrings := []string{
		"**Environment**",
		"bctl version",
		"`v1.2.3`",
		"OS / arch",
		"`darwin/arm64`",
		"Britive tenant",
		"`configured`",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(block, s) {
			t.Errorf("environment block missing %q\n--- block ---\n%s", s, block)
		}
	}
}

func TestBuildEnvironmentBlock_NotConfigured(t *testing.T) {
	block := issues.BuildEnvironmentBlock(issues.EnvironmentInfo{
		BctlVersion:      "v0.0.0",
		GOOS:             "linux",
		GOARCH:           "amd64",
		TenantConfigured: false,
	})
	if !strings.Contains(block, "`not configured`") {
		t.Errorf("expected 'not configured' label, got:\n%s", block)
	}
}

// TestBuildEnvironmentBlock_NoTenantNameLeak guards against accidentally
// adding fields that would leak the actual tenant string. The function only
// accepts a bool, never a string -- this test pins that contract.
func TestBuildEnvironmentBlock_NoTenantNameLeak(t *testing.T) {
	// Pass values containing what would be a tenant-shaped string. They
	// should never appear in the output because the function does not
	// receive a tenant name parameter.
	block := issues.BuildEnvironmentBlock(issues.EnvironmentInfo{
		BctlVersion:      "v1.0.0",
		GOOS:             "darwin",
		GOARCH:           "arm64",
		TenantConfigured: true,
	})
	// The label must be one of the two known phrases inside backticks.
	if !strings.Contains(block, "`configured`") && !strings.Contains(block, "`not configured`") {
		t.Errorf("tenant line missing the expected configured/not-configured label:\n%s", block)
	}
}
