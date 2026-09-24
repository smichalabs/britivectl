# Console Access

bctl can open a cloud provider's web console for a Britive profile, in addition to writing programmatic credentials. One command checks out console access and opens the console in your default browser, already signed in.

## Open the console

```bash
bctl checkout aws-admin-prod --console
```

bctl asks Britive for a console checkout of the profile, waits for it to be ready, and opens the sign-in URL in your default browser. Nothing is written to `~/.aws/credentials` and the local credential cache is not touched.

If you leave out the alias, the profile picker opens as usual:

```bash
bctl checkout --console
```

## Print the sign-in URL instead

On a machine without a browser, for example over SSH, print the URL and open it somewhere else:

```bash
bctl checkout aws-admin-prod --print-url
```

`--print-url` implies `--console`. bctl also prints the URL if it cannot launch a browser.

!!! warning "The sign-in URL is a credential"
    Anyone who has the URL can open the console as you until it expires. Do not paste it into chat, tickets, or shared terminals. bctl only prints it when the browser cannot be opened or when you pass `--print-url`.

## Console and programmatic access are separate

Britive treats console and programmatic access to the same profile as two separate checkouts:

- The profile must allow console access. If it does not, Britive rejects the checkout and bctl shows the error.
- Holding one does not give you the other. You can have both active at once.
- Running `bctl checkout <alias> --console` again while a console checkout is active reuses it instead of starting a new one.
- A plain `bctl checkout <alias>` only looks at programmatic checkouts and never reuses a console one.

## Return console access early

```bash
bctl checkin aws-admin-prod --console
```

Without `--console`, `bctl checkin` returns the programmatic checkout. `bctl checkin --all` returns every active checkout of both types.

## Flags that do not apply

Console access has no local credentials, so these flags cannot be combined with `--console`:

| Flag | Why |
|------|-----|
| `--eks`, `--cluster`, `--region` | kubeconfig needs programmatic credentials |
| `-o, --output` | there are no credentials to format |
| `-f, --force` | Britive allows one active console checkout per profile, and bctl reuses it |

## Supported clouds

Britive generates the console sign-in URL, so `--console` works with any profile that grants console access, including GCP and Azure profiles that do not yet support programmatic credential injection in bctl.

## See also

- [bctl checkout](commands/checkout.md)
- [bctl checkin](commands/checkin.md)
- [Sessions & caching](sessions.md)
