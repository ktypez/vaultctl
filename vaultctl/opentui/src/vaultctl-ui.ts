/**
 * vaultctl-ui.ts — vaultctl terminal UI, built on OpenTUI (@opentui/core).
 *
 *   bun run src/vaultctl-ui.ts          # interactive
 *   bun run src/vaultctl-ui.ts --check  # non-interactive backend self-test
 *
 * Security: entry values are only held in memory. They are never logged,
 * never written to disk by the TUI, and never rendered unless the user
 * explicitly reveals them (v) or copies them (c).
 */
import {
  BoxRenderable,
  InputRenderable,
  SelectRenderable,
  TextRenderable,
  createCliRenderer,
} from "@opentui/core";
import type { KeyEvent } from "@opentui/core";
import { BASE, VaultClient, maskedPreview, sha256Short } from "./vault-client.ts";
import { copyText, pasteText } from "./clipboard.ts";

// ---------------------------------------------------------------- theme ----
const T = {
  bg: "#141011",
  panel: "#1c1617",
  line: "#3b2f2d",
  accent: "#c9433a",
  gold: "#d9a441",
  text: "#e9e3dd",
  dim: "#8b807a",
  ok: "#5fae6e",
  err: "#d9534f",
  warn: "#d9a441",
};

const HINTS_1 = "[/]filter  [n]ew  [e]dit  [d]elete  [c]opy  [v]iew  [u]nlock  [l]ock";
const HINTS_2 = "[a]udit  [r]efresh  [j/k or ↑↓]move  [g/G]top/end  [q]uit";

// ------------------------------------------------------------ self-test ----
async function selfCheck(): Promise<number> {
  const c = new VaultClient();
  const st = await c.status();
  console.log(`backend      ${BASE}`);
  console.log(`initialized  ${st.initialized}`);
  console.log(`unlocked     ${st.unlocked}`);
  if (!st.initialized) {
    console.log("CHECK FAILED: vault not initialized");
    return 1;
  }
  if (!st.unlocked) {
    console.log("(locked — attempting unlock with ~/.vault-password)");
    await c.unlock();
  }
  const names = await c.entries();
  console.log(`entries      ${names.length}`);
  const audit = await c.audit(5);
  console.log(`audit tail   ${audit.length}`);
  console.log("CHECK OK");
  return 0;
}

// ---------------------------------------------------------------- modal ----
type ModalKeyHandler = (key: KeyEvent) => boolean;

class Modal {
  box: BoxRenderable;
  body: BoxRenderable;
  hint: TextRenderable;
  onKey: ModalKeyHandler = () => false;
  focusFirst?: () => void;

  constructor(
    private renderer: any,
    title: string,
    opts: { width?: number | `${number}%`; height?: number | "auto"; top?: number } = {},
  ) {
    const width: number | `${number}%` = opts.width ?? "64%";
    // centre horizontally: left = (100 - width%) / 2
    const left: number | `${number}%` =
      typeof width === "number"
        ? Math.max(0, Math.floor((renderer.terminalWidth - width) / 2))
        : `${(100 - Number(String(width).replace("%", ""))) / 2}%`;
    this.box = new BoxRenderable(renderer, {
      position: "absolute",
      left,
      top: opts.top ?? 4,
      width,
      height: opts.height ?? ("auto" as any),
      flexDirection: "column",
      border: true,
      borderStyle: "double",
      borderColor: T.accent,
      title: ` ${title} `,
      titleAlignment: "center",
      titleColor: T.gold,
      backgroundColor: T.panel,
      paddingLeft: 2,
      paddingRight: 2,
      paddingTop: 1,
      paddingBottom: 1,
      zIndex: 100,
    });
    this.body = new BoxRenderable(renderer, { flexDirection: "column", flexGrow: 1, backgroundColor: T.panel });
    this.hint = new TextRenderable(renderer, { content: "", fg: T.dim, bg: T.panel });
    this.box.add(this.body);
    this.box.add(this.hint);
  }

