# Quick Start

!!! info "Prerequisites"
    You need an active [Britive](https://www.britive.com) tenant and a user account with access to at least one JIT profile. bctl talks to the public [Britive API](https://docs.britive.com/apidocs) -- your tenant admin does not need to enable anything extra.

## The one-command path

```bash
bctl checkout dev
aws s3 ls --profile dev
```

That's it.

On first run `bctl checkout` is an orchestrator: it walks you through tenant
setup, opens the browser for SSO, syncs your profiles from the Britive API,
and writes credentials to `~/.aws/credentials`. Subsequent runs skip every
step that's already done and cache profiles for 24 hours.

!!! tip "Aliases and matching"
    The argument to `checkout` can be an exact alias (`dev`), a substring of
    the alias or Britive path (`sandbox`), or a fuzzy match. If nothing
    matches or the match is ambiguous, you get an interactive picker.

    Leaving it out entirely (`bctl checkout`) launches the picker with every
    profile you have.

---

## The step-by-step path (optional)

The sub-commands still exist if you want explicit control:

```bash
bctl init               # configure tenant + auth method
bctl login              # browser SSO or --token
bctl profiles sync      # pull the latest profiles
bctl profiles list      # see aliases
bctl checkout dev       # check out a specific profile
bctl status             # show active checkouts
bctl checkin dev        # return a checkout early
```

You only need these if you want to script around bctl's individual steps.
For everyday use, `bctl checkout` handles everything.

---

## Common workflows

### AWS CLI access

```bash
bctl checkout dev
aws s3 ls --profile dev
```

### Shell environment

```bash
eval $(bctl checkout dev --output env)
aws s3 ls   # uses exported variables
```

### EKS cluster access

```bash
bctl checkout dev --eks
kubectl get pods
```

The `--eks` flag writes AWS credentials and runs `aws eks update-kubeconfig`
for every cluster listed on the profile, in the right region, with the right
profile name.

### AWS credential_process

Add to `~/.aws/config`:

```ini
[profile dev]
credential_process = bctl checkout dev --output process
```

Then `aws` will call `bctl` automatically when credentials are needed. No manual checkout required.
