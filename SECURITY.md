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
