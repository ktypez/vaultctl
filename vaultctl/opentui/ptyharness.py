#!/usr/bin/env python3
"""pty harness: run a TUI in a real pty, send keys, capture output."""
import os, pty, re, select, subprocess, sys, time

BUN = os.path.expanduser("~/.bun/bin/bun")
CWD = os.path.dirname(os.path.abspath(__file__))


def strip_ansi(b: bytes) -> str:
    s = b.decode("utf-8", "replace")
    s = re.sub(r"\x1b\][^\x07\x1b]*(\x07|\x1b\\)", "", s)   # OSC
    s = re.sub(r"\x1b\[[0-9;?]*[a-zA-Z]", "", s)              # CSI
    s = re.sub(r"\x1b[()][A-Z0-9]", "", s)
    s = re.sub(r"\x1b[=>]", "", s)
    return s.replace("\r", "")


def run(script, keys, wait=2.5, after=1.5, env=None, cols=100, rows=30):
    e = dict(os.environ)
    e.update({"TERM": "xterm-256color", "COLUMNS": str(cols), "LINES": str(rows)})
    if env:
        e.update(env)
    m, s = pty.openpty()
    os.set_blocking(s, False)
    p = subprocess.Popen([BUN, script], cwd=CWD, stdin=s, stdout=s, stderr=s,
                         env=e, close_fds=True)
    os.close(s)
    buf = b""

    def drain(sec):
        nonlocal buf
        end = time.time() + sec
        while time.time() < end and p.poll() is None:
            r, _, _ = select.select([m], [], [], 0.2)
            if r:
                try:
                    d = os.read(m, 65536)
                except OSError:
                    break
                if not d:
                    break
                buf += d

    drain(wait)
    for k in keys:
        try:
            os.write(m, k.encode() if isinstance(k, str) else k)
        except OSError:
            break
        drain(0.6)
    drain(after)
    rc = p.poll()
    if rc is None:
        p.terminate()
        try:
            p.wait(3)
        except subprocess.TimeoutExpired:
            p.kill()
        rc = "still-running(then killed)"
    os.close(m)
    return strip_ansi(buf), rc


if __name__ == "__main__":
    script = sys.argv[1]
    keys = sys.argv[2:] if len(sys.argv) > 2 else ["q"]
    out, rc = run(script, keys)
    print("=== exit:", rc, "=== bytes:", len(out))
    print(out[-3000:])
