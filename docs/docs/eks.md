# EKS Guide

bctl has first-class support for Amazon EKS. It handles the full checkout + kubeconfig update in a single command.

## Prerequisites

- `aws` CLI installed and in PATH
- `kubectl` installed and in PATH
- Profile has `eks_clusters` configured (see below)

Run `bctl doctor` to verify both are available.

## Configure EKS clusters

Add `eks_clusters` to a profile in `~/.bctl/config.yaml`:

```yaml
profiles:
  dev:
    profile_id: "..."
    env_id: "..."
    aws_profile: dev
    region: us-east-1
    eks_clusters:
      - my-dev-cluster
      - my-dev-cluster-2   # multiple clusters supported
```

## Connect

```bash
bctl eks connect dev
```

This does three things in sequence:

1. Checks out the Britive profile (temporary AWS credentials)
2. Writes credentials to `~/.aws/credentials`
3. Runs `aws eks update-kubeconfig` for each cluster in `eks_clusters`

Then `kubectl` is immediately ready:

```bash
kubectl get pods
kubectl get nodes
```

## Checkout with EKS flag

Alternatively, use the `--eks` flag on checkout to do the same thing:

```bash
bctl checkout dev --eks
```

The difference: `bctl eks connect` is dedicated to EKS workflows and always writes `awscreds`. `bctl checkout --eks` lets you combine EKS with other output formats.

## Multi-cluster

If a profile has multiple clusters, bctl updates kubeconfig for each in sequence:

```
✓ Checked out dev
✓ kubeconfig updated for cluster dev-cluster-1
✓ kubeconfig updated for cluster dev-cluster-2
```

If one cluster fails (e.g. wrong region), bctl logs the error and continues to the next.

## Region

The region used for `aws eks update-kubeconfig` is resolved in this order:

1. `region` in the credentials response from Britive
2. `region` in the profile config
3. `default_region` in the top-level config
