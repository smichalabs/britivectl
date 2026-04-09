# bctl status

Show currently active profile checkouts and their expiry times.

## Synopsis

```
bctl status
```

## Description

Queries the Britive API for all active sessions belonging to the current user
and renders them in a table with profile name, cloud, and expiry time.

## Examples

```bash
bctl status
```

Example output:

```
PROFILE       CLOUD   EXPIRES
dev           aws     2026-04-09T14:30:00Z
staging-gcp   gcp     2026-04-09T15:00:00Z
```

## See also

- [bctl checkout](checkout.md)
- [bctl checkin](checkin.md)
