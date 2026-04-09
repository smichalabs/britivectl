# bctl checkout

Check out a Britive profile to obtain temporary cloud credentials.

## Synopsis

```
bctl checkout <alias> [flags]
```

## Description

`checkout` contacts the Britive API and issues temporary credentials for the
named profile alias. The alias must be defined in `~/.bctl/config.yaml` under
the `profiles` key.

By default, AWS credentials are written to `~/.aws/credentials`. Use `--output`
to change the format.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--eks` | false | Also update kubeconfig for EKS clusters listed in the profile |
| `-o, --output` | `awscreds` (AWS) | Output format: `awscreds`, `json`, `env`, `process` |

## Output formats

| Format | Description |
|--------|-------------|
| `awscreds` | Write credentials to `~/.aws/credentials` |
| `json` | Print raw JSON to stdout |
| `env` | Print `export VAR=value` lines (eval in shell) |
| `process` | AWS `credential_process`-compatible JSON |

## Examples

```bash
# Check out a profile (writes to ~/.aws/credentials)
bctl checkout dev

# Check out and update kubeconfig for EKS
bctl checkout dev --eks

# Get credentials as shell exports
eval $(bctl checkout dev --output env)

# Use as AWS credential_process
bctl checkout dev --output process

# Get raw JSON
bctl checkout dev --output json
```

## See also

- [bctl checkin](checkin.md)
- [bctl profiles](profiles.md)
- [bctl status](status.md)
