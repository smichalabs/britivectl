# Configuration

## Config file

bctl stores configuration at `~/.config/bctl/config.yaml`.

```yaml
tenant: acme                    # Britive tenant name (acme.britive-app.com)
default_region: us-east-1       # fallback AWS region
auth:
  method: browser               # browser | token

profiles:
  dev:
    profile_id: "abc123"        # set by 'bctl profiles sync'
    env_id: "xyz789"            # set by 'bctl profiles sync'
    britive_path: "App/Env/Profile"
    aws_profile: dev            # ~/.aws/credentials profile name
    cloud: aws
    region: us-east-1
    eks_clusters:
      - my-dev-cluster
  staging:
    profile_id: "def456"
    env_id: "uvw012"
    britive_path: "App/Staging/Profile"
    aws_profile: staging
    cloud: aws
    region: us-west-2
```

!!! tip
    Run `bctl profiles sync` to populate `profile_id` and `env_id` automatically. The other fields can be customised by hand.

## Environment variables

Environment variables override values in the config file.

| Variable | Description |
|----------|-------------|
| `BCTL_TENANT` | Override the tenant name |
| `BCTL_TOKEN` | Use this API token (skips keychain lookup) |
| `BCTL_OUTPUT` | Default output format (`awscreds`, `json`, `env`, `process`) |
| `BCTL_REGION` | Default AWS region |
| `BCTL_NO_COLOR` | Disable colour output |

## Auth methods

### Browser SSO

```bash
bctl init        # set method: browser
bctl login       # opens browser, stores token in keychain
```

Tokens are stored in the OS keychain (macOS Keychain, Linux libsecret) and refreshed automatically when they expire.

### API token

```bash
bctl login --token <token>
```

Or set `BCTL_TOKEN` in your environment to bypass the keychain entirely. Useful for CI pipelines.

## Profile aliases

Aliases are the short names you use with `checkout`, `checkin`, and `eks connect`. They are set by `bctl profiles sync` using a sanitised version of the Britive profile name, but you can rename or add profiles manually:

```yaml
profiles:
  my-alias:           # whatever you want to type
    profile_id: "..."
    env_id: "..."
    aws_profile: my-alias
    cloud: aws
    region: us-east-1
```

## EKS clusters

Add `eks_clusters` to any profile to enable `bctl eks connect` and `bctl checkout --eks`:

```yaml
profiles:
  dev:
    aws_profile: dev
    region: us-east-1
    eks_clusters:
      - dev-cluster-1
      - dev-cluster-2
```

bctl runs `aws eks update-kubeconfig` for each cluster in sequence after checkout.
