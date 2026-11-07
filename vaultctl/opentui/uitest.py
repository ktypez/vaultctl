#!/usr/bin/env python3
"""Drive the OpenTUI kinko TUI inside a pty and capture each step.

Only a dummy entry (TUI_OT_TEST) is ever revealed, so no real secret is
expected to appear in the captured output.
"""
import os
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from ptyharness import run  # noqa: E402  (opentui/ptyharness.py)

VAULTCTL = os.path.expanduser("~/.local/bin/vaultctl")
SCRIPT = "src/kinko_tui.ts"


def cli(*args):
    return subprocess.run([VAULTCTL, *args], capture_output=True, text=True)


def show(title, out):
    print("\n" + "=" * 78)
    print(f"## {title}")
    print("=" * 78)
    print(out.strip()[-2600:])


def main():
    cli("set", "TUI_OT_TEST", "dummy-value-1234")
    keys = [
        ("initial screen", []),
        ("after j (move down)", ["j"]),
        ("open search", ["/"]),
        ("typed filter TUI_OT", list("TUI_OT_TEST")),
        ("enter (focus list)", ["\r"]),
        ("reveal dummy value", ["v"]),
        ("audit modal", ["a"]),
        ("close audit", ["\x1b"]),
        ("new entry modal", ["n"]),
        ("close new modal", ["\x1b"]),
        ("delete confirm modal", ["d"]),
        ("cancel delete", ["n"]),
        ("quit", ["q"]),
    ]
    # Each step gets its own capture; keys are sent in order in one session.
    payload = []
    for _, ks in keys:
        payload.extend(ks)
    out, rc = run(SCRIPT, payload, wait=3.0, after=2.0)
    show(f"full session (exit {rc})", out)

    if "TUI_OT_TEST" not in out:
        print("\n!! WARN: dummy entry name never appeared in the capture")
    if "dummy-value-1234" in out:
        print("\nOK: reveal showed the dummy value (v works)")
    else:
        print("\n!! WARN: reveal did not show the dummy value")
    if rc not in (0, "still-running(then killed)"):
        print(f"\n!! exit code {rc}")
    cli("delete", "TUI_OT_TEST")


if __name__ == "__main__":
    main()
