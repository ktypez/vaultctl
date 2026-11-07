# vaultctl-web — Svelte Web UI for vaultctl (credentials vault, optional)

You are working in `/home/admin/projects/vaultctl/vaultctl-web/`, a freshly scaffolded Svelte + Vite app.
Build a complete web UI that talks to the Vaultctl backend (credentials vault) HTTP API.

## Backend API (vaultctl, default base URL http://127.0.0.1:8787)

All requests must use `credentials: 'include'` (cookies). State-changing requests (POST/PUT/DELETE) must send header `X-CSRF-Token: <csrf>`.

- GET  /api/status              -> { initialized: bool, unlocked: bool }
- POST /api/setup   { password, confirm } -> { ok, csrf }  (sets session cookie)
- POST /api/unlock  { password, remember } -> { ok, csrf }  (sets session cookie)
- POST /api/lock    (no body) -> { ok }
- GET  /api/entries -> { entries: [name,...] }   (sorted)
- GET  /api/entries/:name -> { name, value }
- POST /api/entries { name, value } -> { ok, name }
- DELETE /api/entries/:name -> { ok }
- POST /api/change-password { current, new, confirm } -> { ok }
- GET  /api/audit?n=100 -> { entries: [...], total }

CSRF token: returned by /api/setup and /api/unlock in the JSON body as `csrf`. Store it in memory (and sessionStorage) and send as X-CSRF-Token on state-changing calls. On lock, clear it.

Base URL configurable via Vite env `VITE_VAULT_URL` (default http://127.0.0.1:8787).

## Features to implement

1. App shell: header with logo "vaultctl web", theme toggle (light/dark, persist in localStorage), and (when unlocked) lock button + settings link.
2. Auth flow:
   - On load, call /api/status.
   - If not initialized: show Setup form (password + confirm, min 8 chars). On submit -> /api/setup, store csrf, go to vault.
   - If initialized but locked: show Unlock form (password, remember-me checkbox). On submit -> /api/unlock, store csrf, go to vault.
   - If unlocked: show vault.
3. Vault view:
   - Entries list with search/filter (by name and revealed value), count "X of Y entries".
   - Each entry row: name (key), value (masked •••••••• or revealed), actions: reveal/hide (👁/🙈), copy (⧉, disabled unless revealed), delete (✕ with confirm).
   - Reveal fetches /api/entries/:name, caches value, toggles revealed.
   - Empty state.
   - "＋ Add entry" button opens a modal: name + value inputs, submit -> POST /api/entries, then refresh list.
4. Settings view (route /settings): change password form (current/new/confirm), audit log list (last 100). Link back to vault.
5. Toast notifications (success/error) for actions.
6. Routing: use hash-based or simple state-based routing (no svelte-kit). Two views: 'vault' and 'settings'.

## Design

Port the visual style from the old template in /home/admin/projects/vaultctl/vaultctl/app.py (the TEMPLATE string). Extract the CSS variables and component styles and put them in src/app.css (global). Keep the washi/anime aesthetic: serif display font, mono for keys/values, the color palette (light + dark), card surfaces, grain overlay, fade-up animations.

Key CSS variables (from the template):
- Light: --bg #f4efe6, --bg2 #fbf8f2, --ink #1f1a14, --muted #6b6256, --line #d9d0c0, --accent #c8553d, --accent-soft rgba(200,85,61,.12), --ok #2f6b3a, --err #9b2c2c, --card #fffaf0
- Dark: --bg #13110e, --bg2 #1c1915, --ink #ece6d8, --muted #9a9184, --line #2e2a23, --accent #e07a5f, --accent-soft rgba(224,122,95,.14), --ok #7fb069, --err #e07070, --card #201d18
- Fonts: --serif 'Fraunces' (or fallback Georgia), --sans 'Inter' (system-ui fallback), --mono 'JetBrains Mono' (monospace fallback).
- Theme toggled via `data-theme` attribute on <html>.

Include Google Fonts link in index.html for Fraunces, Inter, JetBrains Mono.

## Tech constraints

- Svelte 5 (runes) is fine, but keep it simple. Use Svelte 4-style stores if easier. Vite + Svelte (already scaffolded).
- No SvelteKit. No SSR. Pure SPA.
- All API calls go through a small `api.js` helper that handles base URL, credentials, CSRF header, JSON.
- Build output to `dist/` (default Vite). The production server will serve `dist/`.

## Deliverables

- Working `npm run dev` (Vite dev server with proxy /api -> http://127.0.0.1:8787) and `npm run build`.
- All features above functional.
- A `serve.py` (Python, in project root) that serves `dist/` on PORT (default 8788) and proxies `/api/*` to VAULT_URL (default http://127.0.0.1:8787). Used for production.
- A `vaultweb.service` systemd unit example (optional; enable only if you want the web UI) in `systemd/` for the frontend service (port 8788).
- Update README.md with dev/build/run instructions.

Please implement fully. Test by running `npm run build` (must succeed) and `npm run dev` briefly. Do not break the existing vaultctl backend.

When done, print a summary of files created/changed.
SPEC.md