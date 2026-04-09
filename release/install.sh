#!/usr/bin/env bash
set -euo pipefail

APP_NAME="flarness"
REPO="${GH_REPO:-canaanyjn/flarness}"
VERSION="${RELEASE_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
FALLBACK_INSTALL_DIR="${HOME}/.local/bin"
TMP_DIR="$(mktemp -d)"
CHECKSUM_FILE="checksums.txt"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage: ./release/install.sh

Environment:
  GH_REPO          GitHub repo in owner/name form. Default: canaanyjn/flarness
  RELEASE_VERSION  Release tag to install, or "latest". Default: latest
  INSTALL_DIR      Directory to place the flarness binary. Default: /usr/local/bin
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

case "$(uname -s)" in
  Darwin) goos="darwin" ;;
  Linux) goos="linux" ;;
  *)
    echo "unsupported host OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *)
    echo "unsupported host architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

if command -v shasum >/dev/null 2>&1; then
  checksum_cmd=(shasum -a 256)
elif command -v sha256sum >/dev/null 2>&1; then
  checksum_cmd=(sha256sum)
else
  checksum_cmd=()
fi

asset_name="${APP_NAME}"
archive_name="${APP_NAME}_${VERSION}_${goos}_${goarch}.tar.gz"

if [[ "$VERSION" == "latest" ]]; then
  release_tag="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": "\(.*\)".*/\1/p' | head -1)"
  if [[ -z "$release_tag" ]]; then
    echo "failed to resolve latest release tag for $REPO" >&2
    exit 1
  fi
  VERSION="$release_tag"
  archive_name="${APP_NAME}_${VERSION}_${goos}_${goarch}.tar.gz"
fi

archive_url="https://github.com/$REPO/releases/download/$VERSION/$archive_name"
archive_path="$TMP_DIR/$archive_name"
checksum_url="https://github.com/$REPO/releases/download/$VERSION/$CHECKSUM_FILE"
checksum_path="$TMP_DIR/$CHECKSUM_FILE"

curl -fL "$archive_url" -o "$archive_path"
if [[ ${#checksum_cmd[@]} -gt 0 ]]; then
  curl -fL "$checksum_url" -o "$checksum_path"
  expected_checksum="$(awk -v name="./$archive_name" '$2 == name { print $1 }' "$checksum_path")"
  if [[ -z "$expected_checksum" ]]; then
    echo "checksum entry not found for $archive_name" >&2
    exit 1
  fi
  actual_checksum="$("${checksum_cmd[@]}" "$archive_path" | awk '{print $1}')"
  if [[ "$actual_checksum" != "$expected_checksum" ]]; then
    echo "checksum verification failed for $archive_name" >&2
    exit 1
  fi
fi
tar -xzf "$archive_path" -C "$TMP_DIR"

binary_path="$TMP_DIR/${APP_NAME}_${VERSION}_${goos}_${goarch}/$asset_name"
if [[ ! -f "$binary_path" ]]; then
  echo "binary not found in archive: $binary_path" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
target_path="$INSTALL_DIR/$APP_NAME"
if ! install "$binary_path" "$target_path" 2>/dev/null; then
  if [[ "${INSTALL_DIR}" == "/usr/local/bin" ]]; then
    INSTALL_DIR="$FALLBACK_INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
    target_path="$INSTALL_DIR/$APP_NAME"
    install "$binary_path" "$target_path"
  else
    echo "failed to install $APP_NAME to $INSTALL_DIR" >&2
    exit 1
  fi
fi

echo "Installed $APP_NAME $VERSION to $target_path"
