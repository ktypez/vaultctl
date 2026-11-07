---
name: vaultctl-backup
description: Use when backing up or restoring the vaultctl vault.
version: 0.1.0
author: mcky (ktypez), Hermes Agent
license: MIT
platforms: [linux]
metadata:
  hermes:
    tags: [vaultctl, backup, restore, gpg, systemd, github]
    related_skills: []
---

# vaultctl encrypted backup skill

Install, operate, and restore the optional weekly backup of the vaultctl
database. The job wraps the already-AES-256-GCM `credentials.enc` in a second
GPG AES-256 envelope and pushes it to a **private** GitHub repository, so a lost
host does not mean lost secrets. Artifacts live in `vaultctl/` (engine
`vaultctl-db-backup`, installer `install-backup.sh`, units under
`vaultctl/systemd/`).

It does not touch the vault itself, does not read or print the master password,
and never writes plaintext into the backup repository.

## When to Use

- installing, repairing, or uninstalling the weekly backup timer
- taking or verifying one encrypted snapshot on demand
- restoring `credentials.enc` from an encrypted snapshot
- transcript of the design/security model for the backup

Don't use for: retrieving vault secrets (use the `vaultctl` CLI / vault skill),
or for adding anything to the private backup repository by hand — only the
engine writes there, and only ciphertext.

## Prerequisites

- Linux with **user** systemd (`systemctl --user`), no root required.
- `gh` authenticated with repo scope, plus `gpg`, `git`, `flock`, `python3`,
  `curl`, `install`. Check: `terminal(command="command -v gh gpg git flock python3 curl install")`.
- A private destination repo. Default `ktypez/vaultctl-db-backup`; override with
  `VAULTCTL_BACKUP_REPO`. Create once if missing:
  `terminal(command="gh repo create ktypez/vaultctl-db-backup --private")`.
- The live database at `$VAULTCTL_DATA_DIR` (default
  `~/.local/share/vaultctl/data/credentials.enc`, mode `0600`).

## How to Run

```bash
# install engine + units, create the passphrase, arm the weekly timer
terminal(command="./vaultctl/install-backup.sh", timeout=120)

# preview / confirm / refresh by hand / inspect the schedule
terminal(command="./vaultctl/install-backup.sh --dry-run")
terminal(command="./vaultctl/install-backup.sh --verify")
terminal(command="systemctl --user start vaultctl-db-backup.service", timeout=180)
terminal(command="systemctl --user list-timers vaultctl-db-backup.timer")
```

Without a checkout, pipe the raw installer; it fetches the artifacts from GitHub:

```bash
curl -fsSL https://raw.githubusercontent.com/ktypez/vaultctl/master/vaultctl/install-backup.sh | bash
```

## Quick Reference

| Goal | Command |
| --- | --- |
| Install / repair | `./vaultctl/install-backup.sh` |
| Change nothing, show steps | `./vaultctl/install-backup.sh --dry-run` |
| Files only, no timer | `./vaultctl/install-backup.sh --no-enable` |
| Check an install | `./vaultctl/install-backup.sh --verify` |
| Fail if GitHub not ready | `./vaultctl/install-backup.sh --require-gh` |
| Run one backup now | `systemctl --user start vaultctl-db-backup.service` |
| Next scheduled run | `systemctl --user list-timers vaultctl-db-backup.timer` |
| Last run log | `journalctl --user -u vaultctl-db-backup.service -n 50` |

Env overrides: `VAULTCTL_BIN_DIR`, `VAULTCTL_USER_UNIT_DIR`,
`VAULTCTL_BACKUP_CONFIG_DIR`, `VAULTCTL_BACKUP_REPO`, `VAULTCTL_SRC_RAW_BASE`,
`VAULTCTL_DATA_DIR`, `VAULTCTL_BACKUP_RETAIN`.

## Procedure

1. **Preflight.** `gh auth status` succeeds and the default/overridden repo
   exists and is private. Completion: `gh repo view "$VAULTCTL_BACKUP_REPO" --json isPrivate -q .isPrivate` prints `true`.
2. **Install.** Run `./vaultctl/install-backup.sh`. Completion: `--verify` exits 0 and prints an `ok` line for the engine, both units, the `0600` passphrase file, and the enabled timer.
3. **Take one snapshot.** `systemctl --user start vaultctl-db-backup.service`.
   Completion: the log ends with `verified encrypted snapshot pushed (…; private repo)`.
4. **Confirm the schedule.** `systemctl --user list-timers vaultctl-db-backup.timer`
   shows a NEXT time (Sundays 02:00 local, `Persistent=true`).
5. **Restore (only when needed).** Stop the service, decrypt the snapshot with the
   local passphrase to a temp file, `install -m 600` it over the live database, start
   the service, then delete the temp file. Completion: `vaultctl status` reports the
   vault initialized and it unlocks with the master password.

## Pitfalls

- **Never print the key.** The envelope passphrase is written once at
  `~/.config/vaultctl-db-backup/encryption-passphrase` (mode `0600`). Don't `cat`
  it, don't copy it to the backup repo, and keep that config directory out of any
  other backup — otherwise the second encryption layer is pointless.
- **The passphrase alone is not enough.** A snapshot still needs the vault master
  password (the inner AES-256-GCM layer). Losing both keys is unrecoverable.
- **Privacy is checked every run.** The engine aborts unless GitHub reports
  `isPrivate: true`; a public repo fails closed and never receives data.
- **The installer warns on GitHub problems, it does not fix them.** Without
  `--require-gh` a missing repo or unauthenticated `gh` is a warning; the engine
  still refuses to push at run time. Use `--require-gh` in automation.
- **One run at a time.** A `flock` on
  `~/.local/state/vaultctl-db-backup/backup.lock` makes a concurrent run exit 1
  with "already running" — expected, not a failure.
- **Linux/GNU only.** `find -printf`, `stat -c`, and `systemd --user` are assumed.
- **Restore replaces a live file.** Stop `vaultctl` first, keep mode `0600`, and
  remove the decrypted temp file afterwards.
- **No commit/push of repo changes without being asked** — the backup repo is
  runtime state, this project repo is source.

## Verification

```bash
./vaultctl/install-backup.sh --verify                      # exit 0 == installed correctly
systemctl --user is-enabled vaultctl-db-backup.timer       # -> enabled
systemctl --user list-timers vaultctl-db-backup.timer      # NEXT is set
```

A real end-to-end proof needs GitHub access: run the service and confirm the log
line `verified encrypted snapshot pushed`. Offline, `--dry-run` and `--verify`
exercise the installer without touching the private repository.
