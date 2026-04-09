# bctl — Britive CLI

**bctl** is a fast, polished CLI for [Britive](https://www.britive.com) JIT access management.
Replace manual web UI workflows and fragile scripts with a single binary.

```bash
bctl checkout dev        # get AWS credentials in seconds
bctl eks connect dev     # checkout + update kubeconfig
bctl status              # see what's checked out and when it expires
```

---

## Install

=== "macOS"

    ```bash
    brew tap smichalabs/tap
    brew install smichalabs/tap/bctl
    ```

=== "Linux / WSL"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
    ```

    Auto-detects your distro: `.deb`, `.rpm`, or tarball.

=== "Source"

    ```bash
    git clone https://github.com/smichalabs/britivectl.git
    cd britivectl
    make install
    ```

---

## Why bctl?

| Without bctl | With bctl |
|---|---|
| Log into Britive web UI | `bctl checkout dev` |
| Copy credentials manually | Writes to `~/.aws/credentials` automatically |
| Run `aws eks update-kubeconfig` separately | `bctl eks connect dev` does both |
| Check expiry in the browser | `bctl status` shows all active checkouts |
| Script brittle web scraping | Clean API-backed CLI with shell completions |

---

## Next steps

- [Install →](install.md)
- [Quick Start →](quickstart.md)
- [Command Reference →](commands/checkout.md)
