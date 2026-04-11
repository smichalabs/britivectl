# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0](https://github.com/smichalabs/britivectl/compare/v0.3.1...v0.4.0) (2026-04-11)


### Features

* **eks:** require AWS profile for any EKS kubeconfig flow ([#41](https://github.com/smichalabs/britivectl/issues/41)) ([3b3762c](https://github.com/smichalabs/britivectl/commit/3b3762c2d07dad23721dc575c135533f0ae979ed))

## [0.3.1](https://github.com/smichalabs/britivectl/compare/v0.3.1...v0.3.1) (2026-04-11)


### Features

* add AWS credentials writer and EKS kubeconfig updater ([b21555c](https://github.com/smichalabs/britivectl/commit/b21555cb68281bf5ed40f04f43de1b82e851ead8))
* add Britive API client with auth and JIT session management ([0be23e8](https://github.com/smichalabs/britivectl/commit/0be23e885a335c20e798ac239f85f417c1aa11be))
* add cobra CLI with all 13 commands ([900c10f](https://github.com/smichalabs/britivectl/commit/900c10f760b6e8d4fa8edd70fe5eba8e7a287c5f))
* add config package with YAML persistence and OS keychain ([ec3789c](https://github.com/smichalabs/britivectl/commit/ec3789c139b2018b9f697621f06e58e3c67b7026))
* add output package — color, table, spinner, JSON/env/process ([921bd9e](https://github.com/smichalabs/britivectl/commit/921bd9e1cf586549e7682cdc42b1b035864a990e))
* add Route 53 hosted zone for reliable apex DNS ([#13](https://github.com/smichalabs/britivectl/issues/13)) ([2b3c2dd](https://github.com/smichalabs/britivectl/commit/2b3c2dd4d20a752faead886559f97a5173f65f93))
* add self-update via GitHub releases with checksum verification ([dfa7c45](https://github.com/smichalabs/britivectl/commit/dfa7c45051affc905cb4c5cf4464ab701f6da333))
* add version package with ldflags build injection ([059699d](https://github.com/smichalabs/britivectl/commit/059699dadd1cb6c14d0db894be531f389aff9ca1))
* bctl foundation ([#1](https://github.com/smichalabs/britivectl/issues/1)) ([437bb0b](https://github.com/smichalabs/britivectl/commit/437bb0b412279e0458eeb488f1ec634dbb29fdce))
* skip Britive API on bctl checkout when credentials are still fresh ([#28](https://github.com/smichalabs/britivectl/issues/28)) ([a673198](https://github.com/smichalabs/britivectl/commit/a673198c486d8913474238e9be36c622d98d3f86))
* zero-touch checkout orchestrator with fuzzy TUI picker ([#22](https://github.com/smichalabs/britivectl/issues/22)) ([ecfcaed](https://github.com/smichalabs/britivectl/commit/ecfcaed488fe62e672676fc90fd825734396c241))


### Bug Fixes

* broaden S3 and CloudFront permissions for terraform-cli IAM policy ([#5](https://github.com/smichalabs/britivectl/issues/5)) ([d344497](https://github.com/smichalabs/britivectl/commit/d3444972e413390ae140d2e9ef32f145c43b30e2))
* **ci:** use minor bumps for feat commits in 0.x ([#24](https://github.com/smichalabs/britivectl/issues/24)) ([9cbef3f](https://github.com/smichalabs/britivectl/commit/9cbef3f9d1afb3e08beb8ab75d5cec0eed22b4f4))
* friendly non-AWS message, auto-filter TUI, command picker on no args ([#26](https://github.com/smichalabs/britivectl/issues/26)) ([0b448dd](https://github.com/smichalabs/britivectl/commit/0b448dd21c6a95770dc82fc6711649088ea19072))
* skip access key creation if one exists, add CF function permissions ([378aa99](https://github.com/smichalabs/britivectl/commit/378aa99fafceec77e5732b9c2542ead6f5904a19))


### Security

* add checkov, commit-msg hook, remove default root object ([#7](https://github.com/smichalabs/britivectl/issues/7)) ([3cf034c](https://github.com/smichalabs/britivectl/commit/3cf034cc7b8a99f5ff262d17da38e258c4105ab5))
* harden S3 bucket, add security headers, and CloudWatch alerting ([#10](https://github.com/smichalabs/britivectl/issues/10)) ([df10398](https://github.com/smichalabs/britivectl/commit/df1039801f98089883c2f4a34ffd1252ce659e9a))
* migrate Terraform CI from static keys to OIDC ([#11](https://github.com/smichalabs/britivectl/issues/11)) ([9a26df7](https://github.com/smichalabs/britivectl/commit/9a26df722f75c3885332f5548b20a9b795141041))


### CI

* add workflow_dispatch to release-please and force 0.3.1 ([4bf8d0a](https://github.com/smichalabs/britivectl/commit/4bf8d0ad881046ad6f1ec3771d080f2c3d6964ac))

## [0.3.1](https://github.com/smichalabs/britivectl/compare/v0.3.0...v0.3.1) (2026-04-11)


### CI

* add workflow_dispatch to release-please and force 0.3.1 ([4bf8d0a](https://github.com/smichalabs/britivectl/commit/4bf8d0ad881046ad6f1ec3771d080f2c3d6964ac))

## [0.3.0](https://github.com/smichalabs/britivectl/compare/v0.2.0...v0.3.0) (2026-04-11)


### Features

* skip Britive API on bctl checkout when credentials are still fresh ([#28](https://github.com/smichalabs/britivectl/issues/28)) ([a673198](https://github.com/smichalabs/britivectl/commit/a673198c486d8913474238e9be36c622d98d3f86))


### Bug Fixes

* friendly non-AWS message, auto-filter TUI, command picker on no args ([#26](https://github.com/smichalabs/britivectl/issues/26)) ([0b448dd](https://github.com/smichalabs/britivectl/commit/0b448dd21c6a95770dc82fc6711649088ea19072))

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
