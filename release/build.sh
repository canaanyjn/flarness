#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="flarness"
VERSION="${RELEASE_VERSION:-$(git -C "$ROOT_DIR" describe --tags --always 2>/dev/null || echo dev)}"
DIST_DIR="$ROOT_DIR/release/dist/$VERSION"

TARGETS=(
  "darwin amd64 tar.gz"
  "darwin arm64 tar.gz"
  "linux amd64 tar.gz"
  "linux arm64 tar.gz"
  "windows amd64 zip"
)

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

build_one() {
  local goos="$1"
  local goarch="$2"
  local archive_format="$3"
  local ext=""
  local binary_name="$APP_NAME"

  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
    binary_name="${APP_NAME}.exe"
  fi

  local work_dir
  work_dir="$(mktemp -d "$DIST_DIR/.work.XXXXXX")"
  local out_dir="$work_dir/${APP_NAME}_${VERSION}_${goos}_${goarch}"
  mkdir -p "$out_dir"

  GOOS="$goos" GOARCH="$goarch" \
    go build -ldflags "-X main.version=$VERSION" -o "$out_dir/$binary_name" "$ROOT_DIR"

  cp "$ROOT_DIR/README.md" "$out_dir/README.md"
  if [[ -f "$ROOT_DIR/LICENSE" ]]; then
    cp "$ROOT_DIR/LICENSE" "$out_dir/LICENSE"
  fi

  local archive_base="${APP_NAME}_${VERSION}_${goos}_${goarch}"
  if [[ "$archive_format" == "zip" ]]; then
    (
      cd "$work_dir"
      zip -qr "$DIST_DIR/${archive_base}.zip" "$(basename "$out_dir")"
    )
  else
    tar -C "$work_dir" -czf "$DIST_DIR/${archive_base}.tar.gz" "$(basename "$out_dir")"
  fi

  rm -rf "$work_dir"
}

for target in "${TARGETS[@]}"; do
  # shellcheck disable=SC2086
  build_one $target
done

(
  cd "$DIST_DIR"
  shasum -a 256 ./* > checksums.txt
)

echo "Built release artifacts in $DIST_DIR"
