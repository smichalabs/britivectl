package aws

import (
	"context"
	"fmt"
	"os/exec"
)

// UpdateKubeconfig runs `aws eks update-kubeconfig` to add the cluster to kubeconfig.
func UpdateKubeconfig(cluster, region, profile string) error {
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

	cmd := exec.CommandContext(context.Background(), "aws", args...) //nolint:gosec // "aws" is a fixed binary, args are controlled by bctl
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aws eks update-kubeconfig failed: %w\n%s", err, string(out))
	}
	return nil
}
