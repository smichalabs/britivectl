# bctl -- Britive CLI

**Get Britive JIT credentials on your laptop in one command.**

```bash
brew install smichalabs/tap/bctl
bctl
```

That's the whole thing. No tenant config, no login wizard, no `profiles sync`, no memorizing Britive paths. Just run `bctl`, pick a profile with your arrow keys, hit **enter**, and your credentials are ready.

```bash
aws s3 ls --profile aws-admin-prod
```

---

## What you actually see

Run `bctl`. An interactive launcher opens with every action it can do, **checkout** already highlighted:

```
┌ bctl -- pick a command (type to filter, enter to run, esc to cancel) ┐
│                                                                      │
│ > checkout    Check out a Britive profile                            │
│   status      Show active profile checkouts                          │
│   checkin     Return a checked-out profile early                     │
│   profiles    Manage Britive profiles                                │
│   eks         EKS cluster operations                                 │
│   login       Authenticate with Britive                              │
│   ...                                                                │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

Hit **enter**. bctl shows you the profile picker:

```
┌ Pick a profile (type to filter, enter to select, esc to cancel) ┐
│                                                                 │
│ > aws-admin-prod        [aws]  AWS/Prod/Admin                   │
│   aws-admin-staging     [aws]  AWS/Staging/Admin                │
│   aws-data-staging      [aws]  AWS/Staging/Data                 │
│   aws-security-staging  [aws]  AWS/Staging/Security             │
│   gcp-admin-sandbox     [gcp]  GCP/Sandbox/Admin                │
│   ...                                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Type a few letters -- `admin prod`, `sandbox`, `data` -- the list narrows instantly. Hit **enter**. Credentials are now in `~/.aws/credentials`. Run whatever `aws` command you wanted. That's the whole flow.

!!! tip "The very first run"
    On a brand-new machine with no config yet, bctl walks you through tenant setup and opens your browser to sign in the first time. It takes 20 seconds. Every run after that skips straight to the profile picker.

---

## Skip the launcher

The launcher is **optional**. Every subcommand can be called directly -- the launcher is just a shortcut for people who don't want to memorize them.

```bash
bctl checkout aws-admin-prod    # check out a specific profile
bctl status                     # show active checkouts
bctl profiles list              # browse everything you can check out
bctl login --token $MY_TOKEN    # authenticate with an API token
```

Partial profile names work too. All of these resolve to `aws-admin-prod`:

```bash
bctl checkout admin-prod
bctl checkout aws-admin
```

If more than one profile matches, bctl shows the picker pre-filtered.

See [the full command list](#all-commands) below.

---

## EKS clusters

Pass `--eks` on a profile that has EKS clusters configured. bctl checks out the credentials and updates your kubeconfig in one step:

```bash
bctl checkout aws-admin-prod --eks
kubectl get pods
```

---

## Supported clouds

| Cloud | Status |
|---|---|
| AWS   | Fully supported -- credentials written to `~/.aws/credentials` |
| GCP   | Browse and resolve profiles today. Credential injection is coming next. |
| Azure | Browse and resolve profiles today. Credential injection is coming next. |

You can see GCP and Azure profiles in `bctl profiles list` and pick them in the launcher. Running `bctl checkout` against one tells you the profile is recognized and points at the roadmap.

---

## Why bctl instead of the Britive web UI or pybritive?

=== "Britive web UI"

    Log in, click apps, click environment, click profile, click checkout, pick a duration, copy three values from a popup, paste them into `~/.aws/credentials` or export them, then run `aws ...`. Credentials expire in an hour. Repeat.

=== "pybritive"

    ```bash
    pip install pybritive[aws]
    pybritive configure tenant -t acme
    pybritive login
    pybritive checkout "AWS/Prod/Admin" -m integrate
    aws s3 ls --profile dev
    ```

    Works, but you memorize and type the full Britive path every time, and you carry a ~100 MB Python stack.

=== "bctl"

    ```bash
    bctl
    ```

    Then arrow keys or fuzzy search. Single 9 MB binary, no runtime, no paths to memorize.

---

## All commands

The launcher is just a convenience. Every one of these can be called directly whenever you prefer.

| Command | What it does |
|---|---|
| `bctl` | Open the command launcher (picks `checkout` by default) |
| `bctl checkout [name]` | Check out a profile (opens the profile picker if no name given) |
| `bctl checkout [name] --eks` | Check out + update kubeconfig for EKS clusters on the profile |
| `bctl checkout [name] --output env` | Print credentials as `export` lines for shell eval |
| `bctl checkout [name] --output process` | Print AWS credential_process JSON |
| `bctl status` | Show active checkouts and their expiry times |
| `bctl checkin [name]` | Return a checked-out profile early |
| `bctl profiles list` | Show every profile you can check out |
| `bctl profiles sync` | Refresh the profile list from Britive |
| `bctl login` | Browser SSO login |
| `bctl login --token <token>` | Authenticate with a Britive API token |
| `bctl logout` | Clear stored credentials |
| `bctl init` | Reconfigure tenant and auth method |
| `bctl doctor` | Diagnose setup issues |
| `bctl config get <key>` | Read a config value |
| `bctl config set <key> <value>` | Write a config value |
| `bctl update` | Self-update to the latest release |
| `bctl version` | Print version info |
| `bctl completion [bash\|zsh\|fish]` | Generate shell completions |

Run `bctl <command> --help` for details on any of them.

---

## Next

- [Install →](install.md)
- [Quick Start →](quickstart.md)
- [Command Reference →](commands/checkout.md)
