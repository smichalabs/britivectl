# Security Policy

## Reporting a vulnerability

If you discover a security vulnerability in bctl, please report it responsibly. **Do not open a public GitHub issue for security vulnerabilities.**

Email **sajeeve@gmail.com** with:

- A description of the vulnerability
- Steps to reproduce
- The impact you believe it has
- Your bctl version (`bctl version`)

You will receive an acknowledgement within 48 hours. We will work with you to understand the issue and coordinate a fix before any public disclosure.

## Scope

The following are in scope for security reports:

- Command injection or arbitrary code execution via bctl inputs
- Credential leakage (tokens, AWS keys, session data) to unintended locations
- Path traversal in config or cache file operations
- Authentication bypass in the Britive API client
- Dependency vulnerabilities in the Go module tree

The following are out of scope:

- Vulnerabilities in the Britive platform itself (report those to [Britive](https://www.britive.com))
- Social engineering or phishing attacks
- Denial of service against the user's own machine

## For teams evaluating bctl

**TL;DR**: bctl is open source under MIT (`github.com/smichalabs/britivectl`). It only talks to Britive (for credential checkout) and to GitHub (to check for updates). Every release is signed with cosign, has a CycloneDX SBOM and SHA256 checksums attached, and is built by a public GitHub Actions pipeline. Britive tokens are stored in your OS keychain, never in a config file. No telemetry, no daemon, no setuid -- bctl runs as you, with what you can already do.

### What is in place, with receipts

| Concern | What is in place | How to verify |
|---|---|---|
| Can I read the source? | MIT-licensed public repo | <https://github.com/smichalabs/britivectl> |
| Are releases tampered with? | Cosign keyless signatures, SHA256 checksums | `checksums.txt`, `checksums.txt.sig`, `checksums.txt.pem` on every release; verify command in [Verifying releases](#verifying-releases) below |
| What is actually in the binary? | CycloneDX SBOM per platform | `bctl_<platform>.tar.gz.sbom.json` attached to every release |
| Could a malicious commit slip in? | Every PR is gated by gitleaks (secret scan), semgrep (SAST), govulncheck (Go stdlib + dep CVEs), gosec, and checkov (IaC). Blocking, not warn-only. | Public CI logs on every PR |
| Does it phone home? | No telemetry. Outbound calls go only to your Britive tenant and (for the update check) `api.github.com`. Update check is disable-able with `BCTL_NO_UPDATE_CHECK=1`. | `grep -rn http.Client internal/` shows every HTTP client constructor |
| Where are my Britive tokens stored? | macOS Keychain, Windows Credential Manager, libsecret / KWallet on Linux desktop, encrypted file fallback on headless Linux / WSL | `internal/config/keychain.go` -- uses `99designs/keyring` |
| Does it need elevated privileges? | No. Runs as the invoking user. Writes only to `~/.aws/credentials` and `~/.kube/config`, which the user already owns. | `ls -la ~/.aws/credentials ~/.kube/config` after a checkout shows your user as owner |
| What about supply chain (deps)? | `go.mod` and `go.sum` are committed and pinned. `govulncheck` runs on every PR and blocks merges on known CVEs in any transitive dep. | `cat go.mod` and the Security check on any recent PR |
| What about my IdP password? | bctl never sees it. Britive's official browser SSO flow handles authentication; bctl only receives the resulting JWT. | `internal/britive/auth.go` -- the only flow is browser redirect + callback |

### Honest limitations

- **Personal OSS, not vendor-supported.** No SLA, no commercial support contract. If you need that, the official [pybritive](https://github.com/britive/python-cli) is the supported path.
- **Cosign signing is keyless** via sigstore Fulcio. The signature is tied to the GitHub Actions OIDC identity at release time, not a long-lived signing key. That removes key-management risk but means verification commands reference the GitHub identity, not a static public key.
- **Build provenance** uses the standard GitHub Actions OIDC tokens (SLSA Level 2). If your team requires SLSA Level 3 or higher, the pipeline would need `slsa-github-generator`. Not currently wired up.
- **Update checker queries GitHub directly.** If your corp network blocks `api.github.com`, the check silently no-ops -- not a security gap, but worth knowing if you want all artifacts behind your internal Artifactory.

### Common questions

- **"It's a random tool from the internet."** It's an MIT-licensed open source CLI. Source, CI logs, signed releases, and SBOMs are all public and inspectable per release.
- **"What if the author goes rogue?"** Every release is signed and SBOM'd, and every commit is auditable. Blast radius is bounded by the Britive role you check out -- bctl does not grant permissions you do not already have.
- **"How do I know it is not exfiltrating credentials?"** Two outbound calls only: your Britive tenant (required for checkout) and `api.github.com` (update check; disable with `BCTL_NO_UPDATE_CHECK=1`). The HTTP clients are grep-able in a couple hundred lines of code.
- **"Why not just pybritive?"** Both are valid. bctl exists for engineers who want a single static binary, no Python runtime, automatic EKS kubeconfig setup in the same command, and a fuzzy-search picker for daily use. pybritive is Britive's official client.

## Security practices

- Credentials are stored in the OS keychain (macOS Keychain, libsecret on Linux), never in plaintext files
- File writes are atomic (write to temp file, rename)
- The config and cache directories use restrictive permissions (0700 / 0600)
- Every release is signed with [cosign](https://github.com/sigstore/cosign) keyless OIDC and ships with a CycloneDX SBOM
- CI runs gosec, govulncheck, gitleaks, and checkov on every PR
- Dependencies are monitored via GitHub's dependency graph

## Verifying releases

```bash
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  checksums.txt \
  --certificate-identity-regexp 'https://github.com/smichalabs/britivectl' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'
```

## Supported versions

Only the latest release is actively supported with security fixes. Upgrade to the latest version before reporting:

```bash
brew upgrade bctl
```
