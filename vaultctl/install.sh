#!/usr/bin/env bash
# Install Vaultctl system files (systemd unit) from the service root.
set -euo pipefail

ROOT="${VAULTCTL_SERVICE_ROOT:-$(cd "$(dirname "$0")" && pwd)}"
UNIT_SRC="$ROOT/systemd/vaultctl.service"
UNIT_DST="/etc/systemd/system/vaultctl.service"

if [ ! -f "$UNIT_SRC" ]; then
  echo "Error: $UNIT_SRC not found"
  exit 1
fi

echo "Installing vaultctl.service..."
sudo cp "$UNIT_SRC" "$UNIT_DST"
# remove old kinko.service if present
if [ -f /etc/systemd/system/kinko.service ]; then
  sudo systemctl disable kinko.service 2>/dev/null || true
  sudo rm -f /etc/systemd/system/kinko.service /etc/systemd/system/multi-user.target.wants/kinko.service
fi
sudo systemctl daemon-reload
sudo systemctl enable vaultctl.service
sudo systemctl restart vaultctl.service
echo "Vaultctl is now running."
echo "Status: $(sudo systemctl is-active vaultctl.service)"
