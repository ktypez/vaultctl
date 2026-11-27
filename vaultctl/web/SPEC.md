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
- Appearance is driven by `<html data-theme="light|dark" data-palette="mochi|matcha|soda|sakura">`, painted before first paint by `index.html` and kept in sync by `src/lib/design/theme.js`. The mode cycles auto → light → dark and is persisted under `localStorage['mochi.theme']`; the palette under `localStorage['mochi.palette']`.
- Toast messages report action outcomes.

## Visual system

Use the locally-owned pastel "mochi" system vendored from DATA Ledger / Truck: `src/lib/design/tokens.css` (paper/ink/line ramp, four HSL palettes, radii, `--shadow-pop`), `ui.css` (card, eyebrow, pill buttons, chips, badges, progress, mono marks), `fonts.css` + `public/fonts` (Itim / Mali / Prompt, thai + latin), and `theme.js` (auto/light/dark plus mochi/matcha/soda/sakura). `src/app.css` adds vault-specific layout and dashboard-equivalent AA contrast guardrails; there is no shared UI dependency, no gradient decoration and no external font request. Surfaces are pastel paper cards with 1px lines and soft offset shadows, rather than the old sky canvas / cream window shell with ink outlines and hard offset shadows.

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
