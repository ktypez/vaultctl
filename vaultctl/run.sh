#!/usr/bin/env bash
# Manual launcher for vaultctl.
# Usage: ./run.sh            (mock vault by default — dev repo)
#        VAULT_REAL=1 ./run.sh  (real encrypted vault)
#        ./run.sh --daemon     (detached)
set -euo pipefail
cd "$(dirname "$0")"
mkdir -p logs

if [[ "${VAULT_REAL:-}" == "1" || "${VAULT_REAL:-}" == "true" ]]; then
  APP=app.py
else
  APP=mock_vault.py
fi

export VAULT_HOST="${VAULT_HOST:-127.0.0.1}"
export VAULT_PORT="${VAULT_PORT:-8787}"

if [[ "${1:-}" == "--daemon" ]]; then
  setsid ./venv/bin/python "$APP" >> logs/vault.log 2>&1 &
  sleep 1
  echo "started on http://$VAULT_HOST:$VAULT_PORT (pid $!, app=$APP)"
else
  exec ./venv/bin/python "$APP"
fi
