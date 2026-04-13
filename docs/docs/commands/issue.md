# bctl issue

File a bug report or feature request against bctl.

## Synopsis

```
bctl issue bug
bctl issue feature
```

## Description

Opens a pre-filled GitHub issue in your browser. bctl gathers local environment context (version, OS / arch, whether a Britive tenant is configured) and pre-fills the issue body so you do not have to type that information manually.

bctl never holds or stores a GitHub token of its own -- the browser uses your existing GitHub session.

## Examples

```bash
bctl issue bug      # opens the bug-report template
bctl issue feature  # opens the feature-request template
```

## Where issues land

The public [`smichalabs/britivectl`](https://github.com/smichalabs/britivectl) repository. Both maintainers are auto-assigned and notified.

## See also

- [Feedback & issues](../feedback.md) -- the full guide