  setHint(s: string) {
    this.hint.content = s;
  }
}

// ------------------------------------------------------------------ app ----
class VaultctlApp {
  renderer: any;
  client = new VaultClient();

  header!: TextRenderable;
  search!: InputRenderable;
  list!: SelectRenderable;
  listBox!: BoxRenderable;
  detail!: TextRenderable;
  status!: TextRenderable;

  names: string[] = [];
  filtered: string[] = [];
  current: string | null = null;
  currentValue: string | null = null;
  revealed = false;
  unlocked = false;

  modal: Modal | null = null;
  auditLines: string[] = [];
  auditOffset = 0;
  auditView: TextRenderable | null = null;

  constructor(renderer: any) {
    this.renderer = renderer;
  }

  // ---------------------------------------------------------- rendering ----
  build() {
    const r = this.renderer;
    const root = new BoxRenderable(r, {
      flexDirection: "column",
      width: "100%",
      height: "100%",
      backgroundColor: T.bg,
    });

    // header
    const headerBox = new BoxRenderable(r, {
      height: 3,
      border: true,
      borderStyle: "rounded",
      borderColor: T.accent,
      title: " vaultctl ",
      titleAlignment: "center",
      titleColor: T.gold,
      backgroundColor: T.panel,
      paddingLeft: 2,
      paddingRight: 2,
    });
    this.header = new TextRenderable(r, { content: "…", fg: T.text, bg: T.panel });
    headerBox.add(this.header);

    // main
    const main = new BoxRenderable(r, {
      flexDirection: "row",
      flexGrow: 1,
      width: "100%",
      backgroundColor: T.bg,
    });

    const left = new BoxRenderable(r, {
      flexDirection: "column",
      width: "36%",
      backgroundColor: T.bg,
    });
    const searchBox = new BoxRenderable(r, {
      height: 3,
      border: true,
      borderStyle: "rounded",
      borderColor: T.line,
      title: " filter ",
      titleColor: T.dim,
      backgroundColor: T.panel,
      paddingLeft: 1,
      paddingRight: 1,
    });
    this.search = new InputRenderable(r, {
      placeholder: "/ to search",
      backgroundColor: T.panel,
      textColor: T.text,
      focusedBackgroundColor: T.panel,
      focusedTextColor: T.gold,
    });
    searchBox.add(this.search);

    const listBox = new BoxRenderable(r, {
      flexGrow: 1,
      border: true,
      borderStyle: "rounded",
      borderColor: T.line,
      title: " entries ",
      titleColor: T.dim,
      backgroundColor: T.panel,
      paddingLeft: 1,
      paddingRight: 1,
    });
    this.listBox = listBox;
    this.list = new SelectRenderable(r, {
      flexGrow: 1,
      options: [],
      backgroundColor: T.panel,
      focusedBackgroundColor: T.panel,
      textColor: T.text,
      focusedTextColor: T.text,
      selectedBackgroundColor: T.accent,
      selectedTextColor: "#ffffff",
      descriptionColor: T.dim,
      selectedDescriptionColor: "#f2d5cf",
      showScrollIndicator: true,
      wrapSelection: true,
      showDescription: false,
      itemSpacing: 0,
    });
    this.list.on("selectionChanged", () => void this.onSelect());
    listBox.add(this.list);

    left.add(searchBox);
    left.add(listBox);

    const right = new BoxRenderable(r, {
      flexGrow: 1,
      border: true,
      borderStyle: "rounded",
      borderColor: T.line,
      title: " detail ",
      titleColor: T.dim,
      backgroundColor: T.panel,
      paddingLeft: 2,
      paddingRight: 2,
      paddingTop: 1,
    });
    this.detail = new TextRenderable(r, { content: "", fg: T.text, bg: T.panel });
    right.add(this.detail);

    main.add(left);
    main.add(right);

    this.status = new TextRenderable(r, { content: "", fg: T.dim, bg: T.bg });
    const footer = new BoxRenderable(r, {
      flexDirection: "column",
      height: 2,
      width: "100%",
      backgroundColor: T.bg,
    });
    footer.add(new TextRenderable(r, { content: HINTS_1, fg: T.dim, bg: T.bg }));
    footer.add(new TextRenderable(r, { content: HINTS_2, fg: T.dim, bg: T.bg }));

    root.add(headerBox);
    root.add(main);
    root.add(this.status);
    root.add(footer);
    r.root.add(root);

    this.search.on("input", () => this.applyFilter());
    this.search.on("enter", () => this.focusList());

    r.keyInput.on("keypress", (key: KeyEvent) => this.onKey(key));
  }

