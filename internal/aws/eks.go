package aws

import (
	"context"
	"fmt"
	"os/exec"
)

// UpdateKubeconfig runs `aws eks update-kubeconfig` to add the cluster to kubeconfig.
// The context controls cancellation for the aws subprocess.
func UpdateKubeconfig(ctx context.Context, cluster, region, profile string) error {
	args := []string{
		"eks", "update-kubeconfig",
		"--name", cluster,
	}
	if region != "" {
		args = append(args, "--region", region)
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.CommandContext(ctx, "aws", args...) //nolint:gosec // "aws" is a fixed binary, args are controlled by bctl
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws eks update-kubeconfig failed: %w\n%s", err, string(out))
	}
	return nil
}
