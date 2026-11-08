# HTTP API and sessions

`vaultctl/web` is a browser client for the `vaultctl` API. By default, API calls use same-origin `/api/*` URLs. In development Vite proxies them to `http://127.0.0.1:8787`; in production `serve.py` proxies them to `VAULT_URL`.

All API responses are JSON. Error responses commonly include an `error` string. Exact authorization and validation errors are returned by the backend.

## Endpoints used by the web UI

| Method | Path | Request | Successful response / behavior |
|---|---|---|---|
| `GET` | `/api/status` | — | `{ "initialized": boolean, "unlocked": boolean }` |
| `POST` | `/api/setup` | `{ "password": string, "confirm": string }` | `{ "ok": true, "csrf": string }`; creates the vault and an authenticated session |
| `POST` | `/api/unlock` | `{ "password": string, "remember": boolean }` | `{ "ok": true, "csrf": string }`; creates a session |
| `POST` | `/api/lock` | — | `{ "ok": true }`; destroys the current session |
| `GET` | `/api/entries` | — | `{ "entries": string[] }`; names are sorted |
| `GET` | `/api/entries/:name` | — | `{ "name": string, "value": string }`; returns a decrypted value |
| `POST` | `/api/entries` | `{ "name": string, "value": string }` | `{ "ok": true, "name": string }`; creates or replaces an entry |
| `DELETE` | `/api/entries/:name` | — | `{ "ok": true }` |
| `POST` | `/api/change-password` | `{ "current": string, "new": string, "confirm": string }` | `{ "ok": true }`; re-encrypts the vault with the new password |
| `GET` | `/api/audit?n=100` | — | `{ "entries": AuditRecord[], "total": number }`; UI requests the last 100 records |

URL-encode entry names when constructing `:name` paths. Read endpoints require a valid session. State-changing endpoints require both the session cookie and CSRF header.

## Authentication and CSRF flow

1. Call `GET /api/status`.
2. If `initialized` is false, send the setup form to `POST /api/setup`. Otherwise, if `unlocked` is false, send the unlock form to `POST /api/unlock`.
3. Setup/unlock return a session cookie (`sid`, `HttpOnly`, `SameSite=Lax`) and a `csrf` token in the JSON body.
4. Store the CSRF token only for the current browser session and send it in `X-CSRF-Token` on state-changing requests (`POST`, `PUT`, `PATCH`, and `DELETE`). The UI helper uses `credentials: 'include'` so the browser sends the cookie.
5. On lock, clear the client-side token and sensitive values from UI state.

The CSRF token is stored in memory and `sessionStorage` under `vaultctl_csrf` to support page reloads within the browser session. It is not the vault password or secret data. The backend can return `401` for a missing/expired session and `403` for an invalid CSRF token.

## Audit records

Audit entries include event time, action, key name, allow/deny decision, reason, and where available the remote address. The backend explicitly avoids writing secret values to the audit log. The UI displays the records returned by the API and does not expose secret values through audit requests.
