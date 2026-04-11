# bctl

bctl is a command-line tool for getting just-in-time cloud credentials from [Britive](https://www.britive.com).

It runs as a single binary, fuzzy-searches your entitled profiles, and writes credentials to your local cloud config (e.g. `~/.aws/credentials`) so you can immediately run `aws`, `kubectl`, `terraform`, or anything else that reads them.

## Install

```bash
brew tap smichalabs/tap
brew install bctl
```

For Linux, WSL, or installing from source, see [Install](install.md).

## Use

```bash
bctl
```

bctl opens an interactive picker pre-filled with every profile you have access to:

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

Select a profile and press enter. Credentials are written to `~/.aws/credentials` and you can immediately use them:

```bash
aws s3 ls --profile aws-admin-prod
```

The first run on a new machine prompts you once for your Britive tenant and opens the browser for SSO. Every run after that goes straight to the picker.

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

## License

[MIT](https://github.com/smichalabs/britivectl/blob/main/LICENSE)
