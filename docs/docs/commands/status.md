# bctl status

Show currently active profile checkouts and their expiry times.

## Synopsis

```
bctl status
```

## Description

Queries the Britive API for all active sessions belonging to the current user
and renders them in a table with the profile alias, checkout status, the
absolute expiry timestamp (UTC), and the time remaining until expiry.

## Examples

```bash
bctl status
```

Example output:

```
PROFILE              STATUS       EXPIRES                  REMAINING
aws-admin-prod       checkedOut   2026-04-13 18:30:00 UTC  3h 47m
aws-data-staging     checkedOut   2026-04-13 19:00:00 UTC  4h 17m
```

`REMAINING` shows `expired` when the deadline has passed and `?` if the API
returns an unparseable timestamp.

## See also

- [bctl checkout](checkout.md)
- [bctl checkin](checkin.md)
