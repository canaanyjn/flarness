#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${RELEASE_VERSION:-$(git -C "$ROOT_DIR" describe --tags --always 2>/dev/null || echo dev)}"
DIST_DIR="$ROOT_DIR/release/dist/$VERSION"

usage() {
  cat <<'EOF'
Usage: ./release/publish-gh.sh [asset...]

When no assets are given, all files under release/dist/<version>/ are uploaded.

Examples:
  ./release/publish-gh.sh
  ./release/publish-gh.sh flarness_v0.1.0_darwin_arm64.tar.gz checksums.txt

Environment:
  RELEASE_VERSION    Override the release version/tag to publish.
  RELEASE_NOTES_FILE Optional path to release notes text/markdown.
  GH_REPO            Optional owner/repo override.
  RELEASE_PRERELEASE Set to 1/true to mark the release as prerelease.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is not installed or not on PATH" >&2
  exit 1
fi

gh auth status >/dev/null

if [[ -z "${RELEASE_VERSION:-}" && "$VERSION" == *"-g"* ]]; then
  echo "refusing to publish release with inferred non-tag version: $VERSION" >&2
  echo "set RELEASE_VERSION to an explicit tag (for example v0.1.1) or create and checkout the tag first" >&2
  exit 1
fi

if [[ ! -d "$DIST_DIR" ]]; then
  echo "release artifacts not found: $DIST_DIR" >&2
  echo "run ./release/build.sh first" >&2
  exit 1
fi

assets=()
if (($# > 0)); then
  for asset in "$@"; do
    if [[ -f "$asset" ]]; then
      assets+=("$asset")
      continue
    fi

    if [[ -f "$DIST_DIR/$asset" ]]; then
      assets+=("$DIST_DIR/$asset")
      continue
    fi

    echo "release asset not found: $asset" >&2
    exit 1
  done
else
  for asset in "$DIST_DIR"/*; do
    assets+=("$asset")
  done
fi

notes_args=()
if [[ -n "${RELEASE_NOTES_FILE:-}" ]]; then
  notes_args=(--notes-file "$RELEASE_NOTES_FILE")
else
  notes_args=(--generate-notes)
fi

repo_args=("")
if [[ -n "${GH_REPO:-}" ]]; then
  repo_args=(--repo "$GH_REPO")
fi

release_flags=("")
case "${RELEASE_PRERELEASE:-}" in
  1|true|TRUE|yes|YES)
    release_flags+=(--prerelease)
    ;;
esac

gh release create "$VERSION" \
  "${repo_args[@]:1}" \
  "${release_flags[@]:1}" \
  "${notes_args[@]}" \
  "${assets[@]}"

echo "Published GitHub release $VERSION"
