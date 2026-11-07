#!/usr/bin/env python3
"""vaultctl mock vault — in-memory, no encryption, no password.

For dev only. Serves the same HTTP API shape as the real vaultctl vault so
consumers and `vaultctl` can point at it during development.

Usage:
    VAULT_PORT=9787 python mock_vault.py
    VAULT_URL=http://127.0.0.1:9787 vaultctl list
"""
import json
import os
import secrets
import time
from flask import Flask, jsonify, make_response, request

HOST = os.environ.get("VAULT_HOST", "127.0.0.1")
PORT = int(os.environ.get("VAULT_PORT", "9787"))

SEED = {
    "NOUS_API_KEY": "mock-nous-key",
    "OPENAI_API_KEY": "mock-openai-key",
    "EXAMPLE_SECRET": "hunter2",
}

app = Flask(__name__)
_entries = dict(SEED)
_sessions = {}


def _csrf():
    return secrets.token_hex(16)


@app.route("/api/status")
def status():
    return jsonify({"initialized": True, "unlocked": True})


@app.route("/api/unlock", methods=["POST"])
def unlock():
    token = secrets.token_hex(16)
    csrf = _csrf()
    _sessions[token] = csrf
    resp = make_response(jsonify({"ok": True, "csrf": csrf}))
    resp.set_cookie("sid", token, httponly=True, samesite="Lax")
    return resp


@app.route("/api/lock", methods=["POST"])
def lock():
    return jsonify({"ok": True})


@app.route("/api/entries")
def list_entries():
    return jsonify({"entries": sorted(_entries)})


@app.route("/api/entries/<path:name>", methods=["GET"])
def get_entry(name):
    if name not in _entries:
        return jsonify({"error": "not found"}), 404
    return jsonify({"value": _entries[name]})


@app.route("/api/entries", methods=["POST"])
def set_entry():
    data = request.get_json(silent=True) or {}
    name = data.get("name")
    value = data.get("value")
    if not name:
        return jsonify({"error": "name required"}), 400
    _entries[name] = value
    return jsonify({"ok": True, "name": name})


@app.route("/api/entries/<path:name>", methods=["DELETE"])
def delete_entry(name):
    _entries.pop(name, None)
    return jsonify({"ok": True})


@app.route("/api/change-password", methods=["POST"])
def change_password():
    return jsonify({"ok": True})


if __name__ == "__main__":
    print(f"vaultctl MOCK vault on http://{HOST}:{PORT}  (in-memory, no secrets)")
    app.run(host=HOST, port=PORT, debug=False, threaded=True)
