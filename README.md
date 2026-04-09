# bctl — Britive CLI

[![CI](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml/badge.svg)](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A polished CLI for [Britive](https://www.britive.com) JIT access management.
Replace manual web UI workflows and fragile scripts with a single fast binary.

---

## Install

### Homebrew (macOS)

```bash
brew install smichalabs/tap/bctl
```

### curl (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

### From source

```bash
go install github.com/smichalabs/britivectl@latest
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
