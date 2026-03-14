#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="$ROOT_DIR/dist"
BUILD_DIR="$DIST_DIR/build"
VERSION="${1:-}"
COMMIT="$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || true)"
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

if [[ -z "$VERSION" ]]; then
  if VERSION="$(git -C "$ROOT_DIR" describe --tags --always --dirty 2>/dev/null)"; then
    :
  else
    VERSION="dev"
  fi
fi

TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
)

LDFLAGS=(
  "-s"
  "-w"
  "-X" "ajentwork/internal/buildinfo.Version=${VERSION}"
  "-X" "ajentwork/internal/buildinfo.Commit=${COMMIT}"
  "-X" "ajentwork/internal/buildinfo.Date=${BUILD_DATE}"
)

MANPAGE_PATH="$ROOT_DIR/docs/aj.1"

mkdir -p "$DIST_DIR"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
rm -f "$DIST_DIR"/aj_*.tar.gz "$DIST_DIR"/aj_*_checksums.txt

echo "Generating man page"
(
  cd "$ROOT_DIR"
  go run ./cmd/ajgenman --output "$MANPAGE_PATH"
)

for target in "${TARGETS[@]}"; do
  read -r GOOS GOARCH <<<"$target"
  ARCHIVE_STEM="aj_${VERSION}_${GOOS}_${GOARCH}"
  STAGE_DIR="$BUILD_DIR/$ARCHIVE_STEM"
  BINARY_NAME="aj"
  MAN_DIR="$STAGE_DIR/share/man/man1"

  mkdir -p "$STAGE_DIR"
  mkdir -p "$MAN_DIR"

  echo "Building $ARCHIVE_STEM"
  (
    cd "$ROOT_DIR"
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
      go build -trimpath -ldflags "${LDFLAGS[*]}" -o "$STAGE_DIR/$BINARY_NAME" ./cmd/aj
  )

  cat >"$STAGE_DIR/INSTALL.txt" <<EOF
aj ${VERSION}

Install:
  1. Move the aj binary somewhere on your PATH.
  2. Run 'aj --version' to verify the installation.

Source:
  https://github.com/bugatron78/ajentwork
EOF

  cp "$MANPAGE_PATH" "$MAN_DIR/aj.1"

  tar -C "$BUILD_DIR" -czf "$DIST_DIR/${ARCHIVE_STEM}.tar.gz" "$ARCHIVE_STEM"
done

(
  cd "$DIST_DIR"
  shasum -a 256 ./*.tar.gz > "aj_${VERSION}_checksums.txt"
)

echo "Artifacts written to $DIST_DIR"
