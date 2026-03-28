# Release

This directory contains local release tooling for `flarness`.

## Build release artifacts

```bash
./release/build.sh
```

This creates versioned archives under `release/dist/<version>/` and writes
`checksums.txt` for all generated files.

By default the version is resolved from:

1. `RELEASE_VERSION`
2. `git describe --tags --always`
3. `dev`

## Publish with GitHub CLI

```bash
./release/publish-gh.sh
```

Requirements:

- `gh` is installed and available on `PATH`
- `gh auth status` succeeds
- the current repository has a Git remote pointing at GitHub

Environment variables:

- `RELEASE_VERSION`: override the version/tag to publish
- `RELEASE_NOTES_FILE`: optional path to release notes text/markdown
- `GH_REPO`: optional `owner/repo` override

## Typical flow

```bash
./release/build.sh
./release/publish-gh.sh
```
