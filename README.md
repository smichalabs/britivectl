# bctl -- Britive CLI

[![CI](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml/badge.svg)](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A fast, polished command-line tool for [Britive](https://www.britive.com) JIT access. Get temporary cloud credentials on your laptop with two keystrokes.

Full docs: [smichalabs.dev/utils/bctl](https://smichalabs.dev/utils/bctl/)

---

## Install

**macOS**

```bash
brew install smichalabs/tap/bctl
```

**Linux / WSL**

```bash
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

Auto-detects your distro and installs the matching `.deb`, `.rpm`, or tarball.

---

## Using it

```bash
bctl
```

That's the whole command. `bctl` on its own opens a searchable launcher with every action it can do:

```
┌ bctl -- pick a command (type to filter, enter to run, esc to cancel) ┐
│                                                                      │
│ > checkout    Check out a Britive profile                            │
│   status      Show active profile checkouts                          │
│   checkin     Return a checked-out profile early                     │
│   profiles    Manage Britive profiles                                │
│   eks         EKS cluster operations                                 │
│   login       Authenticate with Britive                              │
│   ...                                                                │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

`checkout` is already highlighted. Press **enter** and bctl shows the profile picker, which filters live as you type:

```
┌ Pick a profile (type to filter, enter to select, esc to cancel) ┐
│                                                                 │
│ > llmg-admin-prod       [aws]  AWS/Prod/LLMG Admin              │
│   llmg-admin-nonprod    [aws]  AWS/NonProd/LLMG Admin           │
│   mcpg-admin-nonprod    [aws]  AWS/NonProd/MCPG Admin           │
│   sectools-admin-nonprod [aws] AWS/NonProd/SecTools Admin       │
│   gcp-see-admin-sandbox [gcp]  GCP/Sandbox/SEE Admin            │
│   ...                                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Type `llmg prod`, `sandbox`, or `mcpg` and the list narrows instantly. Hit **enter**, and credentials land in `~/.aws/credentials` automatically.

```bash
aws s3 ls --profile llmg-admin-prod
```

Done. The first time you run bctl on a fresh machine it walks you through tenant setup, opens your browser for SSO, and fetches your profile list. Every run after that skips to the picker.

### Skip the picker if you know what you want

```bash
bctl checkout llmg-admin-prod
```

Partial matches work too. All three of these resolve to `llmg-admin-prod`:

```bash
bctl checkout llmg-prod
bctl checkout llmg
bctl checkout prod
```

### EKS clusters

```bash
bctl checkout llmg-admin-prod --eks
kubectl get pods
```

One command. bctl checks out credentials **and** updates your kubeconfig for every cluster on the profile.

---

## Supported clouds

| Cloud | Status |
|---|---|
| AWS   | Fully supported -- credentials written to `~/.aws/credentials` |
| GCP   | Browse and resolve profiles today. Credential injection is coming next. |
| Azure | Browse and resolve profiles today. Credential injection is coming next. |

---

## Why bctl instead of the Britive web UI or pybritive?

**Britive web UI:** log in, click apps, click environment, click profile, click checkout, copy three values from a popup, paste them into `~/.aws/credentials` or export them in your shell, then run `aws ...`. Credentials expire in an hour. Repeat.

**pybritive:**

```bash
pip install pybritive[aws]
pybritive configure tenant -t acme
pybritive login
pybritive checkout "AWS/Prod/LLMG Admin" -m integrate
aws s3 ls --profile dev
```

Works, but you memorize and type the full Britive path every time, and carry a ~100 MB Python stack.

**bctl:**

```bash
bctl
```

Arrow keys or fuzzy search. Single 9 MB binary, no runtime, no paths to memorize.

---

## All commands

You rarely need these directly -- the launcher shows them all -- but if you want to script around bctl:

| Command | What it does |
|---|---|
| `bctl` | Open the command launcher |
| `bctl checkout [name]` | Check out a profile (opens picker if omitted) |
| `bctl checkout [name] --eks` | Check out + update kubeconfig for EKS clusters |
| `bctl status` | Show active checkouts and expiry |
| `bctl checkin [name]` | Return a checkout early |
| `bctl profiles list` | Show all profiles available to you |
| `bctl profiles sync` | Refresh profile list from Britive |
| `bctl login` | Authenticate (browser SSO or `--token`) |
| `bctl logout` | Clear stored credentials |
| `bctl init` | Interactive tenant + auth setup |
| `bctl doctor` | Diagnose setup issues |
| `bctl config get/set` | Read or write config values |
| `bctl update` | Self-update to the latest release |
| `bctl version` | Print version info |
| `bctl completion [bash\|zsh\|fish]` | Generate shell completions |

### Output formats

```bash
bctl checkout llmg-admin-prod                  # write to ~/.aws/credentials (default)
bctl checkout llmg-admin-prod -o env           # eval-able shell exports
bctl checkout llmg-admin-prod -o process       # AWS credential_process JSON
bctl checkout llmg-admin-prod -o json          # raw JSON to stdout
```

Example: use env output for a one-off shell session:

```bash
eval "$(bctl checkout llmg-admin-prod -o env)"
aws s3 ls
```

Or wire into `~/.aws/config` so the AWS CLI calls bctl automatically whenever credentials are needed:

```ini
[profile llmg-admin-prod]
credential_process = bctl checkout llmg-admin-prod -o process
```

---

## Configuration

Config file: `~/.config/bctl/config.yaml` (migrated automatically from `~/.bctl/` on first run).

```yaml
tenant: acme
default_region: us-east-1
auth:
  method: browser               # browser | token
```

Profile cache lives separately at `~/.cache/bctl/profiles.json` and refreshes every 24 hours or when you run `bctl profiles sync`.

### Environment variables

| Variable | What it does |
|---|---|
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

See [CONTRIBUTING.md](CONTRIBUTING.md). Bug reports and feature requests go in [GitHub Issues](https://github.com/smichalabs/britivectl/issues).

---

## License

[MIT](LICENSE) © smichalabs
