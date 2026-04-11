# Quick Start

## 30 seconds from install to credentials

```bash
brew tap smichalabs/tap
brew install bctl
bctl
```

Pick a profile, hit enter. Your credentials are in `~/.aws/credentials`.

```bash
aws s3 ls --profile aws-admin-prod
```

Done.

!!! info "What you need"
    A Britive tenant and a user account that can check out at least one JIT profile. bctl uses the [public Britive API](https://docs.britive.com/apidocs) -- your admin doesn't need to enable anything.

!!! tip "First run"
    On a brand-new machine, the very first `bctl` also asks for your tenant name and opens your browser for SSO. Takes about 20 seconds, you only do it once.

!!! info "Sign in once, not every time"
    bctl auto-refreshes your Britive session in the background. You sign in once per day at most, not on every command. And repeat checkouts of the same profile within the credential lifetime are instant -- bctl skips the Britive API entirely. See [Sessions & caching](sessions.md).

---

## Fuzzy search tips

The picker filters as you type. Exact matches are not required.

| You type | You get |
|---|---|
| `admin prod` | `aws-admin-prod` |
| `sandbox`    | every profile with `sandbox` in the name or Britive path |
| `security`   | `aws-security-staging` |
| `data`       | `aws-data-staging` |

Prefer to type the full name? Pass it as an argument:

```bash
bctl checkout aws-admin-prod
```

Partial matches work there too -- `bctl checkout admin-prod` resolves to `aws-admin-prod`. If more than one profile matches, the picker opens pre-filtered.

---

## EKS clusters in one command

```bash
bctl checkout aws-admin-prod --eks
kubectl get pods
```

bctl checks out AWS credentials **and** runs `aws eks update-kubeconfig` for every cluster on the profile. Works across regions.

---

## Shell integration

**One-off session with exported env vars:**

```bash
eval "$(bctl checkout aws-admin-prod --output env)"
aws s3 ls
```

**Auto-refresh via `aws` credential_process:**

Add this to `~/.aws/config`:

```ini
[profile aws-admin-prod]
credential_process = bctl checkout aws-admin-prod --output process
```

Now `aws --profile aws-admin-prod ...` calls bctl transparently whenever credentials are needed. No manual checkout.

---

## You can still call every command directly

The launcher is a shortcut, not a requirement. Every subcommand is available on its own, which is what you want for scripts, CI, and muscle memory:

```bash
bctl checkout <name>    # check out a specific profile
bctl checkout <name> --eks  # checkout + update EKS kubeconfig
bctl status             # show active checkouts and expiry
bctl checkin <name>     # return a checkout early
bctl profiles list      # show everything you can check out
bctl profiles sync      # refresh the profile list
bctl login              # browser SSO
bctl login --token $T   # API token
bctl logout             # clear stored credentials
bctl init               # reconfigure tenant + auth method
bctl doctor             # diagnose setup issues
bctl config get/set     # read or write config values
bctl update             # self-update
bctl version            # print version info
```

Run `bctl <command> --help` for flags and details on any of them. See the [full command table on the home page](index.md#all-commands).
