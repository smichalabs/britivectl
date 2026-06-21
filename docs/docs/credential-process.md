# AWS CLI Integration

`bctl` can hand credentials to the AWS CLI, the AWS SDKs, Terraform, and anything
else that reads `~/.aws/config` automatically, through the standard AWS
[`credential_process`](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sourcing-external.html)
hook. Once it is wired up you never run a checkout by hand: the first AWS call
that needs the profile triggers `bctl`, and the credentials refresh themselves
when they expire.

---

## Overview

`credential_process` is an AWS feature, not a bctl one. In `~/.aws/config` a
profile can name an external command instead of carrying static keys. When a
tool needs credentials for that profile it runs the command, reads the command's
**standard output**, and parses it as JSON. That JSON is the credential.

The shape AWS expects is fixed:

```json
{
  "Version": 1,
  "AccessKeyId": "ASIA...",
  "SecretAccessKey": "...",
  "SessionToken": "...",
  "Expiration": "2026-04-15T18:30:00Z"
}
```

- `Version` must be `1`.
- `Expiration` (RFC 3339) tells the SDK when the credentials die. The SDK caches
  them in memory until then and only re-runs the command when they are close to
  expiry. Without `Expiration` there is no caching, and the command runs on every
  API call.

---

## Wire bctl up

Point a profile at `bctl checkout <alias> --output process` in `~/.aws/config`:

```ini
[profile aws-admin-prod]
credential_process = bctl checkout aws-admin-prod --output process
region = us-east-1
```

Use the full path to `bctl` (for example `/opt/homebrew/bin/bctl`) if the tool
that resolves credentials does not inherit your shell `PATH`.

Now all of these just work, and refresh on their own when the checkout lapses:

```bash
aws s3 ls --profile aws-admin-prod
AWS_PROFILE=aws-admin-prod terraform plan
```

---

## How it flows

```text
aws s3 ls --profile aws-admin-prod
  -> AWS reads ~/.aws/config, sees credential_process for the profile
  -> AWS runs:  bctl checkout aws-admin-prod --output process
  -> bctl checks out (or reuses) the Britive session, prints the JSON to stdout
  -> AWS captures that stdout, parses the JSON, signs the request
  -> AWS caches the credentials until Expiration
```

There is no file to manage and no pipe to set up. The config line is the whole
contract, and `stdout` is the channel.

!!! note "stdout is reserved for the JSON"
    When AWS uses `credential_process`, the command's entire standard output is
    parsed as JSON. `--output process` is built for exactly this: it prints only
    the credential object and sends every status message to stderr, so a
    "reusing existing checkout" notice never lands in front of the JSON.

---

## See also

- [Sessions & caching](sessions.md) for how bctl reuses a live checkout.
- [checkout](commands/checkout.md) for the other `--output` formats
  (`awscreds`, `json`, `env`).
