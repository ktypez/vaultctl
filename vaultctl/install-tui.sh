#!/usr/bin/env bash
# Install the vaultctl TUI launcher onto PATH (idempotent; no sudo needed).
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
TUI_SRC="$SRC/vault_tui.py"

if [ ! -f "$TUI_SRC" ]; then
  echo "Error: $TUI_SRC not found"
  exit 1
fi

PY="$(command -v python3 || true)"
if [ -z "$PY" ]; then
  echo "Error: python3 not found"
  exit 1
fi

# Install textual for the system python so the TUI runs anywhere.
if ! "$PY" -c 'import textual' >/dev/null 2>&1; then
  echo "Installing textual into $PY ..."
  "$PY" -m pip install --user --break-system-packages textual 2>/dev/null \
    || "$PY" -m pip install --user textual
fi

WRAP="$HOME/.local/bin/vault-tui"
cat > "$WRAP" <<EOF
#!/usr/bin/env bash
# vault-tui launcher — runs the kinko vault Textual UI.
# Prefers the dev tree, falls back to the service root.
for d in "$TUI_SRC" "$HOME/.local/share/kinko/vault_tui.py"; do
  if [ -f "\$d" ]; then
    exec python3 "\$d" "\$@"
  fi
done
echo "vault_tui.py not found" >&2
exit 1
EOF
chmod 0755 "$WRAP"
mkdir -p "$HOME/.opencode/bin"
ln -sf "$WRAP" "$HOME/.opencode/bin/vault-tui"

echo "installed: $WRAP"
echo "run: vaultctl ui   (or: vault-tui)"
