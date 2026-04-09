# Contributing

## Setup

```bash
git clone https://github.com/smichalabs/britivectl.git
cd britivectl
make bootstrap    # installs all dev tools and git hooks
make build        # verify everything compiles
```

## Make targets

| Target | Description |
|--------|-------------|
| `make build` | Build `bin/bctl` |
| `make test` | Run tests with race detector (90% coverage threshold) |
| `make lint` | Run golangci-lint v2 |
| `make security` | Run gosec + gitleaks + govulncheck |
| `make tidy` | Run go mod tidy + verify |
| `make clean` | Remove build artifacts |
| `make snapshot` | Build release binaries locally via goreleaser |

## Commit style

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for GCP credentials
fix: correct token expiry calculation
chore: update dependencies
```

## Pull requests

- `make test` and `make lint` must pass
- New commands need a `Long` description and examples in `cmd/`
- New packages need tests

## Reporting bugs

Open an issue at [github.com/smichalabs/britivectl/issues](https://github.com/smichalabs/britivectl/issues).
