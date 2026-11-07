#!/usr/bin/env bash
# install-backup.sh — install the encrypted vaultctl database backup on this host.
#
# Installs three artifacts and arms a weekly user timer:
#   <bin>/vaultctl-db-backup                     the backup engine   (mode 0700)
#   <units>/vaultctl-db-backup.service|.timer     user systemd units (mode 0644)
#
# It never prints secret material: the GPG envelope passphrase under
# <config>/encryption-passphrase (mode 0600) is generated if absent and reported
# by path only. It does not create or write the private backup repository's
# contents — the engine does that, and only ever with ciphertext.
#
# Run from a checkout (uses the sibling artifacts), or pipe the raw script and it
# fetches the artifacts from GitHub. Overridable paths/env:
#   VAULTCTL_BIN_DIR            default: $HOME/.local/bin
#   VAULTCTL_USER_UNIT_DIR      default: $HOME/.config/systemd/user
#   VAULTCTL_BACKUP_CONFIG_DIR  default: $HOME/.config/vaultctl-db-backup
#   VAULTCTL_BACKUP_REPO        default: ktypez/vaultctl-db-backup
#   VAULTCTL_SRC_RAW_BASE       raw URL base used when run without siblings
set -Eeuo pipefail
umask 077

SELF_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]:-/dev/stdin}")" 2>/dev/null && pwd || echo .)"
BIN_DIR="${VAULTCTL_BIN_DIR:-$HOME/.local/bin}"
UNIT_DIR="${VAULTCTL_USER_UNIT_DIR:-$HOME/.config/systemd/user}"
CFG_DIR="${VAULTCTL_BACKUP_CONFIG_DIR:-$HOME/.config/vaultctl-db-backup}"
BACKUP_REPO="${VAULTCTL_BACKUP_REPO:-ktypez/vaultctl-db-backup}"
RAW_BASE="${VAULTCTL_SRC_RAW_BASE:-https://raw.githubusercontent.com/ktypez/vaultctl/master/vaultctl}"
PASSFILE="$CFG_DIR/encryption-passphrase"

DRY_RUN=0 NO_ENABLE=0 VERIFY=0 REQUIRE_GH=0
UNIT_NAMES=(vaultctl-db-backup.service vaultctl-db-backup.timer)

say()  { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }
run()  { if (( DRY_RUN )); then say "  [dry-run] $*"; else "$@"; fi; }

usage() {
  cat <<'EOF'
Usage: install-backup.sh [options]

Installs the vaultctl encrypted database backup (script + weekly user timer).

Options:
  --dry-run     print the actions without changing anything
  --no-enable   install files but do not enable/start the systemd timer
  --verify      check an existing installation and exit non-zero on mismatch
  --require-gh  fail (instead of warning) when gh auth/repo checks do not pass
  -h, --help    show this help

Environment overrides: VAULTCTL_BIN_DIR, VAULTCTL_USER_UNIT_DIR,
VAULTCTL_BACKUP_CONFIG_DIR, VAULTCTL_BACKUP_REPO, VAULTCTL_SRC_RAW_BASE.
EOF
}

while (( $# )); do
  case "$1" in
    --dry-run)    DRY_RUN=1 ;;
    --no-enable)  NO_ENABLE=1 ;;
    --verify)     VERIFY=1 ;;
    --require-gh) REQUIRE_GH=1 ;;
    -h|--help)    usage; exit 0 ;;
    *)            usage >&2; die "unknown option: $1" ;;
  esac
  shift
done

install_artifact() { # rel source-path, dest, mode
  local rel="$1" dest="$2" mode="$3" src="$SELF_DIR/$1" tmp
  if [[ -f "$src" ]]; then
    run install -m "$mode" -- "$src" "$dest"
  else
    [[ -n "$RAW_BASE" ]] || die "missing $rel and no VAULTCTL_SRC_RAW_BASE set"
    if (( DRY_RUN )); then
      say "  [dry-run] curl -fsSL $RAW_BASE/$rel -o <tmp> && install -m $mode - <$dest>"
      return 0
    fi
    tmp="$(mktemp)"
    trap 'rm -f -- "$tmp"' RETURN
    curl -fsSL "$RAW_BASE/$rel" -o "$tmp" || die "could not fetch $RAW_BASE/$rel"
    install -m "$mode" -- "$tmp" "$dest"
    rm -f -- "$tmp"
  fi
}

