# Quick Start

## First run

```bash
brew tap smichalabs/tap
brew install bctl
bctl
```

On first run, bctl asks for your Britive tenant name and opens your browser for SSO. After that, `bctl` always opens straight to the profile picker. Select a profile, press enter, and credentials are in `~/.aws/credentials`.

## Common workflows

### Check out a profile by name

```bash
bctl checkout aws-admin-prod
```

Substring matches work too -- `bctl checkout admin-prod`, `bctl checkout admin`, and `bctl checkout prod` all resolve to the same profile when there's only one match. If multiple profiles match, the picker opens pre-filtered.

### Check out and update kubeconfig in one step

```bash
bctl checkout aws-admin-prod --eks
kubectl get pods
```

bctl checks out AWS credentials and runs `aws eks update-kubeconfig` for every cluster on the profile. See the [EKS Guide](eks.md) for setup details.

### Export credentials to your shell

```bash
eval "$(bctl checkout aws-admin-prod -o env)"
aws s3 ls
```

The `-o env` output mode prints `export VAR=value` lines instead of writing to `~/.aws/credentials`. Useful for one-off shell sessions, scripts, or CI.

### Use as an AWS credential_process

Add this to `~/.aws/config`:

```ini
[profile aws-admin-prod]
credential_process = bctl checkout aws-admin-prod -o process
```

Now `aws --profile aws-admin-prod ...` invokes bctl transparently whenever credentials are needed. No manual checkout step.

## Next

- [Sessions & caching](sessions.md) explains how bctl auto-refreshes your Britive session and skips redundant API calls
- [Commands](commands/checkout.md) is the full reference for every subcommand
- [Configuration](configuration.md) covers the config file and environment variables
