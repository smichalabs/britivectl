# Feedback & issues

bctl has built-in commands for reporting bugs and requesting features. Both open a pre-filled GitHub issue in your browser; bctl never holds a GitHub token of its own.

## Report a bug

```bash
bctl issue bug
```

This opens the new-issue page on the [bctl issue tracker](https://github.com/smichalabs/britivectl-releases/issues) with the bug template selected and an environment block already in the body:

- bctl version
- OS and architecture
- Whether a Britive tenant is configured (the tenant name itself is not included)

You fill in the title and the rest of the details, then click **Submit new issue**.

## Request a feature

```bash
bctl issue feature
```

Same flow, with the feature-request template selected.

## What gets filed

Issues land on the public [`smichalabs/britivectl-releases`](https://github.com/smichalabs/britivectl-releases) repository. Both maintainers are auto-assigned and notified.

You do not need any GitHub configuration on your machine. The browser opens to the new-issue form using your existing GitHub session; if you are not logged in to GitHub, the page will prompt you.

## If the browser does not open

bctl prints the URL to stdout as a fallback so you can copy it manually:

```text
$ bctl issue bug
Opening browser to file an issue...

If the browser did not open, the URL is:
  https://github.com/smichalabs/britivectl-releases/issues/new?body=...&template=bug.yml
```

## Browse existing issues

```bash
open https://github.com/smichalabs/britivectl-releases/issues
```

Search before filing a new one in case the same problem has already been reported.
