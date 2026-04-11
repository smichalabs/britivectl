# bctl doctor

Diagnose bctl setup issues.

## Synopsis

```
bctl doctor
```

## Description

Runs a series of checks against your local environment and prints a colored
status report:

| Check | What it verifies |
|-------|-----------------|
| Config file | `~/.config/bctl/config.yaml` exists |
| Tenant | `tenant` is set in config or `BCTL_TENANT` |
| Auth token | A token is stored in the OS keychain |
| API connectivity | Can reach `https://{tenant}.britive-app.com` |
| AWS CLI | `aws` binary is on `$PATH` |
| kubectl | `kubectl` binary is on `$PATH` |

## Examples

```bash
bctl doctor
```

Example output:

```
bctl doctor: checking your environment
✓ Config file found at /Users/you/.config/bctl/config.yaml
✓ Tenant configured: acme
✓ Auth token found in keychain
✓ Britive API reachable
✓ AWS CLI found
⚠ kubectl not found: EKS commands will not work
```
