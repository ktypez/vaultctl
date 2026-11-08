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

HOST = os.environ.get("HOST", "127.0.0.1")
PORT = int(os.environ.get("PORT", "8788"))
MAX_BODY_BYTES = int(os.environ.get("MAX_BODY_BYTES", str(1024 * 1024)))
PROXY_TIMEOUT = float(os.environ.get("PROXY_TIMEOUT", "10"))
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
        # This is a same-origin API proxy, not a general-purpose HTTP proxy.
        if not (self.path == "/api" or self.path.startswith("/api/")):
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        try:
            length = int(self.headers.get("Content-Length", "0") or "0")
        except ValueError:
            self.send_error(HTTPStatus.BAD_REQUEST)
            return
        if length < 0 or length > MAX_BODY_BYTES:
            self.send_error(HTTPStatus.REQUEST_ENTITY_TOO_LARGE)
            return
        body = self.rfile.read(length) if length > 0 else None

        url = VAULT_URL + self.path
        req_headers = {}
        for k, v in self.headers.items():
            if k.lower() in HOP_BY_HOP:
                continue
            req_headers[k] = v

        req = urllib.request.Request(url, data=body, headers=req_headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=PROXY_TIMEOUT) as resp:
                self._send_proxy_response(resp.status, resp.headers, resp.read())
        except urllib.error.HTTPError as e:
            self._send_proxy_response(e.code, e.headers, e.read())
        except Exception as e:
            payload = b"upstream unavailable"
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

    def end_headers(self):
        self.send_header("X-Content-Type-Options", "nosniff")
        self.send_header("X-Frame-Options", "DENY")
        self.send_header("Referrer-Policy", "no-referrer")
        self.send_header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
        self.send_header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
        super().end_headers()

    def log_message(self, fmt, *args):
        # Do not log request paths or headers; API paths can contain identifiers.
        pass


def main():
    if not DIST.exists():
        raise SystemExit(f"dist/ not found at {DIST}; run npm run build first")
    httpd = http.server.ThreadingHTTPServer((HOST, PORT), Handler)
    httpd.daemon_threads = True
    print(f"vaultctl web on http://{HOST}:{PORT} (proxy /api -> {VAULT_URL})")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass


if __name__ == "__main__":
    main()