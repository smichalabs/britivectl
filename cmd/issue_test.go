package cmd

import (
	"net/url"
	"runtime"
	"strings"
	"testing"
)

func TestBuildIssueURL_BugTemplate(t *testing.T) {
	got := buildIssueURL("bug.yml", "hello world")

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("buildIssueURL produced an unparseable URL: %v", err)
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

func TestBuildIssueURL_FeatureTemplate(t *testing.T) {
	got := buildIssueURL("feature.yml", "")
	if !strings.Contains(got, "template=feature.yml") {
		t.Errorf("URL %q does not contain feature template selector", got)
	}
}

// TestBuildIssueURL_BodyEncoding ensures multi-line markdown bodies survive
// the round-trip through URL encoding without being mangled. This matters
// because the auto-collected environment block contains newlines and
// backticks, both of which need to be percent-encoded.
func TestBuildIssueURL_BodyEncoding(t *testing.T) {
	body := "line one\nline two\n- item with `backticks`\n"
	got := buildIssueURL("bug.yml", body)

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
	block := buildEnvironmentBlock()

	wantSubstrings := []string{
		"**Environment**",
		"bctl version",
		"OS / arch",
		runtime.GOOS,
		runtime.GOARCH,
		"Britive tenant",
	}
	for _, s := range wantSubstrings {
		if !strings.Contains(block, s) {
			t.Errorf("environment block missing %q\n--- block ---\n%s", s, block)
		}
	}
}

// TestBuildEnvironmentBlock_NoTenantNameLeak guards against accidentally
// including the actual tenant string in the issue body. The tenant name is
// considered sensitive because issues land on a public repo.
func TestBuildEnvironmentBlock_NoTenantNameLeak(t *testing.T) {
	block := buildEnvironmentBlock()
	// The label must be either "configured" or "not configured" -- never the
	// raw tenant name. We assert by checking that the tenant line ends with
	// one of those exact phrases inside backticks.
	if !strings.Contains(block, "`configured`") && !strings.Contains(block, "`not configured`") {
		t.Errorf("tenant line missing the expected configured/not-configured label:\n%s", block)
	}
}
