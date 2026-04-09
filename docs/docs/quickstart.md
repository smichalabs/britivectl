# Quick Start

## 1. Set up your tenant

```bash
bctl init
```

The wizard will ask for your Britive tenant name (e.g. `acme` for `acme.britive-app.com`) and preferred auth method.

## 2. Log in

=== "Browser SSO (recommended)"

    ```bash
    bctl login
    ```

    Opens your browser to complete SSO. Token is stored securely in the OS keychain.

=== "API token"

    ```bash
    bctl login --token <your-api-token>
    ```

## 3. Sync profiles

```bash
bctl profiles sync
```

Fetches all profiles you have access to and saves them locally as aliases.

## 4. Check out credentials

```bash
bctl checkout dev
```

Obtains temporary AWS credentials and writes them to `~/.aws/credentials` under the profile's alias.

## 5. Check status

```bash
bctl status
```

Shows all active checkouts and their expiry times.

## 6. Check in (optional)

```bash
bctl checkin dev
```

Returns credentials early. They expire automatically — this is just for when you're done early.

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
bctl eks connect dev
kubectl get pods
```

### AWS credential_process

Add to `~/.aws/config`:

```ini
[profile dev]
credential_process = bctl checkout dev --output process
```

Then `aws` will call `bctl` automatically when credentials are needed — no manual checkout required.
