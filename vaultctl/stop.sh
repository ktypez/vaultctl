#!/usr/bin/env bash
# Stop a manually-launched vault (or the systemd unit).
set -euo pipefail
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet vaultctl 2>/dev/null; then
  sudo systemctl stop vaultctl
  echo "stopped systemd unit"
  exit 0
fi
pkill -f "python.*app.py" 2>/dev/null && echo "stopped" || echo "not running"
