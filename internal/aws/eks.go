package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// ListClusters calls `aws eks list-clusters` and returns the cluster names.
// Used when --eks is requested but the profile has no EKSClusters configured,
// so bctl can discover what is available rather than silently doing nothing.
func ListClusters(ctx context.Context, region, profile string) ([]string, error) {
	args := []string{"eks", "list-clusters", "--output", "json"}
	if region != "" {
		args = append(args, "--region", region)
	}
	if profile != "" {
		args = append(args, "--profile", profile)
	}

	cmd := exec.CommandContext(ctx, "aws", args...) //nolint:gosec // "aws" is a fixed binary, args are controlled by bctl
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("aws eks list-clusters failed: %w\n%s", err, string(out))
	}

	var result struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing list-clusters output: %w", err)
	}
	return result.Clusters, nil
}

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
