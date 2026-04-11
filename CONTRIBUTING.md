# Contributing to bctl

Thank you for your interest in contributing to bctl! This document explains how to get involved.

## Getting Started

1. Fork and clone the repo.
2. Install Go 1.25 or later.
3. Bootstrap the dev environment:

   ```bash
   make bootstrap
   ```

   This installs `golangci-lint`, `gosec`, `gitleaks`, `goreleaser`,
   `govulncheck`, and wires up the pre-commit hooks. It is idempotent --
   safe to run multiple times.

4. Verify your setup:

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

Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

The commit type **decides the next version number**, so it matters.
release-please scans commits since the last tag and bumps the version
accordingly.

| Type | When to use | Effect on version |
|---|---|---|
| `feat` | new user-facing functionality | minor bump (in 0.x) |
| `fix` | bug fix | patch bump |
| `perf` | performance improvement | patch bump |
| `sec` | security fix or hardening | patch bump |
| `refactor` | restructuring without behavior change | none |
| `docs` | documentation only | none |
| `chore` | dependency bumps, build config, housekeeping | none |
| `ci` | CI / pipeline changes | none |
| `test` | adding or modifying tests only | none |

Mark a commit as a breaking change with `!` after the type or with a
`BREAKING CHANGE:` line in the body.

**Examples:**

```
feat(checkout): add --force flag to bypass the freshness cache
fix(auth): retry on 502 from Britive
sec(deps): bump x/crypto to fix CVE-2024-XXXXX
chore(ci): pin goreleaser-action to ~> v2
docs: rewrite quickstart around 'just run bctl'
```

For the full release pipeline (how the version flows from your commit
all the way to a published binary), see [docs/release-process.md](docs/release-process.md).

## Adding a New Command

1. Create `cmd/<name>.go` with a `newNameCmd()` function returning `*cobra.Command`.
2. Register it in `cmd/root.go` `init()` function.
3. Add business logic in `internal/` packages (not in `cmd/`).
4. Write tests in the appropriate `_test.go` files.
5. Add a doc page at `docs/commands/<name>.md`.

See [DEVELOPMENT.md](DEVELOPMENT.md) for the full step-by-step guide.

## Pull Request Process

1. Ensure all tests pass and linting is clean (`make test && make lint`).
2. Open a PR against `main` with a clear description and a conventional
   commit message in the title.
3. A maintainer will review and merge.

**You do not edit `CHANGELOG.md` yourself.** release-please reads the
commits since the last tag and generates the changelog automatically
when it opens the next release PR. Anything you write in `CHANGELOG.md`
will be overwritten.

For the full release pipeline see [docs/release-process.md](docs/release-process.md).

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