  renderHeader() {
    const state = this.unlocked
      ? `● unlocked · ${this.names.length} entries`
      : "○ locked";
    this.header.content = `${state}    ${BASE}`;
    this.header.fg = this.unlocked ? T.ok : T.err;
  }

  renderDetail() {
    if (!this.current) {
      this.detail.content = "\n  no entry selected";
      return;
    }
    if (this.currentValue === null) {
      this.detail.content = `\n  NAME     ${this.current}\n  VALUE    … loading`;
      return;
    }
    const v = this.currentValue;
    const lines = [
      "",
      `  NAME     ${this.current}`,
      `  STATE    ● present`,
      `  LENGTH   ${v.length} chars`,
      `  SHA256   ${sha256Short(v)}  (first 10 hex)`,
      "",
      this.revealed
        ? `  VALUE    ${v}`
        : `  VALUE    ${maskedPreview(v.length)}  hidden`,
      "",
      this.revealed
        ? "  [v] hide   [c] copy"
        : "  [v] reveal   [c] copy to clipboard",
    ];
    this.detail.content = lines.join("\n");
  }

  setStatus(msg: string, kind: "info" | "ok" | "err" = "info") {
    this.status.content = "  " + msg;
    this.status.fg = kind === "ok" ? T.ok : kind === "err" ? T.err : T.dim;
  }

  // --------------------------------------------------------------- data ----
  async refresh(keepSelection = true) {
    try {
      const st = await this.client.status();
      this.unlocked = st.unlocked;
      if (!st.initialized) {
        this.unlocked = false;
        this.names = [];
        this.setStatus("vault not initialized — run `vaultctl setup` first", "err");
      } else if (!st.unlocked) {
        this.setStatus("locked — press u to unlock", "err");
        this.names = [];
      } else {
        const prev = keepSelection ? this.current : null;
        this.names = await this.client.entries();
        if (prev && this.names.includes(prev)) this.current = prev;
        else this.current = null;
        this.setStatus(`${this.names.length} entries loaded`, "ok");
      }
    } catch (e) {
      this.unlocked = false;
      this.names = [];
      this.setStatus(`backend unreachable at ${BASE}: ${String(e)}`, "err");
    }
    this.applyFilter();
    this.renderHeader();
    if (this.current) await this.loadCurrent();
    else this.renderDetail();
    if (!this.unlocked) this.openUnlock();
  }

  applyFilter() {
    const q = this.search.value.trim().toLowerCase();
    this.filtered = q ? this.names.filter((n) => n.toLowerCase().includes(q)) : [...this.names];
    const prev = this.current;
    this.list.options = this.filtered.map((n) => ({ name: n, description: "" }));
    const idx = prev ? this.filtered.indexOf(prev) : -1;
    if (idx >= 0) this.list.setSelectedIndex(idx);
    else if (this.filtered.length) this.list.setSelectedIndex(0);
    this.updateListTitle();
    const first = this.filtered.length ? this.filtered[Math.max(0, this.list.getSelectedIndex())] : null;
    if (first !== this.current) {
      this.current = first;
      void this.loadCurrent();
    }
    if (!this.filtered.length && this.names.length) {
      this.setStatus(`no entry matches "${q}"`, "info");
    }
    this.renderer.requestRender();
  }

