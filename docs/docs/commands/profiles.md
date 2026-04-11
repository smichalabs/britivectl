# bctl profiles

Manage Britive profile aliases.

## Synopsis

```
bctl profiles <subcommand>
```

## Subcommands

### list

Display all profiles configured in `~/.config/bctl/config.yaml`.

```bash
bctl profiles list
```

### sync

Pull the latest profiles available to you from the Britive API and update
`~/.config/bctl/config.yaml`.

```bash
bctl profiles sync
```

## Examples

```bash
# See what profiles are configured locally
bctl profiles list

# Sync from Britive API after gaining access to new profiles
bctl profiles sync
```

## See also

- [bctl checkout](checkout.md)
- [bctl status](status.md)
