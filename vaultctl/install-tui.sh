#!/usr/bin/env bash
# Install runtime for the vaultctl TUI (OpenTUI on Bun).
# - Ensures bun is available (~/.bun/bin/bun, installs if missing)
# - Installs @opentui/core deps in opentui/
# - Removes any old vault-tui launcher (vaultctl ui is the single entry point)
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
OT_DIR="$SRC/opentui"
MAIN="$OT_DIR/src/vaultctl-ui.ts"

if [ ! -f "$MAIN" ]; then
  echo "Error: $MAIN not found" >&2
  exit 1
fi

# --- ensure bun --------------------------------------------------------
if command -v bun >/dev/null 2>&1; then
  BUN="$(command -v bun)"
elif [ -x "$HOME/.bun/bin/bun" ]; then
  BUN="$HOME/.bun/bin/bun"
else
  echo "Installing bun..."
  curl -fsSL https://bun.sh/install | bash >/dev/null
  BUN="$HOME/.bun/bin/bun"
fi
echo "bun: $BUN ($("$BUN" --version))"

# --- install OpenTUI deps ----------------------------------------------
echo "Installing OpenTUI deps into $OT_DIR ..."
( cd "$OT_DIR" && "$BUN" install --backend=copyfile )

# --- remove old vault-tui launcher (single entry point: vaultctl ui) ---
for p in "$HOME/.local/bin/vault-tui" "$HOME/.opencode/bin/vault-tui"; do
  if [ -e "$p" ] || [ -L "$p" ]; then
    rm -f "$p"
    echo "removed old launcher: $p"
  fi
done

echo "installed TUI runtime"
echo "run: vaultctl ui"
