# vaultctl-web — Svelte Web UI for vaultctl (optional)

A pure SPA (Svelte + Vite) front-end for the vaultctl credentials vault. Optional — not required; vaultctl CLI / TUI cover everything headlessly.

## Development

```bash
npm install
npm run dev
```

Vite dev server runs on http://127.0.0.1:5173 and proxies `/api/*` to the
vault backend (default http://127.0.0.1:8787). Configure the backend URL via
`VITE_VAULT_URL`.

Make sure the vaultctl backend is running on port 8787 (or point
`VITE_VAULT_URL` at it).

## Build

```bash
npm run build
```

Output goes to `dist/`.

## Run (production)

```bash
python3 serve.py
```

Serves `dist/` on `PORT` (default 8788) and proxies `/api/*` to `VAULT_URL`
(default http://127.0.0.1:8787).

## Environment

- `VITE_VAULT_URL` — base URL for the vault API (build-time, Vite).
- `PORT` — frontend listen port for `serve.py` (default 8788).
- `VAULT_URL` — backend URL proxied by `serve.py` (default http://127.0.0.1:8787).

## Theme

Light/dark theme persisted in `localStorage` and toggled via `data-theme` on
`<html>`. Google Fonts (Fraunces, Inter, JetBrains Mono) loaded from
`index.html`.