#!/usr/bin/env bash
# Build the vaultctl Go CLI and install it onto PATH.
# Usage: ./build-cli.sh [--install]
#   (no flag)  build into cli/vaultctl
#   --install  also copy to ~/.local/bin/vaultctl (+ ~/.opencode/bin)
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$SRC/cli"
OUT="$CLI_DIR/vaultctl"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go toolchain not found on PATH" >&2
  exit 1
fi

VERSION="${VAULTCTL_VERSION:-$(git -C "$SRC" describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X github.com/ktypez/vaultctl/cli/cmd.version=$VERSION"

echo "building vaultctl ($VERSION) ..."
( cd "$CLI_DIR" && go build -trimpath -ldflags "$LDFLAGS" -o "$OUT" . )
echo "built: $OUT"

if [[ "${1:-}" == "--install" ]]; then
  mkdir -p "$BIN_DIR"
  install -m 0755 "$OUT" "$BIN_DIR/vaultctl"
  echo "installed: $BIN_DIR/vaultctl"
  if [[ -d "$HOME/.opencode/bin" ]]; then
    ln -sf "$BIN_DIR/vaultctl" "$HOME/.opencode/bin/vaultctl"
    echo "linked: $HOME/.opencode/bin/vaultctl"
  fi
fi
