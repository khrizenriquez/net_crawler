#!/bin/sh
set -eu

GO_VERSION="${DUKU_GO_VERSION:-1.23.12}"
if command -v go >/dev/null 2>&1; then
  exec go "$@"
fi

CACHE_DIR="${DUKU_GO_CACHE:-$HOME/.cache/duku-net-lab/go-$GO_VERSION}"
GO_BIN="$CACHE_DIR/go/bin/go"
if [ ! -x "$GO_BIN" ]; then
  mkdir -p "$CACHE_DIR"
  archive="go$GO_VERSION.darwin-arm64.tar.gz"
  echo "Downloading local Go toolchain $archive" >&2
  curl -fsSL "https://go.dev/dl/$archive" | tar -xz -C "$CACHE_DIR"
fi
exec "$GO_BIN" "$@"

