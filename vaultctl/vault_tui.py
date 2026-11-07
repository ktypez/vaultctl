#!/usr/bin/env python3
"""vault_tui — pretty terminal UI for the vaultctl credentials vault (vaultctl API).

Launched by `vaultctl ui` / `vaultctl tui`. Talks to the same Flask API as the
plain CLI, reuses ~/.vault-password, ~/.vault-cookie and ~/.vault-csrf.

Keys:
  /            search        n     new entry       Enter/c  copy value (masked)
  e            edit value    d     delete          u/U   unlock / lock
  r            refresh       a     audit log       q     quit
"""
import http.cookiejar
import json
import os
import sys
import time
import urllib.error
import urllib.request

from textual import work
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, Vertical
from textual.message import Message
from textual.reactive import reactive
from textual.screen import ModalScreen
from textual.widgets import DataTable, Footer, Header, Input, Label, Static

BASE = os.environ.get("VAULT_URL", "http://127.0.0.1:8787")
PW_FILE = os.path.expanduser("~/.vault-password")
COOKIE_FILE = os.path.expanduser("~/.vault-cookie")
CSRF_FILE = os.path.expanduser("~/.vault-csrf")


# --------------------------------------------------------------------------- #
# API client (same state files as the plain CLI)
# --------------------------------------------------------------------------- #
class VaultAPI:
    def __init__(self, base=BASE):
        self.base = base

    def _jar(self):
        jar = http.cookiejar.MozillaCookieJar(COOKIE_FILE)
        if os.path.exists(COOKIE_FILE):
            try:
                jar.load(ignore_discard=True, ignore_expires=True)
            except Exception:
                pass
        return jar

    def csrf(self):
        if os.path.exists(CSRF_FILE):
            with open(CSRF_FILE) as f:
                return f.read().strip()
        return ""

    def req(self, method, path, data=None, auth=False):
        headers = {}
        if data is not None:
            headers["Content-Type"] = "application/json"
        if auth:
            headers["X-CSRF-Token"] = self.csrf()
        r = urllib.request.Request(
            self.base + path,
            data=(json.dumps(data).encode() if data is not None else None),
            headers=headers,
            method=method,
        )
        opener = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(self._jar())
        )
        try:
            with opener.open(r) as resp:
                body = resp.read()
                return resp.status, (json.loads(body) if body else {})
        except urllib.error.HTTPError as e:
            try:
                j = json.loads(e.read())
            except Exception:
                j = {"error": "HTTP %d" % e.code}
            return e.code, j
        except Exception as e:
            return 0, {"error": str(e)}

    def status(self):
        code, j = self.req("GET", "/api/status")
        return j if code == 200 else {"initialized": False, "unlocked": False,
                                      "error": j.get("error", "offline")}

    def password(self):
        pw = os.environ.get("VAULT_PASSWORD")
        if pw:
            return pw
        if os.path.exists(PW_FILE):
            with open(PW_FILE) as f:
                return f.read().strip()
        return None

    def unlock(self, password=None):
        password = password or self.password()
        if not password:
            return False, "no password ($VAULT_PASSWORD or ~/.vault-password)"
        jar = self._jar()
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
        r = urllib.request.Request(
            self.base + "/api/unlock",
            data=json.dumps({"password": password}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with opener.open(r) as resp:
                body = json.loads(resp.read())
        except urllib.error.HTTPError as e:
            try:
                msg = json.loads(e.read()).get("error", "HTTP %d" % e.code)
            except Exception:
                msg = "HTTP %d" % e.code
            return False, msg
        except Exception as e:
            return False, str(e)
        jar.save(ignore_discard=True, ignore_expires=True)
        with open(CSRF_FILE, "w") as f:
            f.write(body["csrf"])
        os.chmod(CSRF_FILE, 0o600)
        return True, "unlocked"

    def lock(self):
        code, j = self.req("POST", "/api/lock", {}, auth=True)
        for p in (COOKIE_FILE, CSRF_FILE):
            try:
                os.remove(p)
            except FileNotFoundError:
                pass
        return code == 200, j.get("error", "locked")

    def entries(self):
        code, j = self.req("GET", "/api/entries")
        if code == 401:
            return None, "locked"
        if code != 200:
            return None, j.get("error", "error")
        return sorted(j.get("entries", [])), None

    def get_value(self, name):
        code, j = self.req("GET", "/api/entries/" + urllib.request.quote(name, safe=""))
        if code != 200:
            return None, j.get("error", "error")
        return j["value"], None

    def set_entry(self, name, value):
        code, j = self.req("POST", "/api/entries", {"name": name, "value": value},
                           auth=True)
        if code not in (200, 201):
            return False, j.get("error", "error")
        return True, name

    def delete_entry(self, name):
        code, j = self.req("DELETE", "/api/entries/" + urllib.request.quote(name, safe=""),
                           auth=True)
        if code != 200:
            return False, j.get("error", "error")
        return True, name

    def audit(self, n=50):
        code, j = self.req("GET", "/api/audit?n=%d" % n)
        if code != 200:
            return [], j.get("error", "error")
        return list(reversed(j.get("entries", []))), None


def clip_copy(text):
    """Copy via wl-copy/termux-api/xclip/xsel/pbcopy; last resort OSC52."""
    import subprocess
    for cmd in (["wl-copy"], ["termux-clipboard-set"], ["xclip", "-selection", "clipboard"],
                ["xsel", "-ib", "--clipboard"], ["pbcopy"]):
        try:
            subprocess.run(cmd, input=text.encode(), timeout=3,
                           stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                           check=True)
            return cmd[0]
        except Exception:
            continue
    try:
        import base64
        seq = "\x1b]52;c;%s\x07" % base64.b64encode(text.encode()).decode()
        with open("/dev/tty", "w") as tty:
            tty.write(seq)
            tty.flush()
        return "OSC52"
    except Exception:
        return None


# --------------------------------------------------------------------------- #
# Dialogs
# --------------------------------------------------------------------------- #
class Prompt(ModalScreen):
    """Simple one-line prompt modal. Posts Prompt.Done(value) or Dismissed."""

    class Done(Message):
        def __init__(self, value):
            self.value = value
            super().__init__()

    class Dismissed(Message):
        pass

    BINDINGS = [Binding("escape", "dismiss", "Cancel")]

    def __init__(self, title, placeholder="", secret=False, value=""):
        super().__init__()
        self._title, self._ph, self._secret, self._value = title, placeholder, secret, value

    def _result(self, ok, payload=None):
        """Dismiss with a result; app callbacks receive (ok, payload)."""
        self.dismiss((ok, payload))

    CSS = """
    Prompt { align: center middle; }
    #dialog { width: 70%; min-height: 5; padding: 1 2; border: round $accent;
              background: $surface; }
    #dlg-title { color: $accent; text-style: bold; }
    Input { border: round $panel; margin-top: 1; }
    """

    def compose(self):
        with Vertical(id="dialog"):
            yield Static(self._title, id="dlg-title")
            yield Input(placeholder=self._ph, password=self._secret,
                        value=self._value, id="dlg-input")

    def on_mount(self):
        self.query_one(Input).focus()

    def on_input_submitted(self, event: Input.Submitted):
        self.dismiss((True, event.value))

    def action_dismiss(self):
        self.dismiss((False, None))


class Confirm(ModalScreen):
    BINDINGS = [
        Binding("enter", "yes", "Yes", priority=True),
        Binding("escape", "no", "No", priority=True),
        Binding("y", "yes", "Yes", priority=True),
        Binding("n", "no", "No", priority=True),
    ]

    def __init__(self, question):
        super().__init__()
        self._q = question

    CSS = """
    Confirm { align: center middle; }
    #dialog { width: 60%; min-height: 5; padding: 1 2; border: round $warning;
              background: $surface; }
    #hint { color: $text-muted; }
    """

    def compose(self):
        with Vertical(id="dialog"):
            yield Static(self._q, markup=False)
            yield Static("[Enter] yes   [Esc/n] no", id="hint")

    def action_yes(self):
        self.dismiss(True)

    def action_no(self):
        self.dismiss(False)


# --------------------------------------------------------------------------- #
# Main app
# --------------------------------------------------------------------------- #
SEAL = "[bold red]V[/] "

STATUS_DOT = {True: "[green]●[/]", False: "[red]●[/]"}


class VaultTUI(App):
    TITLE = "vaultctl"
    CSS = """
    Screen { background: $background; }

    #banner {
        height: auto;
        padding: 1 2;
        border-bottom: solid $primary-lighten-2;
    }
    #seal { text-style: bold; }
    #brand { color: $text; text-style: bold; }
    #sub { color: $text-muted; }

    #main {
        height: 1fr;
    }

    #left {
        width: 1fr;
        padding: 0 1;
    }
    #search {
        border: round $panel;
        margin-bottom: 1;
    }
    #table {
        border: round $panel;
    }
    DataTable > .datatable--header {
        text-style: bold;
        color: $accent;
    }

    #right {
        width: 38;
        padding: 0 1;
        display: none;
    }
    #right.-show { display: block; }
    #detail-title { color: $accent; text-style: bold; }
    #detail-value {
        border: round $panel;
        padding: 1;
        color: $success;
        text-style: bold;
    }
    #detail-meta { color: $text-muted; }

    #statusbar {
        height: 1;
        dock: bottom;
        padding: 0 2;
        background: $surface;
        color: $text-muted;
    }
    """

    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("slash", "focus_search", "Search", show=False),
        Binding("n", "new_entry", "New", priority=True),
        Binding("e", "edit_entry", "Edit", priority=True),
        Binding("d", "delete_entry", "Delete", priority=True),
        Binding("c", "copy_entry", "Copy", priority=True),
        Binding("u", "unlock", "Unlock", priority=True),
        Binding("U", "lock", "Lock", priority=True),
        Binding("r", "refresh", "Refresh", priority=True),
        Binding("a", "audit", "Audit", priority=True),
        Binding("escape", "clear_or_quit", "", show=False, priority=True),
    ]

    def __init__(self):
        super().__init__()
        self.api = VaultAPI()
        self.names = []
        self.unlocked = False

    # ---- layout ---------------------------------------------------------- #
    def compose(self):
        yield Header(show_clock=True)
        with Vertical():
            with Horizontal(id="banner"):
                yield Static(SEAL + "[bold]vaultctl[/]  [dim]credentials vault[/]",
                             id="brand-block", markup=True)
            with Horizontal(id="main"):
                with Vertical(id="left"):
                    yield Input(placeholder="search… (/)", id="search")
                    yield DataTable(id="table", cursor_type="row", zebra_stripes=True)
                with Vertical(id="right"):
                    yield Static("", id="detail-title", markup=True)
                    yield Static("", id="detail-value", markup=False)
                    yield Static("", id="detail-meta", markup=True)
            yield Static("", id="statusbar", markup=True)
        yield Footer()

    def on_mount(self):
        table = self.query_one(DataTable)
        table.add_columns("  ", "ENTRY", "LENGTH")
        self.set_status("connecting…")
        self.refresh_all()

    def on_input_changed(self, event: Input.Changed):
        if event.input.id == "search":
            self.render_table()

    # ---- helpers ---------------------------------------------------------- #
    def set_status(self, msg, kind="info"):
        colors = {"info": "$text-muted", "ok": "$success", "warn": "$warning", "err": "red"}
        self.query_one("#statusbar", Static).update(
            "%s %s" % (SEAL, "[%s]%s[/]" % (colors[kind], msg))
        )

    def selected_name(self):
        table = self.query_one(DataTable)
        if table.row_count == 0:
            return None
        try:
            return str(table.get_row_at(table.cursor_row)[1])
        except Exception:
            return None

    def render_table(self):
        table = self.query_one(DataTable)
        term = self.query_one("#search", Input).value.strip().lower()
        rows = [n for n in self.names if not term or term in n.lower()]
        table.clear()
        for i, n in enumerate(rows, 1):
            table.add_row(("%d" % i).rjust(3), n, "-", key=n)
        self.query_one("#left").display = True
        if not self.unlocked:
            self.set_status("vault locked — press [b]u[/] to unlock", "warn")
        else:
            self.set_status("%d entries" % len(self.names), "ok")

    def show_detail(self, name, value):
        box = self.query_one("#right")
        box.display = True
        masked = value[:4] + "…" if len(value) > 8 else "••••"
        self.query_one("#detail-title", Static).update(SEAL + "[bold]%s[/]" % name)
        self.query_one("#detail-value", Static).update(masked)
        self.query_one("#detail-meta", Static).update(
            "length  %d\npress [b]c[/] to reveal + copy\n[b]e[/] edit  [b]d[/] delete"
            % len(value))

    def hide_detail(self):
        self.query_one("#right").display = False

    # ---- actions ----------------------------------------------------------- #
    def action_focus_search(self):
        self.query_one("#search", Input).focus()

    def action_clear_or_quit(self):
        inp = self.query_one("#search", Input)
        if inp.value:
            inp.value = ""
            self.render_table()
        elif self.query_one("#right").display:
            self.hide_detail()
        else:
            self.exit()

    def refresh_all(self):
        st = self.api.status()
        self.unlocked = bool(st.get("unlocked"))
        if self.unlocked:
            names, err = self.api.entries()
            if names is not None:
                self.names = names
                self.render_table()
            else:
                self.set_status(err, "err")
        else:
            self.names = []
            self.render_table()

    def action_unlock(self):
        ok, msg = self.api.unlock()
        if ok:
            self.refresh_all()
            self.set_status("unlocked", "ok")
        else:
            self.set_status("unlock failed: " + msg, "err")

    def action_lock(self):
        ok, msg = self.api.lock()
        self.refresh_all()
        self.hide_detail()
        self.set_status("locked" if ok else msg, "warn" if ok else "err")

    def action_refresh(self):
        self.refresh_all()

    def action_copy_entry(self):
        name = self.selected_name()
        if not name:
            return
        value, err = self.api.get_value(name)
        if value is None:
            self.set_status(err, "err")
            return
        tool = clip_copy(value)
        self.show_detail(name, value)
        if tool:
            self.set_status("copied %s → clipboard (%s)" % (name, tool), "ok")
        else:
            self.set_status("no clipboard tool — value shown in panel", "warn")

    def action_new_entry(self):
        def step1(res):
            ok, name = res
            if not ok:
                return
            name = (name or "").strip()
            if not name:
                self.set_status("empty name", "err")
                return
            self.push_screen(Prompt("value for " + name, "paste value"),
                             lambda ev: self._save(name, ev))
        self.push_screen(Prompt("new entry name", "e.g. FOO_API_KEY"), step1)

    def _save(self, name, res):
        ok, value = res
        if not ok:
            return
        ok, msg = self.api.set_entry(name, value)
        if ok:
            self.refresh_all()
            self.set_status("saved " + name, "ok")
        else:
            self.set_status("save failed: " + msg, "err")

    def action_edit_entry(self):
        name = self.selected_name()
        if not name:
            return
        def done(res):
            ok, value = res
            if not ok:
                return
            ok2, msg = self.api.set_entry(name, value)
            if ok2:
                self.refresh_all()
                self.set_status("updated " + name, "ok")
            else:
                self.set_status("update failed: " + msg, "err")
        self.push_screen(Prompt("new value for " + name, "paste value"), done)

    def action_delete_entry(self):
        name = self.selected_name()
        if not name:
            return
        def on_confirm(confirmed):
            if not confirmed:
                return
            ok, err = self.api.delete_entry(name)
            if ok:
                self.hide_detail()
                self.refresh_all()
                self.set_status("deleted " + name, "warn")
            else:
                self.set_status("delete failed: " + err, "err")
        self.push_screen(Confirm("delete %s ? this cannot be undone" % name), on_confirm)

    def action_audit(self):
        entries, err = self.api.audit(60)
        if err and not entries:
            self.set_status(err, "err")
            return
        lines = [SEAL + "[bold]audit[/] [dim](last %d)[/]" % len(entries)]
        for e in entries:
            ts = (e.get("ts") or "")[11:19]
            dec = e.get("decision", "?")
            col = "green" if dec == "allow" else "red"
            lines.append("[dim]%s[/] [%s]%-5s[/] %-8s %s%s" % (
                ts, col, dec, e.get("action", ""), e.get("name", ""),
                "  [dim]" + str(e.get("reason", "")) + "[/]" if e.get("reason") else ""))
        self.push_screen(TextScreen("\n".join(lines)))


class TextScreen(ModalScreen):
    """Scrollable read-only text overlay (audit log)."""
    BINDINGS = [
        Binding("escape", "close", "Close", priority=True),
        Binding("q", "close", "Close", priority=True),
    ]

    def __init__(self, text):
        super().__init__()
        self._text = text

    CSS = """
    TextScreen { align: center middle; }
    #panel { width: 90%; height: 80%; border: round $primary; background: $surface;
             padding: 1 2; overflow-y: auto; }
    """

    def compose(self):
        yield Static(self._text, id="panel", markup=True)

    def action_close(self):
        self.dismiss()


def main():
    if "--smoke" in sys.argv:
        app = VaultTUI()
        async def smoke():
            async with app.run_test(size=(100, 32)) as pilot:
                await pilot.pause()
                app.refresh_all()
                await pilot.pause()
        import asyncio
        asyncio.run(smoke())
        print("SMOKE OK — unlocked:", app.unlocked, "entries:", len(app.names))
        return
    if not sys.stdin.isatty() or not sys.stdout.isatty():
        sys.stderr.write("vault_tui needs an interactive terminal\n")
        sys.exit(2)
    VaultTUI().run()


if __name__ == "__main__":
    main()
