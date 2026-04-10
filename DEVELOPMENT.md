# bctl development guide

## Project purpose

`bctl` is a production-quality CLI for [Britive](https://www.britive.com) JIT
access management. It wraps the Britive REST API to make temporary cloud
credential checkout frictionless — replacing manual web UI workflows and fragile
Python scripts with a single fast binary.

Supports AWS, GCP, Azure, and any cloud provider Britive supports.

API docs: https://docs.britive.com

---

## Architecture

```
britivectl/
├── main.go                  # Entry point — calls cmd.Execute()
├── cmd/                     # One file per cobra command
│   ├── root.go              # Root command, persistent flags, subcommand wiring
│   ├── checkout.go          # bctl checkout
│   ├── checkin.go           # bctl checkin
│   ├── status.go            # bctl status
│   ├── profiles.go          # bctl profiles list|sync
│   ├── eks.go               # bctl eks connect
│   ├── config.go            # bctl config get|set
│   ├── doctor.go            # bctl doctor
│   ├── login.go             # bctl login
│   ├── logout.go            # bctl logout
│   ├── init.go              # bctl init
│   ├── update.go            # bctl update
│   ├── version.go           # bctl version
│   └── completion.go        # bctl completion [bash|zsh|fish]
├── internal/
│   ├── britive/             # Britive REST API client
│   │   ├── client.go        # HTTP client, auth headers, error parsing
│   │   ├── auth.go          # Token + browser SSO authentication
│   │   ├── profiles.go      # Profile listing
│   │   └── checkout.go      # Checkout/checkin/my-sessions
│   ├── aws/
│   │   ├── credentials.go   # ~/.aws/credentials file writer (atomic)
│   │   └── eks.go           # aws eks update-kubeconfig wrapper
│   ├── config/
│   │   ├── config.go        # Config struct, Load/Save (atomic YAML write)
│   │   └── keychain.go      # OS keychain via go-keyring
│   ├── output/
│   │   ├── output.go        # Colored output helpers, JSON/env/process printers
│   │   ├── table.go         # tablewriter wrapper
│   │   └── spinner.go       # briandowns/spinner wrapper
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

---

## How to add a new command

1. Create `cmd/<name>.go` with a `newXxxCmd() *cobra.Command` function.
2. In the function, build a `cobra.Command` with `Use`, `Short`, `Long`, and
   `Example` fields filled in. Wire `RunE` to a `runXxx()` helper.
3. Register it in `cmd/root.go`'s `init()`:
   ```go
   rootCmd.AddCommand(newXxxCmd())
   ```
4. If the command needs API access, load config and token in `runXxx()`:
   ```go
   cfg, err := config.Load()
   token, err := config.GetToken(cfg.Tenant)
   client := britive.NewClient(cfg.Tenant, token)
   ```
5. Add a doc file at `docs/commands/<name>.md`.
6. Write tests in `cmd/<name>_test.go` if the command has non-trivial logic.

---

## How to add a new cloud provider

1. Add a new package under `internal/<provider>/` mirroring `internal/aws/`.
2. Implement at minimum:
   - `WriteCredentials(profile string, creds ProviderCredentials) error`
3. Update `internal/britive/checkout.go` — the `Credentials` struct may need
   provider-specific fields. Add them as needed.
4. Update `cmd/checkout.go` — add a `case "<provider>"` in the output format
   switch.
5. Update `internal/config/config.go` — `Profile.Cloud` already accepts any
   string; add the new cloud name to the `bctl init` wizard and docs.
6. Add docs in `docs/commands/checkout.md` and the README.

---

## Testing patterns

Tests live alongside the code they test (`_test.go` files in the same package).

- **Table-driven**: Use `[]struct{ name, input, expected }` slices.
- **Temp dirs**: Use `t.TempDir()` for any test that writes files.
- **No external dependencies**: Tests must not hit the real Britive API or the
  OS keychain. Inject interfaces or use `httptest.NewServer` to mock HTTP.
- **Coverage target**: Aim for ≥70% per package.

Run tests:

```bash
make test
# or
go test ./... -v -race
```

---

## Development commands

```bash
make build       # build ./bin/bctl
make install     # build + cp to /usr/local/bin/bctl
make test        # go test ./... with race + coverage
make lint        # golangci-lint run
make completions # generate shell completions into ./completions/
make snapshot    # goreleaser snapshot build (no publish)
make clean       # rm bin/ dist/ coverage.out
```
