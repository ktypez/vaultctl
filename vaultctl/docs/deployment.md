# Deployment

vaultctl separates **source** from **runtime**: edit this repo, deploy a clean
copy to a service root, and let systemd manage it.

## Source vs. service root

- Source (this repo): `~/projects/vaultctl`
- Service root (default): `~/.local/share/vaultctl`
- Data lives in `$VAULTCTL_SERVICE_ROOT/data` and is **never** touched by
  `deploy.sh`.

## Deploy

```bash
./deploy.sh                  # rsyncs source → service root (excl. .git, data, cli build)
# then manage the service:
sudo systemctl restart vaultctl
```

`deploy.sh` also builds the Go CLI (`build-cli.sh`) and installs it onto
`~/.local/bin/vaultctl`, symlinking it into `~/.opencode/bin`, so a single
deploy updates both the service and the client. Build the binary alone with
`./build-cli.sh` (or `--install` to put it on PATH). A Go toolchain is needed
on the machine that runs `deploy.sh`.

## Service management

```bash
sudo systemctl status vaultctl
sudo systemctl stop vaultctl
sudo systemctl restart vaultctl
journalctl -u vaultctl -f
```

The installed unit runs from the service root with `VAULT_DATA_DIR` pointed at
its own `data/` directory, so the production service never executes out of a
dev checkout.

## Backups

`credentials.enc` is fully encrypted — safe to copy into any backup system.
Restore = drop the file back into `data/` and restart. Lose both the file
*and* the master password and there is no recovery path.

`audit.jsonl` is plain JSONL (no secrets) and can be backed up or rotated
freely.

For offsite protection there is an optional weekly job that wraps
`credentials.enc` in a second GPG AES-256 envelope and pushes it to a private
GitHub repository:

```bash
./vaultctl/install-backup.sh          # install engine + weekly user timer
./vaultctl/install-backup.sh --verify # check the installation
```

See [Backups](backup.md) for installation, operation, restore, and security.
