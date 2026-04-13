# Comparison

How bctl compares to the two existing ways to get JIT credentials from Britive on a developer machine.

|  | Britive web UI | pybritive | **bctl** |
|---|---|---|---|
| **Get credentials** | Log in -> click apps -> click environment -> click profile -> click checkout -> pick a duration -> copy three values from a popup -> paste into `~/.aws/credentials` | `pybritive checkout "AWS/Prod/Admin" -m integrate` | `bctl` (then pick) or `bctl checkout admin-prod` |
| **First-time setup** | None -- just open the browser | `pip install pybritive[aws]` then `pybritive configure tenant -t <name>` then `pybritive login` | `brew install bctl`. The first run does setup interactively. |
| **Subsequent logins** | Sign in every time, click through every time | `pybritive login` again when token expires | Auto-refreshes the session in the background. You sign in once a day at most. |
| **Repeat checkouts of the same profile** | Full clickfest again | Full API call again | Instant. Skips the Britive API if credentials still have life. |
| **Profile name memorization** | Visual click path | Type the exact full Britive path | Fuzzy search the alias, or pass any partial name |
| **Footprint on your machine** | Browser tab + manual paste | ~100 MB Python stack | Single ~4 MB binary, no runtime |
| **EKS kubeconfig setup** | Manual `aws eks update-kubeconfig` after every checkout | Manual after every checkout | `--eks` flag does it in the same command |
| **Shell scriptability** | None | Yes | Yes (`-o env`, `-o process`, `-o json`) |
| **AWS credential_process** | Not supported | Manual config | First-class -- `bctl checkout <name> -o process` |

## When each one makes sense

- **Britive web UI** -- one-off interactive checkout from a machine where you can't install anything.
- **pybritive** -- you already have a Python environment and want the official Britive-maintained tool.
- **bctl** -- daily developer workflow on your own laptop. The fuzzy picker, auto-refresh, and skip-if-fresh cache exist specifically for the "I check out 5+ profiles a day" use case.
