#!/usr/bin/env bash
# Install runtime for the kinko vault TUI (OpenTUI on Bun).
# - Ensures bun is available (~/.bun/bin/bun, installs if missing)
# - Installs @opentui/core deps in opentui/
# - Writes ~/.local/bin/vault-tui launcher
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
OT_DIR="$SRC/opentui"
MAIN="$OT_DIR/src/kinko_tui.ts"

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

# --- launcher ----------------------------------------------------------
WRAP="$HOME/.local/bin/vault-tui"
cat > "$WRAP" <<EOF
#!/usr/bin/env bash
# vault-tui launcher — kinko vault terminal UI (OpenTUI).
# Usage: vault-tui [--check]
set -euo pipefail
TUI=""
for cand in \\
  "$SRC/opentui/src/kinko_tui.ts" \\
  "$HOME/.local/share/kinko/opentui/src/kinko_tui.ts"
do
  if [ -f "\$cand" ]; then
    TUI="\$(cd "\$(dirname "\$cand")/.." && pwd)"
    break
  fi
done
if [ -z "\$TUI" ]; then
  echo "kinko_tui.ts not found (expected $SRC/opentui or ~/.local/share/kinko/opentui)" >&2
  exit 1
fi
if [ ! -x "\$TUI/node_modules/.bin" ] && [ ! -d "\$TUI/node_modules/@opentui" ]; then
  echo "OpenTUI deps not installed in \$TUI — run install-tui.sh" >&2
  exit 1
fi
exec "$BUN" "\$TUI/src/kinko_tui.ts" "\$@"
EOF
chmod 0755 "$WRAP"
mkdir -p "$HOME/.opencode/bin"
ln -sf "$WRAP" "$HOME/.opencode/bin/vault-tui"

echo "installed: $WRAP"
echo "run: vaultctl ui   (or: vault-tui)"
