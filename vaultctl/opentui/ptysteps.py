#!/usr/bin/env python3
"""Run the OpenTUI TUI in a pty and render each step through pyte (real grid),
so layout/spacing can be inspected exactly as a terminal would show it.

Usage: python3 ptysteps.py <script.ts> '<key spec>'
Key spec: comma-separated tokens; \r = Enter, \x1b = Esc, ::SEP:: = snapshot marker.
"""
import os
import pty
import select
import subprocess
import sys
import time

import pyte

BUN = os.path.expanduser("~/.bun/bin/bun")
CWD = os.path.dirname(os.path.abspath(__file__))
COLS, ROWS = 120, 34


def start(script, cols=COLS, rows=ROWS):
    env = dict(os.environ)
    env.update({"TERM": "xterm-256color", "COLUMNS": str(cols), "LINES": str(rows)})
    m, s = pty.openpty()
    os.set_blocking(s, False)
    p = subprocess.Popen([BUN, script], cwd=CWD, stdin=s, stdout=s, stderr=s, env=env, close_fds=True)
    os.close(s)
    screen = pyte.Screen(cols, rows)
    stream = pyte.Stream(screen)
    return p, m, screen, stream


def pump(p, m, stream, sec):
    end = time.time() + sec
    while time.time() < end and p.poll() is None:
        r, _, _ = select.select([m], [], [], 0.15)
        if r:
            try:
                d = os.read(m, 65536)
            except OSError:
                return
            if not d:
                return
            stream.feed(d.decode("utf-8", "replace"))


def snapshot(screen, title):
    print("\n" + "=" * COLS)
    print(f"### {title}")
    print("=" * COLS)
    for line in screen.display:
        print(line.rstrip())


def main():
    script = sys.argv[1] if len(sys.argv) > 1 else "src/vaultctl-ui.ts"
    spec = sys.argv[2] if len(sys.argv) > 2 else ""
    steps = [s for s in spec.split(",")] if spec else []
    p, m, screen, stream = start(script)
    pump(p, m, stream, 3.0)
    snapshot(screen, "initial")
    for raw in steps:
        if not raw:
            continue
        token = raw
        label = token
        if token.startswith("TEXT:"):
            payload = token[5:]
            for ch in payload:
                if p.poll() is not None:
                    break
                os.write(m, ch.encode())
                pump(p, m, stream, 0.25)
            snapshot(screen, f"typed {payload!r}")
            continue
        mapping = {r"\r": "\r", r"\x1b": "\x1b", r"\t": "\t", r"^v": "\x16", r"\x7f": "\x7f"}
        payload = mapping.get(token, token)
        if p.poll() is None:
            os.write(m, payload.encode())
        pump(p, m, stream, 1.2)
        snapshot(screen, f"key {label}")
    if p.poll() is None:
        p.terminate()
        try:
            p.wait(3)
        except subprocess.TimeoutExpired:
            p.kill()
    print(f"\n=== process exit: {p.poll()} ===")


if __name__ == "__main__":
    main()
