# Production deployment

## Build and run manually

Build frontend assets from the project root:

```sh
npm ci
npm run build
```

Then start the static server and API proxy:

```sh
PORT=8788 VAULT_URL=http://127.0.0.1:8787 python3 serve.py
```

`serve.py` requires an existing `dist/` directory, serves it on `0.0.0.0:$PORT`, and forwards `/api/*` requests to `VAULT_URL`. Defaults are port `8788` and backend `http://127.0.0.1:8787`. The backend must be running and reachable. The browser should use same-origin API paths; leave `VITE_VAULT_URL` unset for this setup.

Open `http://127.0.0.1:8788/` on the host. Since the server binds to all interfaces, only access it remotely over a trusted private network or behind a reviewed HTTPS reverse proxy and firewall. See [security notes](security.md).

## systemd example

The repository includes `systemd/vaultweb.service`. It assumes:

- The checkout is at `/home/admin/projects/vaultctl/vaultctl-web`.
- The service account/group is `admin`.
- The API backend unit is named `vaultctl.service` and listens on `127.0.0.1:8787`.
- Built files already exist in `dist/`.

Review and adjust the unit for the target host before installing it. Example installation (requires administrator privileges):

```sh
sudo cp systemd/vaultweb.service /etc/systemd/system/vaultweb.service
sudo systemctl daemon-reload
sudo systemctl enable --now vaultweb.service
```

Check the unit and HTTP endpoints:

```sh
systemctl is-active vaultweb.service
curl -fsS -o /dev/null -w 'frontend %{http_code}\n' http://127.0.0.1:8788/
curl -fsS -o /dev/null -w 'API %{http_code}\n' http://127.0.0.1:8788/api/status
```

## Updating the UI

From the project directory:

```sh
npm ci
npm run build
sudo systemctl restart vaultweb.service
```

The Python service serves the files from `dist/` on each request; restarting makes sure the service process is current. Then verify the service and endpoints again. A successful frontend response does not by itself prove the backend is available; check `/api/status` too.

If systemd reports that the operation requires authentication, run the administrative command in an authorized interactive session. A build alone does not restart the running service.

## Logs and troubleshooting

```sh
systemctl status vaultweb.service
journalctl -u vaultweb.service -n 100 --no-pager
```

The example unit also appends stdout/stderr to `serve.log` in the project directory. Common issues:

- `dist/ not found`: run `npm run build`.
- Frontend responds but API proxy fails: confirm the backend is active and `VAULT_URL` points to the correct address.
- Wrong API target during development: update the proxy target in `vite.config.js`; production uses `VAULT_URL`.
- Port already in use: choose a different `PORT` and update the proxy/reverse-proxy configuration.
- Changes do not appear: rebuild the assets, verify the HTML references the new hashed assets, then reload without stale browser cache.
