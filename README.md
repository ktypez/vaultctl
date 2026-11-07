<div align="center">

<p align="center"><img src="assets/logo.png" width="96" alt="Kinko 金庫"></p>

# 🔐 Kinko 金庫

**A zero-knowledge, password-gated credentials vault for your local network.**

Store API keys, tokens, and passwords encrypted at rest — AES-256-GCM,
PBKDF2-derived keys, nothing leaves your machine.

*Self-hosted · No accounts · No telemetry · One small Flask app*

</div>

---

## Repository layout

This is a monorepo with two components:

- `vaultctl/` — **credentials vault backend** (Flask API, encrypted store, `vaultctl` CLI). Service: `vaultctl.service` on `:8787`.
- `kinko/` — **web UI frontend** (Svelte + Vite SPA, served via `serve.py`). Service: `kinko.service` on `:8788` (proxies `/api/*` to vaultctl).

The frontend talks to the backend through relative `/api/*` (proxied in dev by Vite, in prod by `serve.py`).

---


You have secrets scattered across `.env` files, shell history, and browser
password managers. Cloud secret managers need accounts, billing, and trust in
someone else's infrastructure. **Kinko is the opposite:** a single Python file
that runs on your own box, encrypts everything with a key derived only from
your master password, and serves it through a clean web UI plus a tiny CLI.

- **Zero-knowledge by design** — the server never sees or stores your master
  password. Only a PBKDF2 salt and an encrypted verifier live on disk.
- **Encrypted at rest** — every value is sealed with AES-256-GCM using a fresh
  random nonce; the store file is useless without the password.
- **Session-scoped keys** — the derived key exists only in RAM while unlocked.
  Restarting the service wipes every session.
- **Two ways to use it** — a polished anime-styled web UI for humans, and
  `vaultctl` (or plain HTTP) for scripts and agents.
- **Dev-friendly** — ships with a mock server so you can build against the
  same API without touching real secrets.

## How it works

```
master password ──▶ PBKDF2-HMAC-SHA256 (600k iterations) ──▶ 32-byte key
key + value      ──▶ AES-256-GCM (random 12-byte nonce)   ──▶ credentials.enc (0600)
```

| Property        | Detail                                              |
| --------------- | --------------------------------------------------- |
| Key derivation  | PBKDF2-HMAC-SHA256, 600,000 iterations              |
| Encryption      | AES-256-GCM, per-value random nonce                 |
| Auth            | Single master password, never stored                |
| Sessions        | HttpOnly cookie + CSRF token, 30-min idle expiry    |
| Brute force     | Exponentially increasing delay after wrong attempts |
| Dependencies    | Python 3 + Flask + cryptography                     |

## Quick start

### Web UI

```bash
./run.sh --daemon          # starts the vault on http://127.0.0.1:8787
```

Open the browser, set a master password (≥ 8 chars), add entries. Done.

> ⚠️ If you lose the master password, the data is unrecoverable. That's not a
> bug — it's the threat model.

### CLI (`vaultctl`)

```bash
vaultctl list                      # all entry names
vaultctl get NOUS_API_KEY          # print a secret to stdout (trusted shells only)
vaultctl set MY_TOKEN "abc123"     # create/update
vaultctl delete MY_TOKEN           # remove
vaultctl ui                        # pretty terminal UI (OpenTUI) — search, copy, CRUD, audit
vaultctl status                    # initialized / unlocked?
vaultctl lock                      # wipe the session
vaultctl exec NOUS_API_KEY -- <cmd> [args...]   # inject into child env; value never printed (masked)
vaultctl audit [N]                 # last N audit records (JSONL, never values)
```

### Terminal UI (`vaultctl ui` / `vault-tui`)

A full-screen dashboard over the same API, in the Kinko visual style
(金 seal, monochrome + hanko-red accent). Built on **OpenTUI**
(`@opentui/core`, run with Bun):