  updateListTitle() {
    const q = this.search.value.trim().toLowerCase();
    const pos = this.filtered.length ? this.list.getSelectedIndex() + 1 : 0;
    this.listBox.title = q
      ? ` entries ${pos}/${this.filtered.length} of ${this.names.length} (filter: ${q}) `
      : ` entries ${pos}/${this.filtered.length} `;
  }

  async onSelect() {
    const opt = this.list.getSelectedOption();
    if (!opt) return;
    this.updateListTitle();
    const name = String(opt.name);
    if (name === this.current) return;
    this.current = name;
    await this.loadCurrent();
  }

  async loadCurrent() {
    if (!this.current) {
      this.currentValue = null;
      this.renderDetail();
      this.renderer.requestRender();
      return;
    }
    this.revealed = false;
    this.currentValue = null;
    this.renderDetail();
    this.renderer.requestRender();
    try {
      this.currentValue = await this.client.get(this.current);
    } catch (e) {
      this.currentValue = null;
      this.setStatus(`cannot read ${this.current}: ${String(e)}`, "err");
    }
    this.renderDetail();
    this.renderer.requestRender();
  }

  // ------------------------------------------------------------- focus ----
  focusList() {
    this.list.focus();
    this.renderer.requestRender();
  }

  focusSearch() {
    this.search.focus();
    this.setStatus("type to filter · Esc to return to the list");
    this.renderer.requestRender();
  }

  // ------------------------------------------------------------- modal -----
  openModal(
    title: string,
    opts?: { width?: number | `${number}%`; height?: number | "auto"; top?: number },
  ) {
    this.closeModal();
    const m = new Modal(this.renderer, title, opts);
    this.modal = m;
    this.renderer.root.add(m.box);
    if (m.focusFirst) m.focusFirst();
    this.renderer.requestRender();
    return m;
  }

  closeModal() {
    if (!this.modal) return;
    try {
      this.renderer.root.remove(this.modal.box);
    } catch {
      /* ignore */
    }
    this.modal = null;
    this.auditView = null;
    this.focusList();
    this.renderer.requestRender();
  }

  openUnlock() {
    const m = this.openModal("unlock vault", { width: "56%", height: 7, top: 6 });
    const label = new TextRenderable(this.renderer, {
      content: "Master password (typing is hidden; ^v pastes from clipboard):",
      fg: T.dim,
      bg: T.panel,
    });
    const pw = new InputRenderable(this.renderer, {
      placeholder: "master password",
      backgroundColor: T.panel,
      textColor: T.panel,
      focusedBackgroundColor: T.panel,
      focusedTextColor: T.panel,
    });
    const count = new TextRenderable(this.renderer, { content: "length 0", fg: T.dim, bg: T.panel });
    pw.on("input", () => {
      count.content = `length ${pw.value.length}`;
      this.renderer.requestRender();
    });
    m.body.add(label);
    m.body.add(pw);
    m.body.add(count);
    m.focusFirst = () => pw.focus();
    m.setHint("Enter unlock   ^v paste   Esc cancel");

    const submit = async () => {
      const value = pw.value;
      if (!value) return;
      this.setStatus("unlocking…");
      try {
        await this.client.unlock(value);
        this.closeModal();
        await this.refresh(false);
        this.setStatus("unlocked", "ok");
      } catch (e) {
        this.setStatus(`unlock failed: ${String((e as Error).message)}`, "err");
      }
    };

    m.onKey = (key) => {
      if (key.name === "escape") {
        this.closeModal();
        return true;
      }
      if (key.name === "return" || key.name === "enter") {
        void submit();
        return true;
      }
      if (key.ctrl && key.name === "v") {
        void (async () => {
          const clip = await pasteText();
          if (clip === null) this.setStatus("no clipboard backend available", "err");
          else {
            pw.value = clip;
            count.content = `length ${clip.length}`;
            this.setStatus("pasted from clipboard", "ok");
          }
          this.renderer.requestRender();
        })();
        return true;
      }
      return false;
    };
    pw.focus();
  }

