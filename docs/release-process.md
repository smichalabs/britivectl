# Release process

bctl ships releases automatically. As a contributor you almost never
think about versioning -- the commit message you write decides
everything. This document explains the full pipeline so the next person
to inherit it can debug or extend it without guessing.

---

## TL;DR

1. Open a PR with a [Conventional Commits](#conventional-commit-types)
   message.
2. Merge the PR to `main`.
3. **release-please** opens a "release PR" with a proposed version bump
   and CHANGELOG entry.
4. Merge the release PR.
5. **release-please** tags the release. **goreleaser** builds binaries,
   publishes them to the public `britivectl-releases` repo, and updates
   the Homebrew tap.
6. Users run `brew upgrade bctl`.

You never type a version number. You never edit `CHANGELOG.md`. You
never push a tag manually.

---

## How the version number is decided

Versioning follows [Semantic Versioning](https://semver.org/) -- but
**release-please** handles all of it based on conventional commit
prefixes since the last tag.

| Commit prefix on `main` since last release | Effect on version | Why |
|---|---|---|
| `feat:` | minor bump (`0.2.x` -> `0.3.0`) | new user-facing feature |
| `fix:` | patch bump (`0.2.0` -> `0.2.1`) | bug fix |
| `perf:` | patch bump | performance improvement |
| `sec:` | patch bump | security fix |
| `feat!:` or any commit with `BREAKING CHANGE:` in the body | major bump | breaks the public API |
| `docs:`, `chore:`, `ci:`, `refactor:`, `test:` | **no bump** | not user-visible |

While the project is in `0.x` (pre-1.0), `feat!:` and `BREAKING CHANGE:`
bump the **minor** version, not major. SemVer permits this because
`0.x` is by definition unstable. Once we tag `v1.0.0` for the first
time, breaking changes start bumping the major version.

This is configured in `.release-please-config.json` via
`"bump-minor-pre-major": true`.

### What if multiple commit types land in one release window?

release-please picks the **largest** bump implied by any commit. A
release window with one `fix:` and one `feat:` produces a minor bump,
not a patch. The CHANGELOG groups them under separate sections.

### What if I land only `chore:` and `docs:` commits?

Nothing happens. release-please runs on every push to `main`, sees
that no commit warrants a version bump, and exits. The release PR is
not created. This is correct -- there is nothing for users to install.

---

## Conventional commit types

bctl uses the standard set plus a couple of conventions specific to
security and performance. The `commit-msg` git hook enforces this
locally, and a CI check enforces it on PRs.

| Type | Use for | Bumps version? |
|---|---|---|
| `feat` | new user-facing functionality | yes (minor in 0.x) |
| `fix` | bug fix | yes (patch) |
| `perf` | performance improvement | yes (patch) |
| `sec` | security fix or hardening | yes (patch) |
| `refactor` | restructuring without behavior change | no |
| `docs` | documentation only | no |
| `chore` | dependency bumps, build config, housekeeping | no |
| `ci` | CI / pipeline changes | no |
| `test` | adding or modifying tests only | no |

### Scopes

Scopes are optional but recommended when the change is localized:

```
feat(checkout): add --force flag
fix(auth): retry on 502 from Britive
chore(deps): bump cobra to 1.10.2
ci: speed up linter cache
docs(readme): clarify GCP support status
```

### Breaking changes

Two ways to mark a commit as breaking:

```
feat!: rename --eks flag to --kube
```

or in the body:

```
feat: rename --eks flag to --kube

BREAKING CHANGE: --eks no longer exists. Use --kube instead.
```

In `0.x`, this still bumps minor. After `1.0.0`, it bumps major.

---

## What happens after you merge a PR

```
        you merge feature PR
                |
                v
       push to main triggers
   .github/workflows/release-please.yml
                |
                v
   release-please scans commits since
   the last tag (from .release-please-manifest.json)
                |
       any feat/fix/perf/sec? --- no ---> exit, do nothing
                |
                yes
                |
                v
   open or update PR titled
   "chore(main): release X.Y.Z"
   with bumped version + CHANGELOG entries
                |
                v
        you review and merge
                |
                v
   push to main triggers release-please again
                |
                v
   release-please creates tag vX.Y.Z and a draft GitHub release
                |
                v
   pushing the tag triggers
   .github/workflows/release.yml
                |
                v
            goreleaser
                |
                v
+----+----+----+----+----+----+----+
|    |    |    |    |    |    |
v    v    v    v    v    v    v
build SBOM cosign tar deb rpm Homebrew
binaries via via   .gz                tap
        syft cosign                   PR
                |                     |
                v                     v
        publish to            update bctl.rb
   britivectl-releases       in homebrew-tap
   (public GitHub repo)
                                      |
                                      v
                            user runs `brew upgrade bctl`
```

### Why the tag has to be pushed twice

There is one operational quirk worth knowing: when **release-please**
creates a tag using the default `GITHUB_TOKEN`, GitHub deliberately
does not fire downstream workflows on it (to prevent recursive loops).
So the release workflow does not auto-trigger.

The current workaround is to manually delete and re-push the tag from
a developer machine, which counts as a "human push" and triggers the
release workflow normally:

```bash
git fetch --tags
git push origin :refs/tags/vX.Y.Z
git push origin vX.Y.Z
```

A cleaner long-term fix is to give release-please a Personal Access
Token (PAT) so the tag is created by a real user identity. We will do
this when it becomes annoying.

---

## What goreleaser does on each release

Defined in `.goreleaser.yaml` and triggered by tag pushes via
`.github/workflows/release.yml`. Each release produces:

- **Binaries** for `darwin/amd64`, `darwin/arm64`, `linux/amd64`,
  `linux/arm64`, with `-trimpath` and `-buildvcs=true` flags so panics
  do not leak the build host's directory layout and `go version -m
  bctl` shows the source commit.
- **Reproducible archives** -- `mod_timestamp` is pinned to the commit
  timestamp, so two builds of the same source produce byte-identical
  `.tar.gz` files.
- **Linux packages**: `.deb`, `.rpm`, `.apk` for both architectures.
- **CycloneDX SBOM** (`*.sbom.json`) generated by syft, listing every
  Go module and version that ended up in the binary.
- **Cosign signature** of the `checksums.txt` file using GitHub OIDC
  keyless mode. Anyone can verify a downloaded binary:

  ```bash
  cosign verify-blob \
    --certificate checksums.txt.pem \
    --signature checksums.txt.sig \
    checksums.txt \
    --certificate-identity-regexp 'https://github.com/smichalabs/britivectl' \
    --certificate-oidc-issuer 'https://token.actions.githubusercontent.com'
  ```

- **GitHub release** in the public `smichalabs/britivectl-releases`
  repo with all of the above attached.
- **Homebrew formula update** in the `smichalabs/homebrew-tap` repo
  pointing at the new release.

The source repo (`smichalabs/britivectl`) is public. Binary releases
are published to the separate `smichalabs/britivectl-releases` repo
so that release artifacts are cleanly separated from source history.
The Homebrew tap points at the releases repo for downloads.

---

## Where the version number is stored

Three places need to stay in sync. release-please does this for you;
do not edit them by hand.

| File | What it stores | Updated by |
|---|---|---|
| `.release-please-manifest.json` | The current version (just `{".": "0.3.0"}`) | release-please's release PR |
| `CHANGELOG.md` | Per-version release notes generated from commit messages | release-please's release PR |
| Git tag `vX.Y.Z` | The released version | release-please after merge |
| Binary `version` package | Injected at build time via `-ldflags -X` | goreleaser at build |

If any of these drift out of sync, the fix is to merge a release PR --
do not hand-edit.

---

## Configuring release-please

`.release-please-config.json`:

```json
{
  "release-type": "go",
  "bump-minor-pre-major": true,
  "include-component-in-tag": false,
  "include-v-in-tag": true,
  "packages": {
    ".": { "package-name": "bctl" }
  },
  "changelog-sections": [
    { "type": "feat",     "section": "Features" },
    { "type": "fix",      "section": "Bug Fixes" },
    { "type": "perf",     "section": "Performance" },
    { "type": "sec",      "section": "Security" },
    { "type": "refactor", "hidden": true },
    { "type": "docs",     "hidden": true },
    { "type": "chore",    "hidden": true },
    { "type": "ci",       "hidden": true },
    { "type": "test",     "hidden": true }
  ]
}
```

The `hidden` flag means commits of those types do not appear in the
CHANGELOG and (because they are not in the bump list) do not bump the
version.

---

## Troubleshooting

**The release PR did not appear after I merged a feat: commit.**

Check the workflow run at
`https://github.com/smichalabs/britivectl/actions/workflows/release-please.yml`.
The most common causes:

- The commit type is `chore`, `docs`, `ci`, etc. -- no version bump,
  no PR.
- release-please could not authenticate -- needs `contents: write` and
  `pull-requests: write` permissions, both granted in the workflow YAML.
- Org-level "Allow GitHub Actions to create and approve pull requests"
  setting is off. Fix at the org settings page.

**The release workflow did not run after release-please tagged.**

This is the "tag has to be pushed twice" quirk above. Re-push the tag
manually:

```bash
git fetch --tags
git push origin :refs/tags/vX.Y.Z
git push origin vX.Y.Z
```

**goreleaser failed with "Repository is empty".**

The `britivectl-releases` repo needs at least one commit before a
release can be tagged against it. We initialized it with a README the
first time. If it ever gets emptied, just push a placeholder commit.

**Cosign signing failed.**

Check that `id-token: write` is in the workflow `permissions:` block.
Without it, GitHub will not issue an OIDC token to cosign and keyless
signing fails.
