# Comparison

Three ways to get a JIT AWS credential from Britive. Same end result, very different daily workflows.

The big difference: **bctl handles the boilerplate that the other two leave to you.** With the web UI you copy three values and paste them into `~/.aws/credentials` by hand, and with pybritive you remember the right `--mode` flag every time. After either, you still have to run `aws eks update-kubeconfig` separately if you are going to touch a cluster. bctl writes the AWS credentials automatically and updates kubeconfig in the same command.

## At a glance

|  | web UI | pybritive | bctl |
|---|---|---|---|
| Sign in | SSO every checkout | run `pybritive login` separately when the token expires | SSO opens automatically as part of `bctl checkout` when the token expires |
| Find a profile | menu drill-down | type the exact full Britive path | fuzzy match on alias |
| Write to `~/.aws/credentials` | manual copy and paste | requires `-m integrate` flag | automatic, every time |
| Update kubeconfig for EKS | run `aws eks update-kubeconfig` after | run `aws eks update-kubeconfig` after | `--eks` in the same command |
| Repeat checkout (still fresh) | full clickfest again | full API call again | instant, cached |
| Time remaining | not shown | not shown | `bctl status` |
| Release | click Checkin in UI | `pybritive checkin "..."` | `bctl checkin admin-prod` |
| Footprint | browser tab | ~100 MB Python stack | single ~4 MB binary |

## Get a credential

**Britive web UI**

1. Log into the Britive web UI through SSO.
2. Drill into the menu: app -> environment -> profile.
3. Click **Checkout** on the profile and pick a duration.
4. Copy the access key, secret, and session token from the popup.
5. Paste them into `~/.aws/credentials` under the right profile name:

   ```ini
   [admin-prod]
   aws_access_key_id = ASIA...
   aws_secret_access_key = ...
   aws_session_token = ...
   ```

6. Use the AWS CLI:

   ```bash
   aws --profile admin-prod sts get-caller-identity
   ```

7. If you need EKS too, run `aws eks update-kubeconfig` separately.
8. When the credentials expire (usually 1 hour), repeat all of the above.

**pybritive**

One-time setup:

```bash
pip install pybritive[aws]
pybritive configure tenant -t <tenant>
pybritive login
```

Each checkout (the `-m integrate` flag is what writes the AWS credentials, otherwise it just prints them):

```bash
pybritive checkout "AWS/Prod/Admin" -m integrate
aws --profile AWS-Prod-Admin sts get-caller-identity
aws eks update-kubeconfig --name <cluster> --profile AWS-Prod-Admin
```

You type the full Britive path every time. When the token expires you run `pybritive login` again as a separate step. Repeat checkouts re-hit the Britive API even if the credentials are still fresh.

**bctl**

One-time setup is just installing the binary:

```bash
brew install smichalabs/tap/bctl
```

First run walks you through tenant, browser SSO, and profile picker:

```bash
bctl checkout
```

Daily use, with credentials and kubeconfig handled in one command:

```bash
bctl checkout admin-prod --eks
aws --profile admin-prod sts get-caller-identity
kubectl get pods
```

Other commands you will use:

```bash
bctl status                    # show what is checked out and time remaining
bctl checkin admin-prod        # release the credentials when done
```

When the Britive session JWT expires, the next `bctl checkout` opens your browser for SSO automatically and finishes the checkout in the same command -- you do not have to remember to run a separate login step. If your IdP session is still alive, the SSO step is one click; if not, it is the full SSO. Either way, no second command. Cloud credentials themselves are cached locally, so re-checking out a profile that still has time left is instant and skips the Britive API entirely.

## When each makes sense

- **Web UI** for the rare one-off checkout from a machine where you cannot install anything.
- **pybritive** if you already have a Python environment set up and prefer the Britive-maintained tool.
- **bctl** for daily developer work, especially if you check out many profiles or hop between EKS clusters.
