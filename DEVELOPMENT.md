# bctl development guide

## Project purpose

`bctl` is a production-quality CLI for [Britive](https://www.britive.com) JIT
access management. It wraps the Britive REST API to make temporary cloud
credential checkout frictionless -- replacing manual web UI workflows and
Python scripts with a single fast binary.

API docs: https://docs.britive.com

---

## Architecture

```
britivectl/
├── main.go                  # Entry point -- calls cmd.Execute()
├── cmd/                     # One file per cobra command
│   ├── root.go              # Root command, persistent flags, command picker
│   ├── checkout.go          # bctl checkout (orchestrator, skip-if-fresh, EKS)
│   ├── checkin.go           # bctl checkin
│   ├── status.go            # bctl status
│   ├── profiles.go          # bctl profiles list|sync
│   ├── eks.go               # bctl eks connect
│   ├── issue.go             # bctl issue bug|feature (browser-based filing)
│   ├── config.go            # bctl config get|set|view
│   ├── doctor.go            # bctl doctor
│   ├── login.go             # bctl login
│   ├── logout.go            # bctl logout
│   ├── init.go              # bctl init
│   ├── update.go            # bctl update
│   ├── version.go           # bctl version
│   ├── completion.go        # bctl completion [bash|zsh|fish]
│   └── state_callbacks.go   # Callbacks for the EnsureReady orchestrator
├── internal/
│   ├── britive/             # Britive REST API client
│   │   ├── client.go        # HTTP client, auth headers, error parsing
│   │   ├── auth.go          # Token + browser SSO authentication
│   │   ├── profiles.go      # Profile listing (GET /api/access)
│   │   ├── checkout.go      # Checkout/checkin/status API calls
│   │   └── errors.go        # Sentinel error types
│   ├── aws/
│   │   ├── credentials.go   # ~/.aws/credentials file writer (atomic)
│   │   └── eks.go           # aws eks update-kubeconfig wrapper
│   ├── config/
│   │   ├── config.go        # Config struct, Load/Save (atomic YAML write)
│   │   ├── paths.go         # XDG paths, legacy ~/.bctl migration
│   │   ├── cache.go         # Profile catalog cache (~/.cache/bctl/profiles.json)
│   │   ├── checkouts.go     # Per-profile checkout state (skip-if-fresh cache)
│   │   └── keychain.go      # OS keychain via go-keyring (token, expiry, type)
│   ├── issues/
│   │   └── issues.go        # GitHub issue URL builder, environment block
│   ├── output/
│   │   ├── output.go        # Colored output helpers (Success, Error, Warning, Info)
│   │   ├── table.go         # tablewriter wrapper
│   │   └── spinner.go       # briandowns/spinner wrapper
│   ├── resolver/
│   │   ├── resolver.go      # Fuzzy profile matching (exact, substring, subsequence)
│   │   ├── tui.go           # Bubbletea profile picker
│   │   └── command_picker.go # Bubbletea command picker (bctl with no args)
│   ├── state/
│   │   └── state.go         # EnsureReady orchestrator (init, login, sync)
│   ├── system/
│   │   └── browser.go       # Cross-platform browser launcher
│   └── update/
│       └── update.go        # Self-update via GitHub releases
└── pkg/version/
    └── version.go           # Version/Commit/BuildDate injected via ldflags
```

**Key design decisions:**

- All HTTP calls have a 30-second timeout.
- Credentials are stored in the OS keychain (`go-keyring`), never in plaintext.
- File writes are atomic (write to temp, rename).
- `BCTL_NO_COLOR=1` or `NO_COLOR=1` disables color output.
- The EnsureReady orchestrator in `internal/state` handles first-run setup
  (init, login, profile sync) so individual commands do not duplicate that logic.
- The skip-if-fresh cache in `internal/config/checkouts.go` avoids redundant
  Britive API calls when credentials are still valid.
- Testable logic belongs in `internal/` packages, not in `cmd/`. Adding test
  files to `cmd/` pulls the entire package into coverage measurement and
  drops the total below threshold.

---

## How to add a new command

1. Create `cmd/<name>.go` with a `newXxxCmd() *cobra.Command` function.
2. Build a `cobra.Command` with `Use`, `Short`, `Long`, and `Example` fields.
   Wire `RunE` to a `runXxx()` helper.
3. Register it in `cmd/root.go`'s `init()`:
   ```go
   rootCmd.AddCommand(newXxxCmd())
   ```
4. Add the command name to the `ordered` slice in `commandChoices()` if it
   should appear in the interactive command picker.
5. If the command needs API access:
   ```go
   cfg, err := config.Load()
   token, err := requireToken(ctx, cfg.Tenant)
   client := newAPIClient(cfg.Tenant, token)
   ```
6. Add a doc page at `docs/docs/commands/<name>.md` and wire it into
   `mkdocs.yml` under the Commands nav section.
7. Write tests in the appropriate `internal/` package, not in `cmd/`.

---

## How to add a new cloud provider

1. Add a new package under `internal/<provider>/` mirroring `internal/aws/`.
2. Implement at minimum:
   - `WriteCredentials(profile string, creds ProviderCredentials) error`
3. Update `internal/britive/checkout.go` -- the `Credentials` struct may need
   provider-specific fields.
4. Update `cmd/checkout.go` -- add a `case "<provider>"` in the credential
   injection switch. Update `printComingSoon` to remove the provider from the
   "coming soon" message.
5. Update `internal/config/config.go` -- `Profile.Cloud` already accepts any
   string; add the new cloud name to the `bctl init` wizard and docs.
6. Add docs in `docs/docs/commands/checkout.md` and the EKS guide (if the
   provider has a cluster service like GKE or AKS).

---

## Testing patterns

Tests live alongside the code they test (`_test.go` files in the same package).

- **Table-driven**: Use `[]struct{ name, input, expected }` slices.
- **Temp dirs**: Use `t.TempDir()` for any test that writes files.
- **No external dependencies**: Tests must not hit the real Britive API or the
  OS keychain. Inject interfaces or use `httptest.NewServer` to mock HTTP.
- **Coverage target**: CI enforces 75% overall. macOS keychain tests are gated
  with `skipIfNotDarwin` so Linux CI runners do not fail on them.
- **Keep testable logic in `internal/`**: Adding test files to `cmd/` pulls the
  entire cmd package into coverage measurement. Extract helpers into `internal/`
  packages instead.

```bash
make test              # go test with race + coverage threshold
go test ./... -v -race # direct invocation without coverage gate
```

---

## Development commands

```bash
make build       # build ./bin/bctl
make install     # build + cp to /usr/local/bin/bctl
make test        # go test with race detector + coverage (75% threshold)
make lint        # golangci-lint run
make completions # generate shell completions into ./completions/
make snapshot    # goreleaser snapshot build (no publish)
make clean       # rm bin/ dist/ coverage.out
```
