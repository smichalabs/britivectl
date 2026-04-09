# bctl checkin

Return a checked-out Britive profile before it expires.

## Synopsis

```
bctl checkin <alias>
```

## Description

`checkin` voluntarily returns a profile checkout early, releasing the temporary
credentials before their natural expiry. This is good practice once you're done
with a task.

## Examples

```bash
bctl checkin dev
bctl checkin prod-readonly
```

## See also

- [bctl checkout](checkout.md)
- [bctl status](status.md)
