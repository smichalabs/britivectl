# bctl

[![CI](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml/badge.svg)](https://github.com/smichalabs/britivectl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/smichalabs/britivectl)](https://goreportcard.com/report/github.com/smichalabs/britivectl)

A command-line tool for getting just-in-time cloud credentials from [Britive](https://www.britive.com).

## Get started in 60 seconds

```bash
brew tap smichalabs/tap
brew install bctl
bctl
```

That's the whole flow. Pick a profile from the list, hit enter, your AWS credentials are now in `~/.aws/credentials`. You can immediately:

```bash
aws s3 ls --profile aws-admin-prod
```

The first time you run bctl on a new machine it asks for your Britive tenant and opens your browser for SSO. Twenty seconds, one time.

> Linux, WSL, or build from source: see [Install](https://smichalabs.dev/utils/bctl/install/).

**Full documentation: [smichalabs.dev/utils/bctl](https://smichalabs.dev/utils/bctl/)**

## What it does

bctl wraps the Britive REST API to make temporary cloud credential checkout frictionless. Single binary, interactive profile picker, automatic browser re-auth when your Britive session expires, and credential caching so repeat checkouts are instant.

## Install (all platforms)

**macOS**

```bash
brew tap smichalabs/tap
brew install bctl
```

**Linux / WSL**

```bash
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

**Build from source** (requires Go 1.25+)

```bash
git clone https://github.com/smichalabs/britivectl.git
cd britivectl
make install
```

## Use

```bash
bctl
```

bctl opens a command picker. Select **checkout** (highlighted by default), then pick a profile from the fuzzy-searchable list. Credentials are written to `~/.aws/credentials`.

```bash
aws s3 ls --profile aws-admin-prod
```

Skip the pickers if you already know which profile you want:

```bash
bctl checkout aws-admin-prod    # skips both pickers, checks out immediately
bctl checkout admin-prod        # substring match works too
```

## Supported clouds

| Cloud | Status |
|---|---|
| AWS   | Fully supported. Credentials written to `~/.aws/credentials`. |
| GCP   | Profiles browsable. Credential injection on the roadmap. |
| Azure | Profiles browsable. Credential injection on the roadmap. |

## Features

- **Interactive pickers** -- command picker and fuzzy-searchable profile picker via bubbletea TUI
- **Automatic browser re-auth** -- when the Britive session JWT expires, the next bctl command opens your browser for SSO automatically. You do not run `bctl login` separately. The browser flow itself is the same as a normal sign-in (one click if your IdP session is still alive, full SSO if not).
- **Credential caching** -- repeat checkouts of the same profile skip the Britive API entirely if the credentials still have life. Pass `--force` to override.
- **EKS in one step** -- `bctl checkout <profile> --eks` checks out credentials and updates kubeconfig for every cluster on the profile
- **Output formats** -- `awscreds` (default), `env`, `process` (AWS credential_process), `json`
- **In-CLI issue filing** -- `bctl issue bug` and `bctl issue feature` open pre-filled GitHub issues in your browser
- **Supply chain security** -- every release ships with CycloneDX SBOMs and cosign keyless signatures

## Configuration

Config file: `~/.config/bctl/config.yaml`

| Variable | Description |
|---|---|
| `BCTL_TENANT` | Override tenant from config |
| `BCTL_TOKEN` | Use this API token (skips keychain) |
| `BCTL_OUTPUT` | Default output format |
| `BCTL_REGION` | Default AWS region |
| `BCTL_NO_COLOR` | Disable color output |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, commit conventions, and the PR process. The full release pipeline is documented in [docs/release-process.md](docs/release-process.md).

## Security

To report a vulnerability, see [SECURITY.md](SECURITY.md).

## Issues

Bug reports and feature requests: [GitHub Issues](https://github.com/smichalabs/britivectl/issues)

Or use the built-in CLI:

```bash
bctl issue bug
bctl issue feature
```

## License

[MIT](LICENSE)
