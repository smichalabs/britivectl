package aws_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	bctlaws "github.com/smichalabs/britivectl/internal/aws"
)

// makeFakeAWS writes a shell script to tmpDir/aws.
// exitCode controls what the script exits with; stderr is written to stderr
// when exitCode != 0.
func makeFakeAWS(t *testing.T, tmpDir string, exitCode int) {
	t.Helper()
	fakeAWS := filepath.Join(tmpDir, "aws")
	var script string
	if exitCode == 0 {
		script = "#!/bin/sh\nexit 0\n"
	} else {
		script = "#!/bin/sh\necho 'error message' >&2\nexit 1\n"
	}
	if err := os.WriteFile(fakeAWS, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake aws script: %v", err)
	}
}

// prependPath prepends dir to the current PATH for the duration of the test.
func prependPath(t *testing.T, dir string) {
	t.Helper()
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+":"+origPath)
}

func TestUpdateKubeconfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWS(t, tmpDir, 0)
	prependPath(t, tmpDir)

	if err := bctlaws.UpdateKubeconfig(context.Background(), "my-cluster", "us-east-1", "my-profile"); err != nil {
		t.Errorf("UpdateKubeconfig() unexpected error: %v", err)
	}
}

func TestUpdateKubeconfig_NoRegionNoProfile(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWS(t, tmpDir, 0)
	prependPath(t, tmpDir)

	if err := bctlaws.UpdateKubeconfig(context.Background(), "my-cluster", "", ""); err != nil {
		t.Errorf("UpdateKubeconfig() with no region/profile unexpected error: %v", err)
	}
}

func TestUpdateKubeconfig_Error(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWS(t, tmpDir, 1)
	prependPath(t, tmpDir)

	if err := bctlaws.UpdateKubeconfig(context.Background(), "bad-cluster", "us-east-1", "my-profile"); err == nil {
		t.Error("UpdateKubeconfig() expected an error when aws exits 1, got nil")
	}
}

func TestUpdateKubeconfig_WithRegionOnly(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWS(t, tmpDir, 0)
	prependPath(t, tmpDir)

	if err := bctlaws.UpdateKubeconfig(context.Background(), "my-cluster", "ap-southeast-1", ""); err != nil {
		t.Errorf("UpdateKubeconfig() with region only unexpected error: %v", err)
	}
}

func TestUpdateKubeconfig_WithProfileOnly(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWS(t, tmpDir, 0)
	prependPath(t, tmpDir)

	if err := bctlaws.UpdateKubeconfig(context.Background(), "my-cluster", "", "my-profile"); err != nil {
		t.Errorf("UpdateKubeconfig() with profile only unexpected error: %v", err)
	}
}

// makeFakeAWSListClusters writes a fake `aws` script that returns the JSON in
// outputs[N] on the (N+1)-th invocation. The script tracks invocation count
// via a counter file in tmpDir so each test gets a fresh sequence. After
// running out of outputs the script returns the last entry.
//
// Used to simulate the EKS list-clusters propagation race: first call returns
// empty, subsequent calls return the real cluster list.
func makeFakeAWSListClusters(t *testing.T, tmpDir string, outputs []string) {
	t.Helper()
	counter := filepath.Join(tmpDir, "count")
	fakeAWS := filepath.Join(tmpDir, "aws")

	// Build a case statement that picks output by invocation index.
	var cases string
	for i, out := range outputs {
		cases += "  " + intToStr(i) + ") echo '" + out + "' ;;\n"
	}
	// Default to the last output for any further invocations.
	cases += "  *) echo '" + outputs[len(outputs)-1] + "' ;;\n"

	script := "#!/bin/sh\n" +
		"n=$(cat \"" + counter + "\" 2>/dev/null || echo 0)\n" +
		"case \"$n\" in\n" + cases + "esac\n" +
		"echo $((n + 1)) > \"" + counter + "\"\n" +
		"exit 0\n"

	if err := os.WriteFile(fakeAWS, []byte(script), 0o755); err != nil {
		t.Fatalf("writing fake aws script: %v", err)
	}
	if err := os.WriteFile(counter, []byte("0"), 0o600); err != nil {
		t.Fatalf("writing counter: %v", err)
	}
}

// intToStr is a tiny helper to avoid pulling in strconv just for the test
// script generator above.
func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

// withFastBackoffs replaces the package backoff slice with three zeros so
// retry tests do not block on real wall time. Three matches the production
// length; tests that need a different count can set ListClustersBackoffs
// directly. Restored on test cleanup.
func withFastBackoffs(t *testing.T) {
	t.Helper()
	prev := bctlaws.ListClustersBackoffs
	bctlaws.ListClustersBackoffs = []time.Duration{0, 0, 0}
	t.Cleanup(func() { bctlaws.ListClustersBackoffs = prev })
}

func TestListClusters_ReturnsClustersOnFirstCall(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWSListClusters(t, tmpDir, []string{`{"clusters":["prod-eks"]}`})
	prependPath(t, tmpDir)
	withFastBackoffs(t)

	got, err := bctlaws.ListClusters(context.Background(), "us-east-1", "my-profile")
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(got) != 1 || got[0] != "prod-eks" {
		t.Errorf("got %v, want [prod-eks]", got)
	}
}

func TestListClusters_RetriesUntilNonEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	// First call empty, second call returns the cluster -- simulates the
	// STS propagation race the user hit on first checkout.
	makeFakeAWSListClusters(t, tmpDir, []string{
		`{"clusters":[]}`,
		`{"clusters":["prod-eks"]}`,
	})
	prependPath(t, tmpDir)
	withFastBackoffs(t)

	got, err := bctlaws.ListClusters(context.Background(), "us-east-1", "my-profile")
	if err != nil {
		t.Fatalf("ListClusters: %v", err)
	}
	if len(got) != 1 || got[0] != "prod-eks" {
		t.Errorf("got %v, want [prod-eks] after retry", got)
	}
}

func TestListClusters_EmptyAfterAllRetriesReturnsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWSListClusters(t, tmpDir, []string{`{"clusters":[]}`})
	prependPath(t, tmpDir)
	withFastBackoffs(t)

	got, err := bctlaws.ListClusters(context.Background(), "us-east-1", "my-profile")
	if err != nil {
		t.Fatalf("expected nil error on truly-empty result, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestListClusters_HardErrorReturnsImmediately(t *testing.T) {
	tmpDir := t.TempDir()
	makeFakeAWS(t, tmpDir, 1)
	prependPath(t, tmpDir)
	// Even with normal backoffs, this should return immediately because
	// the aws subprocess exits non-zero -- no retry on hard errors.
	withFastBackoffs(t)

	_, err := bctlaws.ListClusters(context.Background(), "us-east-1", "my-profile")
	if err == nil {
		t.Error("expected error from non-zero exit, got nil")
	}
}