  openEntryForm(kind: "new" | "edit") {
    const isEdit = kind === "edit";
    if (isEdit && !this.current) {
      this.setStatus("select an entry first", "err");
      return;
    }
    const m = this.openModal(isEdit ? `edit ${this.current}` : "new entry", { width: "66%", height: 9, top: 5 });
    const nameLabel = new TextRenderable(this.renderer, { content: "name", fg: T.dim, bg: T.panel });
    const nameInput = new InputRenderable(this.renderer, {
      placeholder: "ENTRY_NAME",
      backgroundColor: T.panel,
      textColor: T.text,
      focusedBackgroundColor: T.panel,
      focusedTextColor: T.gold,
      value: isEdit ? String(this.current) : "",
    });
    const valLabel = new TextRenderable(this.renderer, { content: "value", fg: T.dim, bg: T.panel });
    // Typing is invisible: text colour == background. `^v` reads the clipboard
    // so secrets never have to be rendered on screen.
    const valInput = new InputRenderable(this.renderer, {
      placeholder: isEdit ? "leave empty to keep current value" : "secret value",
      backgroundColor: T.panel,
      textColor: T.panel,
      focusedBackgroundColor: T.panel,
      focusedTextColor: T.panel,
    });
    const valCount = new TextRenderable(this.renderer, { content: "value length 0", fg: T.dim, bg: T.panel });
    valInput.on("input", () => {
      valCount.content = `value length ${valInput.value.length}`;
      this.renderer.requestRender();
    });

    m.body.add(nameLabel);
    m.body.add(nameInput);
    m.body.add(valLabel);
    m.body.add(valInput);
    m.body.add(valCount);
    m.focusFirst = () => nameInput.focus();
    m.setHint("Tab next   Enter save   ^v paste value   Esc cancel");

    const save = async () => {
      const name = nameInput.value.trim();
      if (!name) {
        this.setStatus("name is required", "err");
        return;
      }
      const value = valInput.value;
      if (!value && !isEdit) {
        this.setStatus("value is required for a new entry", "err");
        return;
      }
      try {
        if (value) {
          await this.client.set(name, value);
          this.setStatus(`saved ${name}`, "ok");
        } else if (isEdit && name !== this.current) {
          const old = await this.client.get(String(this.current));
          await this.client.set(name, old);
          await this.client.del(String(this.current));
          this.setStatus(`renamed ${this.current} → ${name}`, "ok");
        } else {
          this.setStatus("nothing to save (value empty)", "info");
          return;
        }
        this.current = name;
        this.closeModal();
        await this.refresh(false);
      } catch (e) {
        this.setStatus(`save failed: ${String((e as Error).message)}`, "err");
      }
    };

    m.onKey = (key) => {
      if (key.name === "escape") {
        this.closeModal();
        return true;
      }
      if (key.ctrl && key.name === "v") {
        void (async () => {
          const clip = await pasteText();
          if (clip === null) this.setStatus("no clipboard backend available", "err");
          else {
            valInput.value = clip;
            valCount.content = `value length ${clip.length} (invisible)`;
            this.setStatus("value pasted (hidden)", "ok");
          }
          this.renderer.requestRender();
        })();
        return true;
      }
      if (key.name === "tab") {
        if (this.renderer.currentFocusedRenderable === nameInput) valInput.focus();
        else nameInput.focus();
        this.renderer.requestRender();
        return true;
      }
      if (key.name === "return" || key.name === "enter") {
        if (this.renderer.currentFocusedRenderable === nameInput && valInput.value.length === 0 && !isEdit) {
          valInput.focus();
          this.renderer.requestRender();
          return true;
        }
        void save();
        return true;
      }
      return false;
    };
    nameInput.focus();
  }

