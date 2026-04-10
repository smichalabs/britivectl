# Contributing to bctl

Thank you for your interest in contributing to bctl! This document explains how to get involved.

## Getting Started

1. Fork the repository and clone your fork.
2. Install Go 1.25 or later.
3. Install development tools:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
brew install goreleaser pre-commit
pre-commit install
```

4. Run the tests to verify your setup:

```bash
make test
```

## Development Workflow

### Making changes

1. Create a branch from `main`:
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. Make your changes, writing tests for new functionality.

3. Run the full test suite and linter:
   ```bash
   make test
   make lint
   ```

4. Build and smoke-test locally:
   ```bash
   make build
   ./bin/bctl --help
   ./bin/bctl version
   ```

5. Commit your changes following the [Conventional Commits](https://www.conventionalcommits.org/) specification.

## Conventional Commits

Commit messages must follow this format:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat` — new feature
- `fix` — bug fix
- `docs` — documentation only
- `refactor` — code change that neither fixes a bug nor adds a feature
- `test` — adding or modifying tests
- `chore` — maintenance tasks (deps, CI, etc.)
- `perf` — performance improvement

Examples:
```
feat(checkout): add --eks flag to auto-update kubeconfig
fix(auth): handle token expiry gracefully
docs(readme): add EKS connect example
```

## Adding a New Command

1. Create `cmd/<name>.go` with a `newNameCmd()` function returning `*cobra.Command`.
2. Register it in `cmd/root.go` `init()` function.
3. Add business logic in `internal/` packages (not in `cmd/`).
4. Write tests in the appropriate `_test.go` files.
5. Add a doc page at `docs/commands/<name>.md`.

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full step-by-step guide.

## Pull Request Process

1. Ensure all tests pass and linting is clean.
2. Update `CHANGELOG.md` under `[Unreleased]`.
3. Open a PR against `main` with a clear description.
4. A maintainer will review and merge.

## Test Requirements

- All new packages must have test files.
- Aim for table-driven tests.
- Tests must not make real network calls; use `httptest` for HTTP interactions.
- Run `make test` before pushing.

## Code Style

- Follow standard Go conventions (`gofmt`, `goimports`).
- Use `internal/output` for all user-facing messages.
- Return errors rather than calling `os.Exit` inside library functions.
- Document all exported symbols.