- Live search (`/`), masked detail panel, clipboard copy via
  `wl-copy`/`xclip`/`xsel`/`pbcopy` (OSC52 fallback)
- Keyboard CRUD: `n` new, `e` edit, `d` delete (confirm dialog),
  `u`/`l` unlock/lock, `r` refresh, `a` audit log overlay, `q` quit
- Works anywhere: `vaultctl ui` or the `vault-tui` launcher on PATH.
  Install with `./install-tui.sh` (ensures bun, installs OpenTUI deps, writes
  the launcher — no sudo). `deploy.sh` runs it automatically.
  `vaultctl ui` falls back to the older Textual UI (`vault_tui.py`) if
  opentui/ or bun is missing.
  The TUI reuses the same session files (`~/.vault-password`, `~/.vault-cookie`,
  `~/.vault-csrf`) as the plain CLI. Values are only ever held in memory.

```bash
vaultctl ui                        # open the dashboard
vaultctl ui --check                # non-interactive backend self-test
vault-tui                          # same, via launcher
cd vaultctl && ./install-tui.sh    # (re)install bun + OpenTUI deps + launcher
```

Point tools at any vault instance with `VAULT_URL`:

```bash
NOUS_KEY=$(VAULT_URL=http://127.0.0.1:8787 vaultctl get NOUS_API_KEY)
```

### Safe consumption with `vaultctl exec`

`vaultctl get` prints the plaintext to stdout — fine for a trusted shell, but
the value then lands in your agent's context, the transcript, and any log that
captures command output. **`vaultctl exec` is the safer path:** it resolves the
value internally, injects it straight into the child process's environment, and
masks every occurrence (plus any 6+ character fragment) in the child's stdout
and stderr. The secret never reaches the caller's output.

```bash
# pass the key to a command without ever printing it
vaultctl exec NOUS_API_KEY -- my-tool --auth

# override the injected env var name (default: the entry name)
vaultctl exec NOUS_API_KEY --env AUTH_TOKEN -- my-tool --auth

# inspect only what you need, value stays hidden
vaultctl exec NOUS_API_KEY -- python3 -c 'import os; print(len(os.environ["NOUS_API_KEY"]))'
```

The child's exit code is propagated, so `exec` composes like a normal command.
Prefer `exec` over `get` whenever a secret is consumed by a program rather than
read by a human.

### Audit trail

Every unlock, list, get, set, and delete is appended to
`data/audit.jsonl` (mode `0600`) as one JSON object per line — **never the
secret value**:

```json
{"ts":"2026-10-28T04:48:56Z","action":"get","name":"NOUS_API_KEY","decision":"allow","reason":"ok","remote":"127.0.0.1"}
```

```bash
vaultctl audit 50                  # last 50 records
curl -s -b ~/.vault-cookie \
  http://127.0.0.1:8787/api/audit?n=50
```

The log records the *decision* (allow/deny) and *reason* (e.g. `locked`,
`not_found`, `bad_password`), so you can spot access attempts without ever
seeing the secrets behind them.

### HTTP API

Base URL `http://127.0.0.1:8787`. Mutations require `X-CSRF-Token` (returned
by `/api/unlock`).

| Endpoint                   | Method       | Purpose             |
| -------------------------- | ------------ | ------------------- |
| `/api/status`              | GET          | init / unlock state |
| `/api/setup`               | POST         | first-time password |
| `/api/unlock`              | POST         | open session + CSRF |
| `/api/lock`                | POST         | close session       |
| `/api/entries`             | GET          | list names          |
| `/api/entries/<name>`      | GET          | read one value      |
| `/api/entries`             | POST         | create / update     |
| `/api/entries/<name>`      | DELETE       | remove one          |
| `/api/change-password`     | POST         | rotate master pw    |
| `/api/audit`               | GET          | last N audit records (`?n=N`) |

## Development & mock mode

This repository is the **dev tree**, not the live service. To develop against
the vault without ever touching real secrets, run the built-in mock:

