# bctl -- Britive CLI

[![CI](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml/badge.svg)](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A polished CLI for [Britive](https://www.britive.com) JIT access management.
Replace manual web UI workflows and fragile scripts with a single fast binary.

Full docs: [smichalabs.dev/utils/bctl](https://smichalabs.dev/utils/bctl/)

## Cloud support

| Cloud | Profile listing | Credential injection | Cluster access |
|---|---|---|---|
| AWS   | Available now | `~/.aws/credentials` and env vars | EKS via `bctl checkout --eks` |
| GCP   | Available now | Coming soon | GKE coming soon |
| Azure | Available now | Coming soon | AKS coming soon |

`bctl profiles sync` / `bctl profiles list` show profiles from all three clouds today. GCP and Azure credential injection is on the roadmap -- running `bctl checkout` against a non-AWS profile prints a friendly message with the profile details so you know it's recognized.

---

## AWS credentials

**Web UI (manual)**

1. Log into the Britive web portal
2. Navigate apps -> environment -> profile
3. Click checkout, pick a duration
4. Copy three values from the popup: access key ID, secret access key, session token
5. Paste into `~/.aws/credentials` under a profile name (or `export` in your shell)
6. Run `aws ...`
7. Credentials expire (typically 1 hour) -- repeat from step 1

**pybritive**

```bash
pip install pybritive[aws]
pybritive configure tenant -t acme
pybritive login
pybritive checkout "AWS/Sandbox/Developer" -m integrate
aws s3 ls --profile dev
```

Works, but you type the full Britive path every time. AWS integration requires the `-m integrate` flag and the Python install brings ~100 MB of dependencies.

**bctl**

```bash
brew install smichalabs/tap/bctl
bctl checkout dev
aws s3 ls --profile dev
```

One command. On first run `bctl checkout` walks you through tenant setup,
opens the browser for SSO, syncs your profiles, and writes credentials to
`~/.aws/credentials`. Subsequent runs skip every step that's already done.
Use short aliases (`dev`) or substring matches (`sandbox`). Single static
binary, no Python runtime.

---

## EKS access

**Web UI (manual)**

1. Log into the Britive web portal, check out the profile
2. Copy AWS credentials into `~/.aws/credentials`
3. Run `aws eks update-kubeconfig --region <region> --name <cluster> --profile <profile>`
4. Repeat step 3 for every cluster on the profile
5. Run `kubectl ...`
6. Credentials expire -- repeat from step 1

**pybritive**

```bash
pybritive checkout "AWS/Sandbox/Developer" -m integrate
aws eks update-kubeconfig --region us-east-1 --name my-cluster --profile dev
kubectl get pods
```

Two manual steps. pybritive does the credential checkout, but you still wire up `aws eks update-kubeconfig` yourself for every cluster.

**bctl**

```bash
bctl checkout dev --eks
kubectl get pods
```

One command. bctl checks out credentials and runs `aws eks update-kubeconfig` for every cluster on the profile, in the right region, with the right profile name. Credentials and kubeconfig stay in sync.

---

## Install

### macOS — Homebrew

```bash
brew install smichalabs/tap/bctl
```

### Linux / WSL — apt (Debian, Ubuntu)

```bash
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

The script auto-detects your distro and installs the right package:
- Debian/Ubuntu/WSL → `.deb` via `dpkg`
- RHEL/Fedora/CentOS → `.rpm` via `dnf`/`rpm`
- Everything else → tarball to `/usr/local/bin`

### Build from source

Requires access to this repository:

```bash
git clone https://github.com/smichalabs/britivectl.git
cd britivectl
make install
```

---

## Quick start

```bash
# 1. Set up your tenant and auth method
bctl init

# 2. Log in (browser SSO or token)
bctl login

# 3. Sync available profiles
bctl profiles sync

# 4. Check out credentials
bctl checkout dev

# 5. Check status
bctl status

# 6. Return credentials early
bctl checkin dev
```

---

## Commands

| Command | Description |
|---------|-------------|
| `bctl init` | Interactive setup wizard |
| `bctl login [--token <t>]` | Authenticate (browser SSO or API token) |
| `bctl logout` | Clear stored credentials |
| `bctl checkout <alias>` | Check out a profile, write credentials |
| `bctl checkout <alias> --eks` | Checkout + update kubeconfig |
| `bctl checkout <alias> -o env` | Checkout + print shell exports |
| `bctl checkin <alias>` | Return credentials early |
| `bctl status` | Show active checkouts and expiry |
| `bctl profiles list` | List configured profiles |
| `bctl profiles sync` | Pull latest profiles from Britive API |
| `bctl eks connect <alias>` | Checkout + update kubeconfig |
| `bctl config get <key>` | Read a config value |
| `bctl config set <key> <value>` | Write a config value |
| `bctl doctor` | Diagnose setup issues |
| `bctl update` | Self-update to latest release |
| `bctl version` | Print version info |
| `bctl completion [bash\|zsh\|fish]` | Generate shell completions |

### Checkout output formats

```bash
bctl checkout dev                     # write to ~/.aws/credentials (default for AWS)
bctl checkout dev -o json             # raw JSON to stdout
bctl checkout dev -o env              # eval-able shell exports
bctl checkout dev -o process          # AWS credential_process JSON
bctl checkout dev -o awscreds         # explicit ~/.aws/credentials write
```

---

## Configuration

Config file: `~/.bctl/config.yaml`

```yaml
tenant: acme                    # your Britive tenant name
default_region: us-east-1
auth:
  method: browser               # browser | token

profiles:
  dev:
    britive_path: "app/env/profile"
    aws_profile: dev
    cloud: aws
    region: us-east-1
    eks_clusters:
      - my-dev-cluster
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `BCTL_TENANT` | Override tenant from config |
| `BCTL_TOKEN` | Use this API token (skips keychain) |
| `BCTL_OUTPUT` | Default output format |
| `BCTL_REGION` | Default AWS region |
| `BCTL_NO_COLOR` | Disable color output |

---

## Shell completions

```bash
# bash
bctl completion bash > /usr/local/etc/bash_completion.d/bctl

# zsh
bctl completion zsh > "${fpath[1]}/_bctl"

# fish
bctl completion fish > ~/.config/fish/completions/bctl.fish
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Bug reports and feature requests go in
[GitHub Issues](https://github.com/smichalabs/britivectl/issues).

---

## License

[MIT](LICENSE) © smichalabs
