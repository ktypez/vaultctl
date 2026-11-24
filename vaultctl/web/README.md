# vaultctl web

A Svelte 5 single-page web interface for the `vaultctl` encrypted credentials vault. The UI is optional; the vault backend and CLI can run without it.

The interface includes master-password setup and unlock, masked/revealable secrets, search, copy and delete, password management, an audit log, and light/dark themes. Its visual system follows the shared MCKY UI V6 "Work Shell" package (`@mcky/ui`, pinned to commit `d208f003f106ed33c714c9acd0ca6fdac76c6b99`): canonical tokens, theme, motion and primitives via `@mcky/ui/v6-base.css`, plus the shared animated `<Scenery />` backdrop — three clouds, three ridges and two trees by day; twelve stars, a crescent and ten lit city buildings at night. The cream window shell keeps ink outlines and hard offset shadows, yellow primary actions, and Space Grotesk / Sarabun / DM Mono, all self-hosted under `public/fonts` (the strict same-origin CSP forbids webfont CDNs). The mode is persisted under `localStorage['mcky-v6-mode']`; an older `localStorage['theme']` value is migrated on first load.

## Requirements

- Node.js and npm (for development/build). The pinned `@mcky/ui` tarball needs remote npm fetches enabled; the local `.npmrc` sets `allow-remote=all` so plain `npm ci` works.
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
