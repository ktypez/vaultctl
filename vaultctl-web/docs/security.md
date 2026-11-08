# Security notes

`vaultctl-web` is a frontend for the local `vaultctl` credentials service. The frontend server is a static-file server plus a reverse proxy; it does not encrypt or store secrets itself. The backend is responsible for authentication, encryption, authorization, CSRF validation, and audit logging.

## Backend security model

According to `vaultctl/app.py`, the backend:

- Derives a 32-byte key with PBKDF2-HMAC-SHA256 (600,000 iterations by default) using a per-vault salt.
- Encrypts the verifier and each entry with AES-256-GCM and fresh nonces.
- Does not store the master password; the derived key is held in an authenticated server session.
- Uses an opaque `sid` cookie marked `HttpOnly` and `SameSite=Lax`.
- Requires `X-CSRF-Token` for state-changing API operations.
- Adds increasing delays after repeated failed unlock attempts.
- Writes audit events without secret values.
- Stores `credentials.enc`, `sessions.json`, and `audit.jsonl` under `VAULT_DATA_DIR` (default: backend directory); sensitive files are written with restrictive `0600` permissions where supported.

Backend defaults, including KDF iterations and session TTLs, can be configured with `VAULT_*` environment variables. Refer to the backend source and its deployment configuration for authoritative values.

## Frontend handling

- Values are masked by default and fetched only when the user reveals an entry.
- Values and reveal state are held in frontend memory and cleared when the UI locks the vault.
- Copy is disabled until a value has been revealed.
- The CSRF token is held in memory and browser `sessionStorage`; the master password is not saved by this frontend.
- The optional “Remember me” behavior is implemented by the backend session cookie (30 days by default); it is not a browser password-store feature.
- The light/dark theme preference is stored in `localStorage` and contains no vault data.

Avoid leaving the vault unlocked or revealed on a shared screen. Lock it when finished.

## Network and deployment precautions

`serve.py` binds to `0.0.0.0` by default, which exposes its port on every network interface unless restricted externally. Do not expose the service directly to an untrusted network. Prefer loopback/private-network access or place it behind a suitably secured reverse proxy and firewall. Keep the API backend private as well.

The current backend cookie configuration uses `secure=False`; use an HTTPS-terminating trusted proxy and review cookie settings before exposing the app beyond a trusted local network. Same-origin proxying is recommended; setting `VITE_VAULT_URL` to a separate origin requires correct credentialed CORS and cookie configuration.

The frontend and backend are separate services. Protect their host, data directory, backups, logs, and administrative access. Do not commit vault data, session files, passwords, CSRF tokens, or secret values.

## Scope

These notes describe the checked-in implementation and are not a security audit or guarantee. Review the backend, reverse proxy, cookie behavior, firewall, and operational access controls before changing the deployment boundary.
