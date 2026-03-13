#!/usr/bin/env bash
set -euo pipefail

REPO="bugatron78/ajentwork"
INSTALL_DIR="${HOME}/.local/bin"
VERSION=""

usage() {
  cat <<'EOF'
Install aj from a GitHub release.

Usage:
  install.sh [--version <tag>] [--install-dir <path>]

Examples:
  install.sh
  install.sh --version v0.1.0
  install.sh --install-dir "$HOME/.local/bin"
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) os="darwin" ;;
    Linux) os="linux" ;;
    *)
      echo "unsupported operating system: $os" >&2
      exit 1
      ;;
  esac

  case "$arch" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      echo "unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac

  printf '%s %s\n' "$os" "$arch"
}

fetch_latest_version() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
    | head -n 1
}

read -r GOOS GOARCH < <(detect_platform)

if [[ -z "$VERSION" ]]; then
  VERSION="$(fetch_latest_version)"
fi

if [[ -z "$VERSION" ]]; then
  echo "failed to determine release version" >&2
  exit 1
fi

ARCHIVE_NAME="aj_${VERSION}_${GOOS}_${GOARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${DOWNLOAD_URL}"
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"

tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"

mkdir -p "$INSTALL_DIR"
install "$TMP_DIR/aj_${VERSION}_${GOOS}_${GOARCH}/aj" "$INSTALL_DIR/aj"

echo "Installed aj to $INSTALL_DIR/aj"
echo "Run '$INSTALL_DIR/aj --help' to verify the installation."
