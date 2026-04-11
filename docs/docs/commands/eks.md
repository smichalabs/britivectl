# bctl eks

EKS cluster operations.

## Synopsis

```
bctl eks connect <alias>
```

Equivalent to `bctl checkout <alias> --eks`. Both check out the profile and run `aws eks update-kubeconfig` for every cluster on it.

## Example

```bash
bctl eks connect aws-admin-prod
kubectl get pods
```

## Requirements

- The profile must be an AWS profile (the command rejects non-AWS profiles up front)
- The profile must have `eks_clusters` configured -- see the [EKS Guide](../eks.md)
- `aws` and `kubectl` must be on `$PATH`

## See also

- [EKS Guide](../eks.md) -- full setup and multi-cluster behaviour
- [bctl checkout](checkout.md)
