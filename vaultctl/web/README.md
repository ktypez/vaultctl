# vaultctl web

A Svelte 5 single-page web interface for the `vaultctl` encrypted credentials vault. The UI is optional; the vault backend and CLI can run without it.

The interface includes master-password setup and unlock, masked/revealable secrets, search, copy and delete, password management, an audit log, and an appearance picker. Its visual system is the locally-owned pastel "mochi" design from DATA Ledger / Truck: paper `#FFFDF8` with a dot grid, sticker cards with 1px lines and soft offset shadows, pill buttons/chips, pastel badges, and Itim / Mali / Prompt typography. Tokens, primitives, fonts and the theme controller live in `src/lib/design/` (no shared UI package and no external assets); the 12 self-hosted woff2 files are under `public/fonts` because the strict same-origin CSP forbids webfont CDNs. Four palettes (mochi / matcha / soda / sakura) and light / dark / auto mode are stored in `localStorage['mochi.palette']` and `localStorage['mochi.theme']`, with the attributes painted on `<html>` before first paint.

## Requirements

- Node.js and npm (for development/build). All dependencies come from the normal npm registry, so plain `npm ci` works.
- Python 3 (for the static server)
- A running `vaultctl` API, default `http://127.0.0.1:8787`

## Quick start

```sh
npm install
npm run dev
```

Open the Vite address shown in the terminal (normally `http://localhost:5173`). The Vite dev server proxies `/api/*` to `http://127.0.0.1:8787`; start the backend first. That proxy target is configured in `vite.config.js`.

Build the static site:

```sh
npm run build
```

Vite writes production assets to `dist/`.

## Run the production server

After building, run:

```sh
python3 serve.py
```

The server serves `dist/` on `127.0.0.1:8788` by default and proxies `/api/*` to the backend. Set `PORT` and `VAULT_URL` to change those values:

```sh
PORT=8788 VAULT_URL=http://127.0.0.1:8787 python3 serve.py
```

The server binds to loopback by default. Put it behind a trusted HTTPS reverse proxy or expose it only through a private VPN; see [deployment](docs/deployment.md). Set `HOST=0.0.0.0` only when you intentionally need LAN binding and have firewall/TLS controls. For the deployed same-origin setup, leave `VITE_VAULT_URL` unset so the browser sends API requests through `serve.py`.

## Configuration

| Variable | Used by | Default | Purpose |
|---|---|---|---|
| `HOST` | `serve.py` | `127.0.0.1` | Listen address |
| `PORT` | `serve.py` | `8788` | Frontend listen port |
| `VAULT_URL` | `serve.py` | `http://127.0.0.1:8787` | Backend target for `/api/*` proxy requests |
| `VITE_VAULT_URL` | Vite build | empty | Optional absolute API base URL in the browser. Leave empty when using the same-origin production proxy. |

The Vite dev/preview proxy target is set separately in `vite.config.js` and currently points to `http://127.0.0.1:8787`.

## Documentation

- [Development and project layout](docs/development.md)
- [HTTP API and session flow](docs/api.md)
- [Production deployment and systemd](docs/deployment.md)
- [Security notes and operational precautions](docs/security.md)

## Project status

The checked-in systemd example is `systemd/vaultweb.service`; it expects this repository at `/home/admin/projects/vaultctl/vaultctl/web` and serves port `8788`. Adjust paths, account, and backend URL before using it elsewhere. Build the frontend again after changing source files, then restart the service to load the new `dist/` assets.
