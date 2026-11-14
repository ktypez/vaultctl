#!/usr/bin/env bash
# Deploy vaultctl from the dev repo to the service root (no git, runtime files).
# Dev = this git repo; Service = $VAULTCTL_SERVICE_ROOT (default ~/.local/share/vaultctl).
# Data lives in $VAULTCTL_SERVICE_ROOT/data and is never touched by this script.
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
DST="${VAULTCTL_SERVICE_ROOT:-$HOME/.local/share/vaultctl}"

mkdir -p "$DST"
rsync -a --delete \
  --exclude '.git' \
  --exclude 'node_modules' \
  --exclude '__pycache__' \
  --exclude 'data' \
  --exclude 'cli/vaultctl' \
  "$SRC/" "$DST/"

mkdir -p "$DST/data/logs"
echo "deployed: $SRC -> $DST"

# Install the vaultctl CLI onto PATH (idempotent).
# The CLI is the Go binary (built here): one static binary that provides
# every command plus the `vaultctl ui` dashboard.
if [ -x "$SRC/build-cli.sh" ]; then
  "$SRC/build-cli.sh" --install
elif [ -x "$SRC/vaultctl" ]; then
  install -m 0755 "$SRC/vaultctl" "$HOME/.local/bin/vaultctl"
  mkdir -p "$HOME/.opencode/bin"
  ln -sf "$HOME/.local/bin/vaultctl" "$HOME/.opencode/bin/vaultctl"
  echo "installed: vaultctl -> $HOME/.local/bin/vaultctl"
fi

# run installer if present
if [ -x "$DST/install.sh" ]; then
  VAULTCTL_SERVICE_ROOT="$DST" "$DST/install.sh"
fi
