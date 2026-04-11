# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0](https://github.com/smichalabs/britivectl/compare/v0.1.1...v0.2.0) (2026-04-11)


### Features

* zero-touch checkout orchestrator with fuzzy TUI picker ([#22](https://github.com/smichalabs/britivectl/issues/22)) ([ecfcaed](https://github.com/smichalabs/britivectl/commit/ecfcaed488fe62e672676fc90fd825734396c241))


### Bug Fixes

* **ci:** use minor bumps for feat commits in 0.x ([#24](https://github.com/smichalabs/britivectl/issues/24)) ([9cbef3f](https://github.com/smichalabs/britivectl/commit/9cbef3f9d1afb3e08beb8ab75d5cec0eed22b4f4))

## [Unreleased]

## [0.1.1] - 2026-04-10

### Added
- Documentation link in `bctl --help` output

### Fixed
- Release artifacts now publish to the public `britivectl-releases` repo so `brew install smichalabs/tap/bctl` works without authentication

### Project changes (non-binary)
- Documentation site at [smichalabs.dev/utils/bctl](https://smichalabs.dev/utils/bctl/) built with MkDocs Material
- Side-by-side comparison of manual web UI, pybritive, and bctl workflows for AWS and EKS
- Cloud support roadmap section noting upcoming GCP and Azure support
- CODEOWNERS file requiring owner review on all changes
- Migrated infra Terraform CI from static IAM user keys to GitHub OIDC role assumption
- Replaced Namecheap apex CNAME with Route 53 hosted zone and ALIAS record for reliable DNS
- Hardened S3 bucket with SSE-S3 encryption and deny-insecure-transport policy
- Added CloudFront response headers policy (HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy)
- Added CloudWatch alarms with SNS email notifications for CloudFront 5xx and 4xx error rates
- Added checkov Terraform IaC scanning to bootstrap, `make security`, pre-commit hooks, and CI
- Added conventional commit-msg hook enforcing the conventional commit format
- Removed CodeQL workflow (private repo without GitHub Advanced Security; coverage provided by gosec, govulncheck, gitleaks, and checkov)

## [0.1.0] - 2026-04-09

### Added
- Initial release of `bctl` CLI tool
- `bctl init` -- interactive configuration wizard
- `bctl login` -- token and browser SSO authentication
- `bctl logout` -- remove stored credentials
- `bctl checkout <alias>` -- check out a Britive profile for temporary credentials
- `bctl checkin <alias>` -- return a checkout early
- `bctl status` -- view active checkout sessions
- `bctl profiles list` -- list locally configured profiles
- `bctl profiles sync` -- sync profiles from Britive API
- `bctl eks connect <alias>` -- checkout and update kubeconfig for EKS clusters
- `bctl config get/set` -- read and write config values
- `bctl doctor` -- environment health checks
- `bctl update` -- self-update from GitHub releases
- `bctl completion` -- shell completion for bash, zsh, fish, PowerShell
- `bctl version` -- print version info (plain and JSON)
- OS keychain integration via go-keyring
- AWS credentials file integration
- Colored, spinner-enhanced terminal output
- Goreleaser configuration for macOS and Linux (amd64 + arm64)
- Homebrew tap support (smichalabs/tap/bctl)

[Unreleased]: https://github.com/smichalabs/britivectl/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/smichalabs/britivectl/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/smichalabs/britivectl-releases/releases/tag/v0.1.0
