# Install

## macOS: Homebrew

```bash
brew tap smichalabs/tap
brew install bctl
```

Upgrades:

```bash
brew upgrade bctl
```

## Linux

```bash
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

The script detects your distro and installs the appropriate package:

| Distro | Package | Manager |
|--------|---------|---------|
| Debian / Ubuntu | `.deb` | `dpkg` |
| RHEL / Fedora / CentOS | `.rpm` | `dnf` / `rpm` |
| Everything else | `tar.gz` | extracts to `/usr/local/bin` |

## WSL (Windows Subsystem for Linux)

Same as Linux. Install `wslu` first so browser auth works:

```bash
sudo apt install wslu
curl -fsSL https://raw.githubusercontent.com/smichalabs/britivectl/main/scripts/install.sh | bash
```

!!! note
    `bctl login` opens the browser via `wslview` (from `wslu`). Without it, you'll need to copy the URL manually.

## Build from source

Requires Go 1.25+:

```bash
git clone https://github.com/smichalabs/britivectl.git
cd britivectl
make install          # builds and copies to /opt/homebrew/bin
```

## Shell completions

=== "zsh"

    ```bash
    bctl completion zsh > "${fpath[1]}/_bctl"
    ```

=== "bash"

    ```bash
    bctl completion bash > /usr/local/etc/bash_completion.d/bctl
    ```

=== "fish"

    ```bash
    bctl completion fish > ~/.config/fish/completions/bctl.fish
    ```

## Verify

```bash
bctl version
```
