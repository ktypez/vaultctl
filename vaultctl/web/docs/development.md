# Development

## Prerequisites

- Node.js/npm compatible with the versions required by the locked Vite and Svelte dependencies.
- Python 3 only if running `serve.py` locally.
- A running `vaultctl` API for end-to-end use. The Vite proxy defaults to `http://127.0.0.1:8787` (`vite.config.js`).

## Install and run

```sh
npm install
npm run dev
```

Open the local Vite URL printed in the terminal. API calls use `/api/*` and are proxied to the backend by Vite; the browser normally does not need a separately configured API origin.

Useful scripts from `package.json`:

- `npm run dev` — Vite development server.
- `npm run build` — production build to `dist/`.
- `npm run preview` — Vite preview server. Its `/api` proxy uses the same target as the dev server.

For production-like local serving, build first and then run `python3 serve.py` (default port `8788`). See [deployment](deployment.md).

## Configuration

`VITE_VAULT_URL` is read by `src/lib/api.js` and becomes a browser-side absolute API base at build time. It defaults to an empty string so same-origin `/api` paths work through the dev or production proxy. If setting it to a different origin, the backend must allow credentialed CORS and cookie policy must be configured for that origin; prefer the same-origin proxy.

The Vite dev/preview proxy target is configured in `vite.config.js` (default `http://127.0.0.1:8787`). The Python production proxy uses `VAULT_URL` at runtime, defaulting to the same backend address. `PORT` sets the production frontend listen port (default `8788`).

## Project layout

```text
index.html                 HTML shell, theme bootstrap, Sora/JetBrains Mono fonts
src/
  App.svelte                Auth flow, vault list, settings, interactions
  app.css                   Global UI styles and Paper Pop V5 theme
  main.js                   Svelte mount entry point
  lib/api.js                Fetch helper, cookies, CSRF token handling
vite.config.js              Svelte plugin and dev/preview API proxy
serve.py                    Static production server and API reverse proxy
systemd/vaultweb.service    Example systemd unit
```

## UI behavior

- The app calls `GET /api/status` on startup to choose setup, unlock, or vault state.
- Secrets stay masked until reveal. Revealed values are fetched on demand and kept in the current page's state; locking clears the in-memory entry and value state.
- Search matches entry names and values that have already been revealed.
- The Settings view is hash-routed at `#/settings`; it shows password-change controls and the latest 100 audit records.
- The selected light/dark theme is stored in browser `localStorage`.

## Verify changes

```sh
npm run build
git diff --check
```

`dist/` is generated output and is ignored by Git. Rebuild before restarting the production server so it serves the latest frontend.
