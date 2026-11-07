# Security & threat model

## What vaultctl protects against

- **Secrets at rest.** `credentials.enc` is AES-256-GCM ciphertext. The key is
  derived from your master password with PBKDF2-HMAC-SHA256 (600,000
  iterations). Stealing the file gives an attacker an offline password-guessing
  problem, not the secrets.
- **Master password never stored.** Only a salt and an encrypted verifier live
  on disk. The server holds the derived key only in RAM, only while unlocked.
- **Session scoping.** Sessions are HttpOnly cookies + CSRF tokens with a
  30-minute idle expiry. A service restart wipes every session.
- **Brute force.** Failed unlocks accumulate an exponentially increasing delay.
- **Accidental leakage.** `vaultctl exec` injects a secret into a child
  process's environment and masks the value (and any 6+ character fragment) in
  its stdout/stderr, so it never reaches an agent's context, a transcript, or a
  log. `vaultctl get` still prints plaintext and is intended for trusted shells
  only.
- **Auditability.** Every unlock, list, get, set, and delete is appended to
  `data/audit.jsonl` (mode `0600`) with the decision and reason — never the
  value — so access attempts are reviewable after the fact.

## What vaultctl does *not* protect against

- **Single-uid mode is not a boundary.** The broker (and `vaultctl`) run as
  your user. Anything else running as you — including an agent with a shell —
  can read `~/.vault-password`, the cookie files, or the vault file directly.
  vaultctl assumes the OS user is the trust boundary.
- **Output masking is not containment.** It byte-matches the resolved value in
  child output, so `base64`, `tr`, or `rev` walk straight through it. It
  prevents accidental echo, not a hostile child. Do not allow untrusted binaries
  to receive secrets.
- **Use is not safety.** An agent that can use a credential can do whatever
  that credential allows. vaultctl hides the *value*, not the *power*.
- **Static secrets.** There is no TTL or just-in-time issuance; a stored
  password is long-lived by nature.
- **Root / passwordless sudo.** Anyone with root can read process memory and
  the vault. No password manager solves that.

## Recommendations

1. Prefer `vaultctl exec` over `vaultctl get` for any programmatic consumption.
2. Front the service with TLS or reach it over Tailscale/VPN; the default
   deployment speaks cleartext HTTP.
3. Keep `credentials.enc`, `sessions.json`, `audit.jsonl`, and
   `~/.vault-password` at mode `0600`.
4. Back up `credentials.enc` (it is fully encrypted and safe to copy). Lose
   both the file and the master password and there is no recovery path.
5. Rotate the weak default password (`meowmeow`) with
   `vaultctl change-password <new>`.
6. Review `vaultctl audit` periodically and prune `audit.jsonl` if it grows.