check_deps() {
  local missing=() c
  for c in bash curl install gh git gpg flock python3 stat; do
    command -v "$c" >/dev/null || missing+=("$c")
  done
  if ((${#missing[@]})); then
    die "missing required commands: ${missing[*]}
install them first (e.g. Debian/Ubuntu: sudo apt-get install curl coreutils gnupg git util-linux python3; GitHub CLI: https://cli.github.com), then 'gh auth login'"
  fi
  return 0
}

# ---------------------------------------------------------------- verify mode
if (( VERIFY )); then
  rc=0
  report() { # label, ok?
    if [[ "$2" == ok ]]; then say "  ok    $1"; else say "  FAIL  $1"; rc=1; fi
  }
  say "verifying installed backup:"
  [[ -x "$BIN_DIR/vaultctl-db-backup" ]] && report "$BIN_DIR/vaultctl-db-backup (executable)" ok || report "$BIN_DIR/vaultctl-db-backup missing/not executable" no
  for u in "${UNIT_NAMES[@]}"; do
    [[ -f "$UNIT_DIR/$u" ]] && report "$UNIT_DIR/$u" ok || report "$UNIT_DIR/$u missing" no
  done
  if [[ -s "$PASSFILE" && "$(stat -c '%a' "$PASSFILE")" == 600 ]]; then
    report "$PASSFILE (mode 0600)" ok
  else
    report "$PASSFILE missing or mode != 0600" no
  fi
  if command -v systemctl >/dev/null; then
    [[ "$(systemctl --user is-enabled vaultctl-db-backup.timer 2>/dev/null)" == enabled ]] \
      && report "vaultctl-db-backup.timer enabled" ok \
      || report "vaultctl-db-backup.timer not enabled" no
  fi
  exit "$rc"
fi

# -------------------------------------------------------------- install mode
[[ "$(uname -s)" == Linux ]] || die "the backup timer targets Linux/user systemd"
check_deps

if (( DRY_RUN )); then
  say "dry run — nothing will be changed"
  say "  binary  : $BIN_DIR/vaultctl-db-backup"
  say "  units   : $UNIT_DIR/${UNIT_NAMES[*]}"
  say "  passphrase: $PASSFILE (generated if absent; never printed)"
  say "  repo    : $BACKUP_REPO (verified private at backup time)"
  [[ -f "$PASSFILE" ]] && say "  passphrase: already present" || say "  passphrase: would be generated"
  say "  timer   : $([[ $NO_ENABLE == 1 ]] && echo 'installed only' || echo 'enable --now')"
fi

say "installing backup artifacts"
run mkdir -p -- "$BIN_DIR" "$UNIT_DIR" "$CFG_DIR"
install_artifact 'vaultctl-db-backup' "$BIN_DIR/vaultctl-db-backup" 700
install_artifact 'systemd/vaultctl-db-backup.service' "$UNIT_DIR/vaultctl-db-backup.service" 644
install_artifact 'systemd/vaultctl-db-backup.timer' "$UNIT_DIR/vaultctl-db-backup.timer" 644

# Envelope passphrase: create once at mode 0600, report by path only.
if [[ ! -s "$PASSFILE" ]]; then
  say "generating envelope passphrase (not displayed)"
  if (( DRY_RUN )); then
    say "  [dry-run] generate $PASSFILE"
  else
    python3 - "$PASSFILE" <<'PY'
import os, secrets, sys
fd = os.open(sys.argv[1], os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
with os.fdopen(fd, 'w') as f:
    f.write(secrets.token_urlsafe(48) + '\n')
PY
  fi
fi
if (( ! DRY_RUN )); then
  chmod 600 -- "$PASSFILE"
  [[ "$(stat -c '%a' "$PASSFILE")" == 600 ]] || die "could not secure $PASSFILE"
fi

# Preflight the GitHub side (warn by default; the engine re-checks and fails closed).
gh_ok=1
if command -v gh >/dev/null; then
  if gh auth status >/dev/null 2>&1; then
    vis="$(gh repo view "$BACKUP_REPO" --json isPrivate -q '.isPrivate' 2>/dev/null || true)"
    case "$vis" in
      true)  say "github: authenticated; $BACKUP_REPO is private" ;;
      false) gh_ok=0; warn "$BACKUP_REPO is not private" ;;
      *)     gh_ok=0; warn "cannot read $BACKUP_REPO (create the private repo first)" ;;
    esac
  else
    gh_ok=0; warn "gh is not authenticated — run: gh auth login"
  fi
else
  gh_ok=0; warn "gh not found — install the GitHub CLI and authenticate"
fi
(( REQUIRE_GH && ! gh_ok )) && die "--require-gh: GitHub preflight failed"

if (( NO_ENABLE )); then
  say "skipping enable (--no-enable); arm later with:"
  say "  systemctl --user daemon-reload && systemctl --user enable --now vaultctl-db-backup.timer"
elif command -v systemctl >/dev/null; then
  run systemctl --user daemon-reload
  run systemctl --user enable --now vaultctl-db-backup.timer
  if (( ! DRY_RUN )); then
    say "timer: $(systemctl --user is-enabled vaultctl-db-backup.timer 2>/dev/null || echo unknown)"
    systemctl --user list-timers vaultctl-db-backup.timer --no-pager 2>/dev/null | sed -n '1,2p' || true
  fi
else
  warn "systemctl not found — timer files installed but not enabled"
fi

say "done. status: PATH=\"$BIN_DIR:\$PATH\" vaultctl-db-backup --help"
say "verify any time with: $0 --verify"
