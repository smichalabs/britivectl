# bctl: Britive CLI

**bctl** is a fast, polished CLI for [Britive](https://www.britive.com) JIT access management.
Replace manual web UI workflows and fragile scripts with a single binary.

```bash
bctl checkout dev        # get AWS credentials in seconds
bctl eks connect dev     # checkout + update kubeconfig
bctl status              # see what's checked out and when it expires
```

---

## Prerequisites

- An active [Britive](https://www.britive.com) tenant with JIT access profiles configured
- A valid Britive user account with permissions to check out profiles
- bctl uses the publicly available [Britive REST API](https://docs.britive.com/apidocs) -- no special entitlements required

---

## Install

=== "macOS"

    ```bash
    brew tap smichalabs/tap
    brew install smichalabs/tap/bctl
    ```

=== "Linux / WSL"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
    ```

    Auto-detects your distro: `.deb`, `.rpm`, or tarball.


---

## AWS credentials

=== "Web UI (manual)"

    1. Log into the Britive web portal
    2. Navigate apps -> environment -> profile
    3. Click checkout, pick a duration
    4. Copy three values from the popup: access key ID, secret access key, session token
    5. Paste into `~/.aws/credentials` under a profile name (or `export` in your shell)
    6. Run `aws ...`
    7. Credentials expire (typically 1 hour) -- repeat from step 1

=== "pybritive"

    ```bash
    pip install pybritive[aws]
    pybritive configure tenant -t acme
    pybritive login
    pybritive checkout "AWS/Sandbox/Developer" -m integrate
    aws s3 ls --profile dev
    ```

    Works, but you type the full Britive path every time. AWS integration requires the `-m integrate` flag and the Python install brings ~100 MB of dependencies.

=== "bctl"

    ```bash
    brew install smichalabs/tap/bctl
    bctl checkout dev
    aws s3 ls --profile dev
    ```

    That's it. On first run `bctl checkout` walks you through tenant setup,
    opens the browser for SSO, fetches your profiles from the Britive API,
    and writes credentials to `~/.aws/credentials` -- all in one command.
    Subsequent runs skip every step that's already done.

    No full Britive path to memorize: use short aliases (`dev`) or
    substring matches (`sandbox`). Single static binary, no Python runtime.

---

## EKS access

=== "Web UI (manual)"

    1. Log into the Britive web portal, check out the profile
    2. Copy AWS credentials into `~/.aws/credentials`
    3. Run `aws eks update-kubeconfig --region <region> --name <cluster> --profile <profile>`
    4. Repeat step 3 for every cluster on the profile
    5. Run `kubectl ...`
    6. Credentials expire -- repeat from step 1

=== "pybritive"

    ```bash
    pybritive checkout "AWS/Sandbox/Developer" -m integrate
    aws eks update-kubeconfig --region us-east-1 --name my-cluster --profile dev
    kubectl get pods
    ```

    Two manual steps. pybritive does the credential checkout, but you still wire up `aws eks update-kubeconfig` yourself for every cluster.

=== "bctl"

    ```bash
    bctl eks connect dev
    kubectl get pods
    ```

    One command. bctl checks out credentials and runs `aws eks update-kubeconfig` for every cluster on the profile, in the right region, with the right profile name. Credentials and kubeconfig stay in sync.

---

## Cloud support

| Cloud | Profile listing | Credential injection | Cluster access |
|---|---|---|---|
| AWS   | Available now | `~/.aws/credentials` and env vars | EKS via `bctl checkout --eks` |
| GCP   | Available now | Coming soon | GKE coming soon |
| Azure | Available now | Coming soon | AKS coming soon |

`bctl profiles sync` and `bctl profiles list` already show profiles from all
three clouds. `bctl checkout` resolves any of them, but writing GCP/Azure
credentials to their respective local formats is on the roadmap -- when you
check out a GCP or Azure profile today, bctl prints a friendly message with
the profile details so you know it's recognized.

bctl is built on the public Britive API, so any cloud Britive supports can be wired in. AWS and EKS are shipping today. GCP and Azure are next on the roadmap.

---

## Next steps

- [Install →](install.md)
- [Quick Start →](quickstart.md)
- [Command Reference →](commands/checkout.md)
