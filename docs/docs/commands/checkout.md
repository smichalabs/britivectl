# bctl checkout

Check out a Britive profile to obtain temporary cloud credentials.

## Synopsis

```
bctl checkout <alias> [flags]
```

## Description

`checkout` contacts the Britive API and issues temporary credentials for the
named profile alias. The alias must be defined in `~/.config/bctl/config.yaml`
under the `profiles` key.

By default, AWS credentials are written to `~/.aws/credentials`. Use `--output`
to change the format.

Pass `--console` to check out web console access instead. bctl requests a
console checkout from Britive and opens the cloud provider's console in your
default browser. No credentials are written locally. If a console checkout of
the profile is already active, bctl reuses it rather than starting a new one.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--eks` | false | Also update kubeconfig for EKS clusters listed in the profile |
| `-o, --output` | `awscreds` (AWS) | Output format: `awscreds`, `json`, `env`, `process` |
| `--console` | false | Check out web console access and open it in the browser |
| `--print-url` | false | Print the console sign-in URL instead of opening a browser (implies `--console`) |

`--console` cannot be combined with `--eks`, `--cluster`, `--region`, `--force`,
or `--output`.

## Console access

Programmatic and console access are separate checkouts in Britive. The profile
must allow console access, and holding one type does not give you the other.

The sign-in URL grants console access to anyone who has it. bctl prints it only
when the browser cannot be opened or when you pass `--print-url`. Do not paste
it into chat or tickets.

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
eval "$(bctl checkout dev --output env)"

# Use as AWS credential_process
bctl checkout dev --output process

# Get raw JSON
bctl checkout dev --output json

# Open the cloud console in the browser
bctl checkout dev --console

# Print the console sign-in URL (for example over SSH)
bctl checkout dev --print-url
```

## See also

- [Console Access](../console.md)
- [bctl checkin](checkin.md)
- [bctl profiles](profiles.md)
- [bctl status](status.md)
