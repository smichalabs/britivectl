# Sessions and credential caching

bctl is built around the idea that **you should sign in once a day at most**, and **repeat checkouts of the same profile should be instant**. Two independent mechanisms make that happen.

You don't have to do anything to opt in -- both are on by default. This page exists so the behavior is not a black box when you wonder why bctl just opened your browser, or why a checkout returned in 50 milliseconds instead of calling Britive.

---

## 1. Britive session token (auto-refresh)

When you run `bctl login` (or the very first `bctl` on a new machine), bctl opens your browser, you authenticate to Britive via SSO, and Britive hands back a JWT. bctl stores three things in your OS keychain:

- the JWT itself
- the token type (`Bearer` for SSO, `TOKEN` for an API token)
- the expiry timestamp, decoded from the JWT's `exp` claim

Every command that talks to Britive (`checkout`, `checkin`, `status`, `profiles sync`, `eks`) starts by calling an internal `requireToken` helper. That helper:

1. Reads the token from the keychain.
2. If it's a Bearer (SSO) token, compares the stored expiry to the current time.
3. If the token is **still valid**, returns it immediately. The command continues.
4. If the token is **expired**, prints `Session expired -- re-authenticating...`, re-opens your browser for SSO, stores the fresh JWT and new expiry, and continues with the new token.

The result: **you do not run `bctl login` manually after the first time**. You sign in once, you keep using bctl all day, and when the JWT eventually expires the next command silently re-prompts your browser. There is no "your session has expired, please run bctl login" wall.

!!! info "API tokens don't auto-refresh"
    If you authenticated with `bctl login --token <token>` (the static API token form, typically used in CI), bctl does **not** try to refresh it. API tokens have no JWT `exp` claim, so bctl trusts them until Britive returns an auth error. Rotate them on your own schedule.

### What you'll see

```text
$ bctl checkout aws-admin-prod
Session expired -- re-authenticating...
   (browser opens, you click through SSO)
Checked out aws-admin-prod (expires in 4h)
```

The "session expired" line only appears at the boundary -- once per JWT lifetime. Every other invocation skips it entirely.

---

## 2. Profile credential cache (skip-if-fresh)

The Britive session token is one thing. The **temporary cloud credentials** Britive hands you when you check out a profile are another. Those credentials live their full duration on Britive's side -- typically 4 hours for AWS -- and they're already written to `~/.aws/credentials`. There is no reason to ask Britive for them again before they expire.

So bctl doesn't.

### How it works

After every successful `bctl checkout`, bctl writes a small JSON file:

```text
~/.cache/bctl/checkouts/<alias>.json
```

containing:

```json
{
  "alias": "aws-admin-prod",
  "transactionId": "txn-abc123",
  "checkedOutAt": "2026-04-11T01:30:00Z",
  "expiresAt": "2026-04-11T05:30:00Z"
}
```

The next time you run `bctl checkout aws-admin-prod`, **before** calling Britive, bctl reads that file and checks: do these credentials have at least 5 minutes of life left?

- **Yes** -> bctl prints `aws-admin-prod is already checked out (expires in 3h 47m)` and returns. **Zero Britive API calls.** The credentials are already in `~/.aws/credentials`. Your `aws s3 ls` works immediately.
- **No** (file missing, expired, or within the 5-minute buffer) -> bctl does the full Britive checkout and writes a fresh state file.

The 5-minute buffer exists so that downstream tools (kubectl, aws-cli, terraform) don't get handed credentials that are about to expire mid-operation.

### Force a refresh

If you actually want to talk to Britive -- for example, to get a fresh transaction ID, or to test your network path -- pass `--force` (or `-f`):

```bash
bctl checkout aws-admin-prod --force
```

This bypasses the freshness check and always calls the Britive API.

### When the cache is bypassed automatically

The skip-if-fresh path **only applies** to the default `awscreds` output format (writing to `~/.aws/credentials`). For the other output modes -- where the user clearly wants the actual credential values printed to stdout or piped to another tool -- bctl always calls Britive:

| Output mode | Cache used? |
|---|---|
| `awscreds` (default for AWS profiles) | yes |
| `env` (eval-able shell exports) | no |
| `process` (AWS credential_process JSON) | no |
| `json` (raw JSON to stdout) | no |

This is why `eval "$(bctl checkout aws-admin-prod -o env)"` always feels slightly slower than the bare `bctl checkout aws-admin-prod`.

### Releasing a checkout

`bctl checkin <alias>` returns the checkout to Britive **and** removes the local state file. The next checkout will be a full Britive call, as expected.

`bctl logout` removes the session JWT but **does not** wipe the per-profile cache files. If you want to start completely clean:

```bash
bctl logout
rm -rf ~/.cache/bctl/checkouts
```

---

## What `bctl status` shows

`bctl status` reads the cache files (and Britive's `app-access-status` endpoint) and prints what is currently checked out, with how much life is left on each:

```text
ALIAS                  CLOUD  EXPIRES IN
aws-admin-prod         aws    3h 47m
gcp-admin-sandbox      gcp    1h 02m
```

This is the source of truth if you're trying to remember whether you already have credentials for a profile.

---

## Where things live

| Thing | Where | What removes it |
|---|---|---|
| Session JWT + expiry | OS keychain (macOS Keychain, libsecret on Linux) | `bctl logout` |
| Per-profile checkout state | `~/.cache/bctl/checkouts/<alias>.json` | `bctl checkin <alias>` |
| AWS credentials (the actual ones) | `~/.aws/credentials` | overwritten on next checkout, or `bctl checkin` |
| Profile catalog cache | `~/.cache/bctl/profiles.json` | `bctl profiles sync` (24h auto-refresh) |
| bctl config | `~/.config/bctl/config.yaml` | `bctl init` rewrites it |

Nothing about bctl writes to `~/.bctl/` (the early-version path). If you have one left over from an old install, bctl migrated it on first run and you can delete it.
