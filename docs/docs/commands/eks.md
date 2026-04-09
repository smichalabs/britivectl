# bctl eks

EKS cluster operations.

## Synopsis

```
bctl eks connect <alias>
```

## Description

`eks connect` checks out a Britive profile and immediately updates your local
kubeconfig so `kubectl` points at the EKS clusters defined in that profile.

Equivalent to:

```bash
bctl checkout <alias> --eks
```

## Examples

```bash
# Check out credentials and update kubeconfig
bctl eks connect dev

# Then use kubectl normally
kubectl get pods -n default
```

## Prerequisites

- `aws` CLI must be on `$PATH`
- `kubectl` must be on `$PATH`
- The profile must have `eks_clusters` set in `~/.bctl/config.yaml`

## See also

- [bctl checkout](checkout.md)
- [bctl doctor](doctor.md)
