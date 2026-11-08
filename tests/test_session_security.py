"""Regression tests for vault session key handling.

Run with: python -m unittest discover -s tests
"""
import os
import tempfile
import unittest

from vaultctl import app as vault


class SessionSecurityTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.old = {
            "DATA_DIR": vault.DATA_DIR,
            "VAULT_PATH": vault.VAULT_PATH,
            "SESSIONS_FILE": vault.SESSIONS_FILE,
            "AUDIT_PATH": vault.AUDIT_PATH,
            "ITERATIONS": vault.ITERATIONS,
            "COOKIE_SECURE": vault.COOKIE_SECURE,
        }
        vault.DATA_DIR = self.tmp.name
        vault.VAULT_PATH = os.path.join(self.tmp.name, "credentials.enc")
        vault.SESSIONS_FILE = os.path.join(self.tmp.name, "sessions.json")
        vault.AUDIT_PATH = os.path.join(self.tmp.name, "audit.jsonl")
        vault.ITERATIONS = 1_000
        vault.COOKIE_SECURE = True
        vault._sessions.clear()
        self.client = vault.app.test_client()

    def tearDown(self):
        vault._sessions.clear()
        for name, value in self.old.items():
            setattr(vault, name, value)
        self.tmp.cleanup()

    def test_derived_session_key_is_never_persisted(self):
        response = self.client.post(
            "/api/setup",
            json={"password": "correct horse battery staple", "confirm": "correct horse battery staple"},
        )
        self.assertEqual(response.status_code, 200)
        self.assertIn("Secure", response.headers.get("Set-Cookie", ""))
        self.assertFalse(os.path.exists(vault.SESSIONS_FILE))

    def test_restart_invalidates_existing_session(self):
        response = self.client.post(
            "/api/setup",
            json={"password": "correct horse battery staple", "confirm": "correct horse battery staple"},
        )
        self.assertEqual(response.status_code, 200)
        self.assertTrue(vault._sessions)

        # A process restart starts with an empty in-memory session map.
        vault._sessions.clear()
        status = self.client.get("/api/status")
        self.assertEqual(status.status_code, 200)
        self.assertFalse(status.get_json()["unlocked"])


if __name__ == "__main__":
    unittest.main()
