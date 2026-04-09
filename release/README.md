# Release

This directory contains local release tooling for `flarness`.

## Build release artifacts

```bash
./release/build.sh
```

This creates versioned archives under `release/dist/<version>/` and writes
`checksums.txt` for all generated files.

Build only selected targets:

```bash
./release/build.sh darwin/arm64 linux/amd64
```

Or by environment variable:

```bash
RELEASE_TARGETS="darwin/arm64 linux/amd64" ./release/build.sh
```

By default the version is resolved from:

1. `RELEASE_VERSION`
2. `git describe --tags --always`
3. `dev`

## Publish with GitHub CLI

```bash
./release/publish-gh.sh
```

Publish only selected assets from the built release directory:

```bash
./release/publish-gh.sh \
  flarness_v0.1.0_darwin_arm64.tar.gz \
  flarness_v0.1.0_linux_amd64.tar.gz \
  checksums.txt
```

Requirements:

- `gh` is installed and available on `PATH`
- `gh auth status` succeeds
- the current repository has a Git remote pointing at GitHub

Environment variables:

- `RELEASE_VERSION`: override the version/tag to publish
- `RELEASE_NOTES_FILE`: optional path to release notes text/markdown
- `GH_REPO`: optional `owner/repo` override
- `RELEASE_PRERELEASE`: set to `1`/`true` to mark the GitHub release as prerelease

Note:

- `publish-gh.sh` now refuses to publish an inferred `git describe` value like
  `v0.1.0-8-g<sha>` unless you explicitly set `RELEASE_VERSION`. This prevents
  accidental GitHub releases from untagged commits.

## Install a published release

Install the latest release for the current Darwin/Linux host:

```bash
./release/install.sh
```

Install a specific release tag into a custom directory:

```bash
RELEASE_VERSION=v0.1.0 INSTALL_DIR="$HOME/.local/bin" ./release/install.sh
```

`install.sh` downloads the archive and `checksums.txt` from the GitHub release
and verifies the SHA-256 checksum before installing when `shasum` or
`sha256sum` is available.

## Typical flow

```bash
git tag v0.1.1
git push origin v0.1.1
./release/build.sh darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
./release/publish-gh.sh
```
