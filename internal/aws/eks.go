package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// ListClustersBackoffs is the wait pattern between ListClusters retry attempts.
// AWS STS credentials minted by Britive (or any AssumeRole flow) have a brief
// propagation window during which the EKS control plane can silently return
// an empty cluster list even when the role has access -- the credentials look
// valid to STS but the EKS endpoint has not seen them yet. The first attempt
// runs immediately; subsequent attempts wait the corresponding delay before
// retrying. Total worst-case latency for a genuinely empty result is the sum
// of the non-zero entries.
//
// Exposed as a package var so tests can shorten it without changing behavior.
var ListClustersBackoffs = []time.Duration{0, time.Second, 2 * time.Second}

// ListClusters calls `aws eks list-clusters` and returns the cluster names.
// Used when --eks is requested but the profile has no EKSClusters configured,
// so bctl can discover what is available rather than silently doing nothing.
//
// Retries on an empty result with a small backoff (see ListClustersBackoffs).
// This handles the STS-propagation race that follows a fresh Britive checkout:
// without the retry, the first `--eks` call after a new checkout often gets
// an empty list and the second one a few seconds later sees the cluster fine.
// Hard errors (non-zero exit, parse failure) are returned immediately without
// retry -- the retry exists for the propagation case, not for real failures.
func ListClusters(ctx context.Context, region, profile string) ([]string, error) {
	var clusters []string
	for i, delay := range ListClustersBackoffs {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		var err error
		clusters, err = listClustersOnce(ctx, region, profile)
		if err != nil {
			return nil, err
		}
		if len(clusters) > 0 {
			return clusters, nil
		}
		// Empty result. Retry unless this was the last attempt.
		_ = i
	}
	return clusters, nil
}

// listClustersOnce is the single-attempt body of ListClusters, exported only
// for testability of the no-retry path.
func listClustersOnce(ctx context.Context, region, profile string) ([]string, error) {
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