  openDelete() {
    if (!this.current) {
      this.setStatus("select an entry first", "err");
      return;
    }
    const m = this.openModal("confirm delete", { width: "60%", height: 6, top: 6 });
    const name = String(this.current);
    const text = new TextRenderable(this.renderer, {
      content: `Delete entry "${name}"?\nThis cannot be undone.`,
      fg: T.text,
      bg: T.panel,
    });
    m.body.add(text);
    m.setHint("y confirm   n / Esc cancel");
    m.onKey = (key) => {
      if (key.name === "y") {
        void (async () => {
          try {
            await this.client.del(name);
            this.setStatus(`deleted ${name}`, "ok");
            this.current = null;
            this.closeModal();
            await this.refresh(false);
          } catch (e) {
            this.setStatus(`delete failed: ${String((e as Error).message)}`, "err");
          }
        })();
        return true;
      }
      if (key.name === "n" || key.name === "escape") {
        this.closeModal();
        this.setStatus("delete cancelled");
        return true;
      }
      return false;
    };
  }

  async openAudit() {
    const m = this.openModal("audit log", { width: "86%", height: this.auditRows + 5, top: 2 });
    let rows: any[] = [];
    try {
      rows = await this.client.audit(200);
    } catch (e) {
      rows = [];
      this.setStatus(`audit failed: ${String((e as Error).message)}`, "err");
    }
    const innerW = Math.max(40, Math.floor(this.renderer.terminalWidth * 0.86) - 8);
    this.auditLines = rows.map((r) => {
      const ts = String(r.ts ?? "");
      const t = ts.replace("T", " ").replace("Z", "");
      const action = String(r.action ?? "?").slice(0, 9).padEnd(9);
      const dec = String(r.decision ?? "").slice(0, 7).padEnd(7);
      const nm = String(r.name ?? "-");
      const nameCol = (nm.length > 26 ? nm.slice(0, 25) + "…" : nm).padEnd(26);
      const reason = String(r.reason ?? "");
      return `${t}  ${action} ${dec}  ${nameCol}  ${reason}`.slice(0, innerW);
    });
    if (!this.auditLines.length) this.auditLines = ["(no audit records)"];
    const visible = this.auditRows;
    this.auditOffset = Math.max(0, this.auditLines.length - visible);
    const view = new TextRenderable(this.renderer, { content: "", fg: T.text, bg: T.panel });
    this.auditView = view;
    m.body.add(view);
    this.renderAudit();
    m.setHint("↑↓ / j k scroll   PgUp PgDn page   Esc close");
    m.onKey = (key) => {
      const page = this.auditRows;
      const maxOffset = Math.max(0, this.auditLines.length - this.auditRows);
      if (key.name === "escape" || key.name === "q") {
        this.closeModal();
        return true;
      }
      if (key.name === "down" || key.name === "j") {
        this.auditOffset = Math.min(maxOffset, this.auditOffset + 1);
      } else if (key.name === "up" || key.name === "k") {
        this.auditOffset = Math.max(0, this.auditOffset - 1);
      } else if (key.name === "pagedown") {
        this.auditOffset = Math.min(maxOffset, this.auditOffset + page);
      } else if (key.name === "pageup") {
        this.auditOffset = Math.max(0, this.auditOffset - page);
      } else {
        return false;
      }
      this.renderAudit();
      this.renderer.requestRender();
      return true;
    };
  }

  get auditRows(): number {
    return Math.max(6, Math.floor(this.renderer.terminalHeight * 0.8) - 6);
  }

  renderAudit() {
    if (!this.auditView) return;
    const rows = this.auditRows;
    const total = this.auditLines.length;
    const start = Math.max(0, Math.min(this.auditOffset, Math.max(0, total - rows)));
    const win = this.auditLines.slice(start, start + rows);
    const head = `records ${total}   showing ${start + 1}-${start + win.length}\n`;
    this.auditView.content = head + win.join("\n");
  }

  // ------------------------------------------------------------ actions ----
  async doCopy() {
    if (!this.current) {
      this.setStatus("select an entry first", "err");
      return;
    }
    try {
      const value = this.currentValue ?? (await this.client.get(this.current));
      const how = await copyText(value);
      if (how === "failed") this.setStatus("copy failed — no clipboard backend", "err");
      else this.setStatus(`copied ${this.current} to clipboard (${how})${how === "osc52" ? " — needs terminal OSC52 support" : ""}`, "ok");
    } catch (e) {
      this.setStatus(`copy failed: ${String(e)}`, "err");
    }
    this.renderer.requestRender();
  }

