# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.0.1-alpha] - 2026-04-09

### Added
- Initial scaffold of `bctl` CLI tool
- `bctl init` — interactive configuration wizard
- `bctl login` — token and browser SSO authentication
- `bctl logout` — remove stored credentials
- `bctl checkout <alias>` — check out a Britive profile for temporary credentials
- `bctl checkin <alias>` — return a checkout early
- `bctl status` — view active checkout sessions
- `bctl profiles list` — list locally configured profiles
- `bctl profiles sync` — sync profiles from Britive API
- `bctl eks connect <alias>` — checkout and update kubeconfig for EKS clusters
- `bctl config get/set` — read and write config values
- `bctl doctor` — environment health checks
- `bctl update` — self-update from GitHub releases
- `bctl completion` — shell completion for bash, zsh, fish, PowerShell
- `bctl version` — print version info (plain and JSON)
- OS keychain integration via go-keyring
- AWS credentials file integration
- Colored, spinner-enhanced terminal output
- Goreleaser configuration for macOS and Linux (amd64 + arm64)
- Homebrew tap support (smichalabs/tap/bctl)

[Unreleased]: https://github.com/smichalabs/britivectl/compare/v0.0.1-alpha...HEAD
[0.0.1-alpha]: https://github.com/smichalabs/britivectl/releases/tag/v0.0.1-alpha
