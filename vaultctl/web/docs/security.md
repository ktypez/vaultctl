# Security notes

`vaultctl/web` is a frontend for the local `vaultctl` credentials service. The frontend server is a static-file server plus a restricted `/api/*` reverse proxy; it does not encrypt or store secrets itself. The backend is responsible for authentication, encryption, authorization, CSRF validation, and audit logging.

## Backend security model

According to `vaultctl/app.py`, the backend:

- Derives a 32-byte key with PBKDF2-HMAC-SHA256 (600,000 iterations by default) using a per-vault salt.
- Encrypts the verifier and each entry with AES-256-GCM and fresh nonces.
- Does not store the master password; derived keys live only in process memory and are never persisted.
- Uses an opaque `sid` session cookie marked `HttpOnly` and `SameSite=Lax`; `Secure` is enabled with `VAULT_COOKIE_SECURE=1`.
- Requires `X-CSRF-Token` for state-changing API operations.
- Adds increasing delays after repeated failed unlock attempts.
- Writes audit events without secret values.
- Stores `credentials.enc` and `audit.jsonl` under `VAULT_DATA_DIR` (default: backend directory); sensitive files are written with restrictive `0600` permissions where supported.

Backend defaults, including KDF iterations and session TTLs, can be configured with `VAULT_*` environment variables. Refer to the backend source and its deployment configuration for authoritative values.

## Frontend handling

- Values are masked by default and fetched only when the user reveals an entry.
- Values and reveal state are held in frontend memory and cleared when the UI locks the vault.
- Copy is disabled until a value has been revealed.
- The CSRF token is held in memory and browser `sessionStorage`; the master password is not saved by this frontend.
- Sessions are not persistent: a browser session cookie is backed by an in-memory server session and service restart invalidates all sessions.
- The light/dark theme preference is stored in `localStorage` and contains no vault data.

Avoid leaving the vault unlocked or revealed on a shared screen. Lock it when finished.

## Network and deployment precautions

`serve.py` binds to `127.0.0.1` by default. Set `HOST=0.0.0.0` only when you intentionally need network binding and have firewall/TLS controls. Do not expose the service directly to an untrusted network. Prefer loopback/private-network access or place it behind a suitably secured reverse proxy and firewall. Keep the API backend private as well.

For HTTPS deployments set `VAULT_COOKIE_SECURE=1` on the backend service. The reverse proxy must preserve same-origin `/api/*` routing. Same-origin proxying is recommended; setting `VITE_VAULT_URL` to a separate origin requires correct credentialed CORS and cookie configuration.

The frontend and backend are separate services. Protect their host, data directory, backups, logs, and administrative access. Do not commit vault data, session files, passwords, CSRF tokens, or secret values.

## Scope

These notes describe the checked-in implementation and are not a security audit or guarantee. Review the backend, reverse proxy, cookie behavior, firewall, and operational access controls before changing the deployment boundary.
