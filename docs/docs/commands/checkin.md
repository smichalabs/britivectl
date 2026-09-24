# bctl checkin

Return a checked-out Britive profile before it expires.

## Synopsis

```
bctl checkin <alias> [--console]
bctl checkin --all
```

## Description

`checkin` voluntarily returns a profile checkout early, releasing the temporary
credentials before their natural expiry. This is good practice once you're done
with a task.

Programmatic and console checkouts of the same profile are separate sessions.
By default `checkin` returns the programmatic one. Pass `--console` to return
the console session instead. `--all` returns every active session of both
types.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--console` | false | Check in the console session instead of the programmatic one |
| `--all` | false | Check in every active session |

## Examples

```bash
bctl checkin dev
bctl checkin prod-readonly
bctl checkin dev --console
bctl checkin --all
```

## See also

- [Console Access](../console.md)
- [bctl checkout](checkout.md)
- [bctl status](status.md)
