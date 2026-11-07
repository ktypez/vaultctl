#!/usr/bin/env python3
"""Serve the built vaultctl-web frontend (dist/) and proxy /api/* to the vault backend."""
import http.server
import os
import socket
import threading
import urllib.error
import urllib.request
from http import HTTPStatus
from pathlib import Path

PORT = int(os.environ.get("PORT", "8788"))
VAULT_URL = os.environ.get("VAULT_URL", "http://127.0.0.1:8787").rstrip("/")
DIST = Path(__file__).resolve().parent / "dist"

HOP_BY_HOP = {
    "connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
    "te", "trailers", "transfer-encoding", "upgrade", "content-length", "host",
}


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(DIST), **kwargs)

    def do_GET(self):
        if self.path.startswith("/api"):
            self._proxy("GET")
        else:
            super().do_GET()

    def do_POST(self):
        self._proxy("POST")

    def do_PUT(self):
        self._proxy("PUT")

    def do_DELETE(self):
        self._proxy("DELETE")

    def do_PATCH(self):
        self._proxy("PATCH")

    def do_OPTIONS(self):
        self._proxy("OPTIONS")

    def _proxy(self, method):
        length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(length) if length > 0 else None

        url = VAULT_URL + self.path
        req_headers = {}
        for k, v in self.headers.items():
            if k.lower() in HOP_BY_HOP:
                continue
            req_headers[k] = v

        req = urllib.request.Request(url, data=body, headers=req_headers, method=method)
        try:
            with urllib.request.urlopen(req) as resp:
                self._send_proxy_response(resp.status, resp.headers, resp.read())
        except urllib.error.HTTPError as e:
            self._send_proxy_response(e.code, e.headers, e.read())
        except Exception as e:
            payload = ("proxy error: " + str(e)).encode()
            self.send_response(HTTPStatus.BAD_GATEWAY)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

    def _send_proxy_response(self, status, headers, body):
        self.send_response(status)
        for k, v in headers.items():
            if k.lower() in HOP_BY_HOP:
                continue
            self.send_header(k, v)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, fmt, *args):
        # quiet-ish
        pass


def main():
    if not DIST.exists():
        raise SystemExit(f"dist/ not found at {DIST}; run npm run build first")
    httpd = http.server.ThreadingHTTPServer(("0.0.0.0", PORT), Handler)
    print(f"vaultweb frontend on http://0.0.0.0:{PORT} (proxy /api -> {VAULT_URL})")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()