  toggleReveal() {
    if (!this.currentValue) {
      this.setStatus("nothing to reveal", "err");
      return;
    }
    this.revealed = !this.revealed;
    this.setStatus(
      this.revealed
        ? "value shown on screen — press v to hide again"
        : "value hidden",
      this.revealed ? "err" : "info",
    );
    this.renderDetail();
    this.renderer.requestRender();
  }

  async doLock() {
    try {
      await this.client.lock();
      this.currentValue = null;
      this.current = null;
      this.revealed = false;
      await this.refresh(false);
      this.setStatus("locked", "ok");
    } catch (e) {
      this.setStatus(`lock failed: ${String(e)}`, "err");
    }
  }

  // ------------------------------------------------------------ key map ----
  onKey(key: KeyEvent) {
    if (this.modal) {
      if (this.modal.onKey(key)) key.preventDefault();
      return;
    }
    const focused = this.renderer.currentFocusedRenderable;
    if (focused === this.search) {
      if (key.name === "escape") {
        this.search.value = "";
        this.applyFilter();
        this.focusList();
        this.setStatus("filter cleared");
        key.preventDefault();
      }
      return;
    }
    const name = key.name;
    switch (name) {
      case "q":
        void this.quit();
        key.preventDefault();
        break;
      case "/":
        this.focusSearch();
        key.preventDefault();
        break;
      case "n":
        this.openEntryForm("new");
        key.preventDefault();
        break;
      case "e":
        this.openEntryForm("edit");
        key.preventDefault();
        break;
      case "d":
        this.openDelete();
        key.preventDefault();
        break;
      case "c":
        void this.doCopy();
        key.preventDefault();
        break;
      case "v":
        this.toggleReveal();
        key.preventDefault();
        break;
      case "u":
        this.openUnlock();
        key.preventDefault();
        break;
      case "l":
        void this.doLock();
        key.preventDefault();
        break;
      case "a":
        void this.openAudit();
        key.preventDefault();
        break;
      case "r":
        void this.refresh();
        this.setStatus("refreshed", "ok");
        key.preventDefault();
        break;
      case "j":
        this.list.moveDown();
        this.updateListTitle();
        key.preventDefault();
        break;
      case "k":
        this.list.moveUp();
        this.updateListTitle();
        key.preventDefault();
        break;
      case "g":
        this.list.setSelectedIndex(0);
        this.updateListTitle();
        key.preventDefault();
        break;
      case "G":
        this.list.setSelectedIndex(Math.max(0, this.filtered.length - 1));
        this.updateListTitle();
        key.preventDefault();
        break;
      default:
        break;
    }
  }

  async quit() {
    try {
      this.renderer.destroy();
      await this.renderer.closed;
    } catch {
      /* ignore */
    }
    process.exit(0);
  }

  async start() {
    this.build();
    await this.refresh(false);
    this.focusList();
  }
}

// ------------------------------------------------------------------ main ----
async function main() {
  const args = process.argv.slice(2);
  if (args.includes("--check") || args.includes("--smoke")) {
    process.exit(await selfCheck());
  }
  if (args.includes("--help") || args.includes("-h")) {
    console.log("vaultctl TUI (OpenTUI)\n  --check  non-interactive backend self-test\n  --help   this text");
    process.exit(0);
  }
  if (!process.stdin.isTTY || !process.stdout.isTTY) {
    process.stderr.write("vaultctl TUI needs an interactive terminal (run `vaultctl ui`)\n");
    process.exit(2);
  }

  const renderer = await createCliRenderer({ exitOnCtrlC: true, useMouse: false });
  const app = new VaultctlApp(renderer);
  await app.start();
}

await main();
