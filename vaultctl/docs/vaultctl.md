# `vaultctl` CLI reference

The CLI is a single static Go binary. It talks to the vault over HTTP; point it
at any instance with `VAULT_URL` (default `http://127.0.0.1:8787`).

```
vaultctl status                          # initialized / unlocked?
vaultctl list                            # all entry names
vaultctl get <name>                      # print a secret to stdout (trusted shells only)
vaultctl set <name> [value]              # create / update (omit value → masked prompt)
vaultctl delete <name>                   # remove (confirms on a TTY; --force to skip)
vaultctl copy <name>                     # copy to the clipboard via OSC52 (works over SSH)
vaultctl lock                            # wipe the session
vaultctl unlock                          # open a session (password from env/file)
vaultctl change-password [new]           # rotate the master password
vaultctl exec <name> [--env VAR] -- <cmd> [args...]   # inject into child env; value never printed
vaultctl audit [N]                       # last N audit records (JSONL, never values)
vaultctl ui                              # full-screen dashboard
```

Output adapts to its destination: on a TTY commands render a styled interface
(monochrome + hanko-red accent); when stdout is a pipe commands emit plain text
or JSON. `--json` forces machine-readable output on any command. The machine
contract matches the legacy CLI (`list` → one name per line, `status`/`set`/
`delete` → JSON, `audit` → JSONL).

## Authentication

The master password is read from `$VAULT_PASSWORD` or `~/.vault-password`
(mode `0600`). When neither is available and stdin is a terminal, commands
prompt for it with a masked input. Session state is cached in `~/.vault-cookie`
and `~/.vault-csrf` (mode `0600`) so repeated calls are fast; sessions expire
after 30 minutes of inactivity.

```bash
export VAULT_PASSWORD=...        # or create ~/.vault-password
vaultctl list
```

## `exec` — safe consumption

`vaultctl get` prints the plaintext to stdout — the value then lands in your
agent's context, the transcript, and any log that captures command output.
`vaultctl exec` is the safer path: it resolves the value internally, injects it
into the child process's environment, and masks every occurrence (plus any
6+ character fragment) in the child's stdout/stderr. The secret never reaches
the caller's output.

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

## `audit` — review access

```bash
vaultctl audit 50                  # last 50 records
vaultctl audit 1                   # just the most recent
```

Each line is a JSON object: `ts`, `action`, `name`, `decision`, `reason`,
`remote`. Values are never recorded.

## `ui` — terminal dashboard

`vaultctl ui` opens a full-screen Bubble Tea dashboard built into the same
binary (no Bun/Node). Keys: `a` add, `e` edit, `r` rename, `d` delete,
`enter` copy (OSC52), `/` filter, `n` audit view, `R` refresh, `?` help,
`q` quit. The clipboard copy works over SSH via OSC52.

```bash
vaultctl ui            # open the dashboard
vaultctl ui --check    # non-interactive backend self-test
```

## Building

The CLI is Go (Cobra + Bubble Tea + Lip Gloss + Huh):

```bash
cd vaultctl
./build-cli.sh              # → cli/vaultctl
./build-cli.sh --install    # → ~/.local/bin/vaultctl
cd cli && go test ./...     # unit tests
```

`deploy.sh` builds and installs the binary automatically.
