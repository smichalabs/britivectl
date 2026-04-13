# bctl status

Show currently active profile checkouts and their expiry times.

## Synopsis

```
bctl status
```

## Description

Queries the Britive API for all active sessions belonging to the current user
and renders them in a table with the profile alias, checkout status, and expiry.

## Examples

```bash
bctl status
```

Example output:

```
PROFILE              STATUS       EXPIRES
aws-admin-prod       checkedOut   2026-04-13T18:30:00Z
aws-data-staging     checkedOut   2026-04-13T19:00:00Z
```

## See also

- [bctl checkout](checkout.md)
- [bctl checkin](checkin.md)
