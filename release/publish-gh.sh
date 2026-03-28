#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${RELEASE_VERSION:-$(git -C "$ROOT_DIR" describe --tags --always 2>/dev/null || echo dev)}"
DIST_DIR="$ROOT_DIR/release/dist/$VERSION"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is not installed or not on PATH" >&2
  exit 1
fi

gh auth status >/dev/null

if [[ ! -d "$DIST_DIR" ]]; then
  echo "release artifacts not found: $DIST_DIR" >&2
  echo "run ./release/build.sh first" >&2
  exit 1
fi

notes_args=()
if [[ -n "${RELEASE_NOTES_FILE:-}" ]]; then
  notes_args=(--notes-file "$RELEASE_NOTES_FILE")
else
  notes_args=(--generate-notes)
fi

repo_args=()
if [[ -n "${GH_REPO:-}" ]]; then
  repo_args=(--repo "$GH_REPO")
fi

gh release create "$VERSION" \
  "${repo_args[@]}" \
  "${notes_args[@]}" \
  "$DIST_DIR"/*

echo "Published GitHub release $VERSION"
