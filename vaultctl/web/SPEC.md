# vaultctl/web — current implementation specification

This document describes the current frontend. It supersedes the original scaffold prompt that used to be stored here. For setup and operations, start at [README.md](README.md).

## Purpose and architecture

- Optional Svelte 5 + Vite single-page application for the `vaultctl` encrypted credentials API.
- No SvelteKit, SSR, or separate frontend database.
- `src/App.svelte` owns setup/unlock, vault interactions, and hash-routed settings; `src/lib/api.js` centralizes HTTP, cookie credentials, and CSRF handling.
- `serve.py` serves Vite's `dist/` output and proxies `/api/*` to the backend. The backend is a separate service.
- Development and preview use the `/api` proxy configured in `vite.config.js`.

## User-facing behavior

- Startup checks `GET /api/status` and shows setup, unlock, or the unlocked vault.
- Initial master-password setup validates at least eight characters and a matching confirmation.
- Unlock offers an optional backend “Remember me” session.
- The vault lists sorted key names, filters by name and already-revealed values, masks values by default, reveals a value on demand, allows copying revealed values, and confirms deletes.
- Add/update sends a key/value pair to the backend.
- Lock clears frontend entry/value/reveal state and the CSRF token.
- `#/settings` provides password change, session lock, and the latest 100 audit events.
- Light/dark appearance preference is persisted in browser `localStorage`.
- Toast messages report action outcomes.

## Visual system

Use the DATA Ledger V5 Paper Pop direction in `src/app.css`: warm dotted-paper background, Sora and JetBrains Mono typography, soft rounded surfaces, hand-placed sticker-like accents, outlined controls with hard offset shadows, and responsive desktop/mobile layouts. Maintain the light/dark theme toggle.

## API contract

The UI uses the following backend endpoints:

- `GET /api/status`
- `POST /api/setup`
- `POST /api/unlock`
- `POST /api/lock`
- `GET /api/entries`
- `GET /api/entries/:name`
- `POST /api/entries`
- `DELETE /api/entries/:name`
- `POST /api/change-password`
- `GET /api/audit?n=100`

Requests include cookies (`credentials: 'include'`). State-changing requests send `X-CSRF-Token` from the token returned by setup/unlock. Full payloads and session flow are documented in [docs/api.md](docs/api.md).

## Configuration and delivery

- `PORT` and `VAULT_URL` configure `serve.py` at runtime.
- `VITE_VAULT_URL` configures an optional browser API base at build time; leave empty for same-origin proxying.
- Vite's development/preview proxy target is configured in `vite.config.js`.
- Build with `npm run build`; output goes to ignored `dist/`.
- `systemd/vaultweb.service` is an example unit with host-specific paths; review before installing.

## Verification

```sh
npm run build
git diff --check
```

See [docs/development.md](docs/development.md), [docs/deployment.md](docs/deployment.md), and [docs/security.md](docs/security.md) for practical instructions and limitations.
