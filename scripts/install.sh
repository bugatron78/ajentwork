#!/usr/bin/env bash
set -euo pipefail

REPO="bugatron78/ajentwork"
INSTALL_DIR="${HOME}/.local/bin"
MAN_DIR="${HOME}/.local/share/man/man1"
VERSION=""
INSTALL_MANPAGE=1

usage() {
  cat <<'EOF'
Install aj from a GitHub release.

Usage:
  install.sh [--version <tag>] [--install-dir <path>] [--man-dir <path>] [--no-man]

Examples:
  install.sh
  install.sh --version v0.1.6
  install.sh --install-dir "$HOME/.local/bin"
  install.sh --man-dir "$HOME/.local/share/man/man1"
EOF
}

compute_sha256() {
  local file
  file="$1"

  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi

  echo "missing checksum tool: need shasum or sha256sum" >&2
  exit 1
}

verify_archive_checksum() {
  local archive_path checksums_path expected actual
  archive_path="$1"
  checksums_path="$2"

  expected="$(awk -v name="$ARCHIVE_NAME" '$2 == name { print $1 }' "$checksums_path")"
  if [[ -z "$expected" ]]; then
    echo "failed to find checksum for $ARCHIVE_NAME in $(basename "$checksums_path")" >&2
    exit 1
  fi

  actual="$(compute_sha256 "$archive_path")"
  if [[ "$actual" != "$expected" ]]; then
    echo "checksum mismatch for $ARCHIVE_NAME" >&2
    echo "expected: $expected" >&2
    echo "actual:   $actual" >&2
    exit 1
  fi

  echo "Verified checksum for $ARCHIVE_NAME"
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
    --man-dir)
      MAN_DIR="${2:-}"
      shift 2
      ;;
    --no-man)
      INSTALL_MANPAGE=0
      shift
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
CHECKSUMS_NAME="aj_${VERSION}_checksums.txt"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/${CHECKSUMS_NAME}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${DOWNLOAD_URL}"
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME"
echo "Downloading ${CHECKSUMS_URL}"
curl -fsSL "$CHECKSUMS_URL" -o "$TMP_DIR/$CHECKSUMS_NAME"

verify_archive_checksum "$TMP_DIR/$ARCHIVE_NAME" "$TMP_DIR/$CHECKSUMS_NAME"

tar -xzf "$TMP_DIR/$ARCHIVE_NAME" -C "$TMP_DIR"

mkdir -p "$INSTALL_DIR"
install "$TMP_DIR/aj_${VERSION}_${GOOS}_${GOARCH}/aj" "$INSTALL_DIR/aj"

echo "Installed aj to $INSTALL_DIR/aj"
if [[ "$INSTALL_MANPAGE" -eq 1 && -f "$TMP_DIR/aj_${VERSION}_${GOOS}_${GOARCH}/share/man/man1/aj.1" ]]; then
  mkdir -p "$MAN_DIR"
  install "$TMP_DIR/aj_${VERSION}_${GOOS}_${GOARCH}/share/man/man1/aj.1" "$MAN_DIR/aj.1"
  echo "Installed man page to $MAN_DIR/aj.1"
fi

echo "Run '$INSTALL_DIR/aj --version' to verify the installation."
