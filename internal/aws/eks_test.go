package aws_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
