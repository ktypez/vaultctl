# Encrypted database backups

`credentials.enc` is already AES-256-GCM ciphertext, so copying it anywhere is
safe. This page covers the **optional offsite backup**: a weekly job that wraps
the database in a second encryption layer and pushes it to a **private** GitHub
repository, so a lost host does not mean lost secrets.

```
credentials.enc                     AES-256-GCM   (vault master password)
  └─ GPG symmetric envelope         AES-256       (envelope passphrase, host-only)
       └─ credentials-<UTC>.gpg     git push      (private repo, newest 52 kept)
```

Two independent keys are required to read a snapshot: the **envelope
passphrase** (only on the host) and the **vault master password**. Losing both
the host and the passphrase leaves the snapshots unreadable by design.

## Install

Artifacts: `vaultctl/vaultctl-db-backup` (the engine) and
`vaultctl/systemd/vaultctl-db-backup.{service,timer}` (weekly user timer,
Sundays 02:00 local, `Persistent=true`, 15 min jitter).

One-liner — fetches the artifacts from the repository's default branch (`master`),
so no checkout is needed:

```bash
curl -fsSL https://raw.githubusercontent.com/ktypez/vaultctl/master/vaultctl/install-backup.sh | bash
```

Override `VAULTCTL_SRC_RAW_BASE` to install from another branch, tag, or fork.
From a checkout:

```bash
./vaultctl/install-backup.sh
```

Step by step, if you prefer to control each stage:

```bash
# 1. prerequisites — the installer refuses to run without these
gh auth login                        # GitHub CLI (never assumed); 'repo' scope needed
command -v gpg git flock python3     # gpg + git + flock + python3 on PATH
                                     # (nothing is auto-installed; apt-get install gnupg git util-linux python3)

# 2. the private destination repo (create once; the installer never writes it)
gh repo create ktypez/vaultctl-db-backup --private --description 'vaultctl encrypted backups'

# 3. install artifacts + arm the timer
./vaultctl/install-backup.sh

# 4. confirm
./vaultctl/install-backup.sh --verify
systemctl --user list-timers vaultctl-db-backup.timer
```

If `gh` is missing, unauthenticated, or the private repository does not exist
yet, the installer reports it explicitly (`--require-gh` makes it fatal instead
of a warning). It never prints or copies key material, and it never reads or
writes the backup repository's contents.

Installer flags: `--dry-run`, `--no-enable`, `--verify`, `--require-gh`.
Environment overrides: `VAULTCTL_BIN_DIR`, `VAULTCTL_USER_UNIT_DIR`,
`VAULTCTL_BACKUP_CONFIG_DIR` (default `~/.config/vaultctl-db-backup`),
`VAULTCTL_BACKUP_REPO` (default `ktypez/vaultctl-db-backup`),
`VAULTCTL_SRC_RAW_BASE`.

An agent can install and operate this end to end with the bundled skill
(`skills/vaultctl-backup/SKILL.md`) — see [Agent skill](#agent-skill) below.

## What the installer writes

| Path | Mode | Purpose |
| --- | --- | --- |
| `~/.local/bin/vaultctl-db-backup` | `0700` | backup engine |
| `~/.config/systemd/user/vaultctl-db-backup.{service,timer}` | `0644` | user units |
| `~/.config/vaultctl-db-backup/encryption-passphrase` | `0600` | GPG envelope key — generated once, **never printed** |
| `~/.local/state/vaultctl-db-backup/` | `0700` | run lock + temporary work dir |
| `~/.local/share/vaultctl-db-backup/repo/` | `0700` | clone of the private repo |

The installer prints paths, never key material. It does not touch the private
repository's contents; only the engine does, and only with ciphertext.

## Operation

```bash
systemctl --user list-timers vaultctl-db-backup.timer   # next scheduled run
systemctl --user start vaultctl-db-backup.service       # run one backup now
journalctl --user -u vaultctl-db-backup.service -n 50   # last run log
/usr/bin/env HOME="$HOME" ~/.local/bin/vaultctl-db-backup   # run in foreground
```

Each run is fail-closed. It aborts unless: the encrypted database exists with
mode `0600`; the remote is verified **private** via the GitHub API; the new
snapshot decrypts back to the exact source bytes (compared with `cmp`, nothing
displayed). A lock file (`flock`) prevents overlapping runs, and a successful
run keeps the newest 52 snapshots (`VAULTCTL_BACKUP_RETAIN` to change).

## Restore

Stop the service before touching the live database.

```bash
# 1. read a snapshot back with the local envelope passphrase (no output on success)
gpg --batch --pinentry-mode loopback \
    --passphrase-file ~/.config/vaultctl-db-backup/encryption-passphrase \
    --decrypt ~/.local/share/vaultctl-db-backup/repo/credentials-<UTC>.gpg \
  > /tmp/restore.enc

# 2. sanity-check it is the same shape as the live file
stat -c '%s' /tmp/restore.enc ~/.local/share/vaultctl/data/credentials.enc

# 3. drop it back in place and restart
sudo systemctl stop vaultctl
install -m 600 /tmp/restore.enc ~/.local/share/vaultctl/data/credentials.enc
sudo systemctl start vaultctl
shred -u /tmp/restore.enc 2>/dev/null || rm -f /tmp/restore.enc
```

The restored file is still AES-256-GCM encrypted; you unlock it with the
**vault master password** as usual. If the master password is gone too, the
snapshot is unrecoverable — that is the threat model, not a bug.

## Agent skill

`skills/vaultctl-backup/SKILL.md` is a portable agent skill (no machine-local
paths, no embedded secrets) covering install, operation, and restore of this
backup. Install it into an agent's skills directory to let that agent operate the
backup safely:

```bash
# Hermes Agent
cp -r skills/vaultctl-backup ~/.hermes/profiles/<profile>/skills/
# Claude Code / other SKILL.md agents
cp -r skills/vaultctl-backup ~/.claude/skills/
```

## Security notes

- **Two keys, two places.** The envelope passphrase lives only on the host; the
  snapshots live only in the private repo. Neither alone is enough.
- **Repository stays ciphertext-only.** The engine writes its own `.gitignore`
  (`*` except `.gitignore`, `README.md`, `credentials-*.gpg`) so vault session
  files, the passphrase, or plaintext exports can never be committed.
- **Private is verified per run.** The visibility check runs on every backup and
  fails closed if it cannot confirm `isPrivate: true`.
- **Nothing is echoed.** The passphrase file is written `0600` and never
  printed; `vaultctl` values are never part of a backup run.
- **Timer is user-scoped.** No root or system unit; `systemctl --user` only.
- **Keep `~/.config/vaultctl-db-backup` out of other backups** or the envelope
  key travels with the snapshots, defeating the second layer.
