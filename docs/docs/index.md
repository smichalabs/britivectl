# bctl

bctl is a command-line tool for getting just-in-time cloud credentials from [Britive](https://www.britive.com).

It runs as a single binary, fuzzy-searches your entitled profiles, and writes credentials to your local cloud config (e.g. `~/.aws/credentials`) so you can immediately run `aws`, `kubectl`, `terraform`, or anything else that reads them.

## Install

**macOS**

```bash
brew tap smichalabs/tap
brew install bctl
```

**Linux / WSL**

```bash
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

Auto-detects your distro and installs the matching `.deb`, `.rpm`, or tarball. On WSL, install `wslu` first (`sudo apt install wslu`) so browser-based SSO works.

**Build from source** (requires Go 1.25+)

```bash
git clone https://github.com/smichalabs/britivectl.git
cd britivectl
make install
```

Shell completions and platform-specific notes are in the [Install reference](install.md).

## Use

```bash
bctl
```

bctl opens a command picker with every action it can do. **checkout** is already highlighted:

```
┌ bctl -- pick a command (type to filter, enter to run, esc to cancel) ┐
│                                                                      │
│ > checkout    Check out a Britive profile                            │
│   status      Show active profile checkouts                          │
│   checkin     Return a checked-out profile early                     │
│   profiles    Manage Britive profiles                                │
│   eks         EKS cluster operations                                 │
│   ...                                                                │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

Press **enter**. The profile picker opens with every profile you have access to:

```
┌ Pick a profile (type to filter, enter to select, esc to cancel) ┐
│                                                                 │
│ > aws-admin-prod        [aws]  AWS/Prod/Admin                   │
│   aws-admin-staging     [aws]  AWS/Staging/Admin                │
│   aws-data-staging      [aws]  AWS/Staging/Data                 │
│   aws-security-staging  [aws]  AWS/Staging/Security             │
│   gcp-admin-sandbox     [gcp]  GCP/Sandbox/Admin                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Select a profile and press **enter**. Credentials are written to `~/.aws/credentials` and you can immediately use them:

```bash
aws s3 ls --profile aws-admin-prod
```

You can also skip either or both pickers by passing arguments directly:

```bash
bctl checkout                   # skips the command picker, opens the profile picker
bctl checkout aws-admin-prod    # skips both pickers, checks out immediately
```

The first run on a new machine prompts you once for your Britive tenant and opens the browser for SSO. Every run after that goes straight to the pickers.

## Supported clouds

| Cloud | Status |
|---|---|
| AWS   | Fully supported. Credentials written to `~/.aws/credentials`. |
| GCP   | Profiles browsable today. Credential injection on the roadmap. |
| Azure | Profiles browsable today. Credential injection on the roadmap. |

## Documentation

- [Quick Start](quickstart.md) -- first-time setup and the common workflows
- [Sessions & caching](sessions.md) -- how bctl handles session refresh and credential caching
- [Configuration](configuration.md) -- config file and environment variables
- [Commands](commands/checkout.md) -- full reference for every subcommand
- [EKS Guide](eks.md) -- using bctl with Amazon EKS
- [Comparison](comparison.md) -- bctl vs the Britive web UI vs pybritive
- [Feedback & issues](feedback.md) -- how to file a bug or feature request with `bctl issue`

## License

[MIT](https://github.com/smichalabs/britivectl/blob/main/LICENSE)