```bash
./run.sh                     # mock vault (default) — in-memory, any password unlocks
VAULT_PORT=9787 ./run.sh     # mock on another port
VAULT_REAL=1 ./run.sh        # run the real encrypted vault instead
```

The mock (`mock_vault.py`) implements the exact same HTTP surface with seeded
fake entries — swap `VAULT_URL` and your consumers keep working.

## Configuration

All optional, via environment variables:

| Variable             | Default      | Meaning                          |
| -------------------- | ------------ | -------------------------------- |
| `VAULT_HOST`         | `127.0.0.1`  | bind address                     |
| `VAULT_PORT`         | `8787`       | port                             |
| `VAULT_SESSION_TTL`  | `1800`       | session idle timeout (seconds)   |
| `VAULT_ITERATIONS`   | `600000`     | PBKDF2 rounds                    |
| `VAULT_DATA_DIR`     | *(app dir)*  | where `credentials.enc` lives    |

## Deployment (systemd)

Kinko separates **source** from **runtime**: edit this repo, deploy a clean
copy to a service root, and let systemd manage it.

```bash
./deploy.sh                  # rsyncs source → ~/.local/share/kinko (excl. .git, venv cache, data)
sudo systemctl restart kinko # then manage as usual
```

The installed unit runs from the service root with `VAULT_DATA_DIR` pointed at
its own `data/` directory, so the production service never executes out of a
dev checkout.

```bash
sudo systemctl status kinko
sudo systemctl stop kinko
journalctl -u kinko -f
```

## Security notes

- **Plain HTTP.** The default deployment speaks cleartext. On a trusted LAN
  that's often fine; anywhere else, front it with a TLS reverse proxy or reach
  it over Tailscale/VPN. Bind to `127.0.0.1` if you don't need LAN access.
- **File permissions.** `credentials.enc` and `sessions.json` are written
  `0600`; keep them that way.
- **Backups.** `credentials.enc` is fully encrypted — safe to copy into any
  backup system. Restore = drop the file back and restart. Lose both the file
  *and* the password and there is no recovery path.
- **Sessions are in-memory.** A service restart logs everyone out by design.
- **Audit log.** `data/audit.jsonl` records every access decision (allow/deny +
  reason) but never secret values. Inspect with `vaultctl audit` or
  `/api/audit?n=N`. Rotate/truncate it manually if it grows.
- **Safe consumption.** Prefer `vaultctl exec` over `vaultctl get` so secrets
  never enter a shell's stdout, an agent's context, or a transcript. `get`
  exists for trusted interactive use only.

## Repository layout

```
kinko/
├── app.py              # the vault: Flask service + embedded UI (~1,300 lines)
├── mock_vault.py       # dev-only mock, identical API shape
├── vaultctl            # CLI client (audit, exec, get, set, …)
├── vault_tui.py        # legacy terminal UI (Textual) — fallback for vaultctl ui
├── install-tui.sh      # installs bun + OpenTUI deps + the vault-tui launcher (no sudo)
├── opentui/            # OpenTUI front-end (bun + @opentui/core)
│   └── src/kinko_tui.ts     # main TUI — vaultctl ui / vault-tui
├── run.sh / stop.sh    # manual launchers (mock by default)
├── deploy.sh           # source → service-root sync (+ installs vaultctl)
├── install.sh          # installs the systemd unit
└── systemd/
    └── kinko.service   # production unit template
```

## FAQ

**Can I recover a forgotten master password?**
No. Zero-knowledge means no backdoor, no reset email, no support ticket.

**Is the web UI required?**
No — `vaultctl` and the HTTP API cover everything headlessly.

**What happens if someone steals `credentials.enc`?**
They face 600k PBKDF2 iterations per guess, offline, with nothing else to
leak. Use a strong password and sleep well.

**Why "Kinko"?**
金庫 (*kinko*) — Japanese for "safe/vault." The UI matches the vibe. 🌸

## License

MIT
