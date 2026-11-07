/**
 * vault-client.ts — HTTP client for the vaultctl backend.
 *
 * Shares session state with the `vaultctl` CLI:
 *   ~/.vault-password (master password, 0600)
 *   ~/.vault-cookie   (Netscape cookie jar, cookie name "sid")
 *   ~/.vault-csrf     (CSRF token for state-changing calls)
 *
 * Security rules:
 *   - secret values are only ever held in memory, never logged or written;
 *   - nothing in this module prints or echoes an entry value.
 */
import { existsSync, readFileSync, writeFileSync, chmodSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

export const BASE = process.env.VAULT_URL ?? "http://127.0.0.1:8787";
const PW_FILE = join(homedir(), ".vault-password");
const COOKIE_FILE = join(homedir(), ".vault-cookie");
const CSRF_FILE = join(homedir(), ".vault-csrf");
const COOKIE_NAME = "sid";

export type ApiResult = { status: number; json: any; networkError?: string };

function readCookie(): string {
  try {
    if (!existsSync(COOKIE_FILE)) return "";
    for (const line of readFileSync(COOKIE_FILE, "utf8").split("\n")) {
      if (!line.trim() || line.startsWith("#")) continue;
      const f = line.split("\t");
      if (f.length >= 7 && f[5] === COOKIE_NAME) return f[6].trim();
    }
  } catch {
    /* ignore */
  }
  return "";
}

function readCsrf(): string {
  try {
    return existsSync(CSRF_FILE) ? readFileSync(CSRF_FILE, "utf8").trim() : "";
  } catch {
    return "";
  }
}

export function readPassword(): string | null {
  const env = process.env.VAULT_PASSWORD;
  if (env) return env;
  try {
    return existsSync(PW_FILE) ? readFileSync(PW_FILE, "utf8").trim() : null;
  } catch {
    return null;
  }
}

function writeCookie(token: string): void {
  const exp = Math.floor(Date.now() / 1000) + 30 * 24 * 3600;
  const body =
    "# Netscape HTTP Cookie File\n" +
    `127.0.0.1\tFALSE\t/\tFALSE\t${exp}\t${COOKIE_NAME}\t${token}\n`;
  writeFileSync(COOKIE_FILE, body);
  try {
    chmodSync(COOKIE_FILE, 0o600);
  } catch {
    /* ignore */
  }
}

function writeCsrf(token: string): void {
  writeFileSync(CSRF_FILE, token);
  try {
    chmodSync(CSRF_FILE, 0o600);
  } catch {
    /* ignore */
  }
}

export class VaultClient {
  cookie = readCookie();
  csrf = readCsrf();
  lastError = "";

  private async raw(method: string, path: string, body?: unknown, csrf = false): Promise<ApiResult> {
    const headers: Record<string, string> = {};
    if (this.cookie) headers["Cookie"] = `${COOKIE_NAME}=${this.cookie}`;
    if (body !== undefined) headers["Content-Type"] = "application/json";
    if (csrf) headers["X-CSRF-Token"] = this.csrf;
    try {
      const res = await fetch(BASE + path, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      });
      let json: any = {};
      const text = await res.text();
      if (text) {
        try {
          json = JSON.parse(text);
        } catch {
          json = { raw: text };
        }
      }
      return { status: res.status, json };
    } catch (e) {
      this.lastError = String(e);
      return { status: 0, json: {}, networkError: String(e) };
    }
  }

  async status(): Promise<{ initialized: boolean; unlocked: boolean }> {
    const r = await this.raw("GET", "/api/status");
    return { initialized: !!r.json?.initialized, unlocked: !!r.json?.unlocked };
  }

  async unlock(password?: string): Promise<boolean> {
    const pw = password ?? readPassword();
    if (!pw) throw new Error("no master password (set VAULT_PASSWORD or ~/.vault-password)");
    // Unlock with a raw fetch so we can capture Set-Cookie ourselves.
    const res = await fetch(BASE + "/api/unlock", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password: pw }),
    });
    const json: any = await res.json().catch(() => ({}));
    if (res.status !== 200 || !json?.ok) {
      throw new Error(json?.error ?? `unlock failed (${res.status})`);
    }
    const setCookie = res.headers.getSetCookie?.() ?? [];
    let token = "";
    for (const c of setCookie) {
      const m = /(?:^|;\s*)sid=([^;]+)/.exec(c);
      if (m) token = m[1];
    }
    if (!token) token = readCookie();
    this.cookie = token;
    this.csrf = json.csrf ?? "";
    writeCookie(token);
    writeCsrf(this.csrf);
    return true;
  }

  async lock(): Promise<void> {
    await this.raw("POST", "/api/lock", {}, true);
    this.cookie = "";
    this.csrf = "";
    try {
      writeCookie("");
    } catch {
      /* ignore */
    }
  }

  async entries(): Promise<string[]> {
    const r = await this.raw("GET", "/api/entries");
    if (r.status !== 200) throw new Error(r.json?.error ?? `entries failed (${r.status})`);
    return (r.json?.entries ?? []) as string[];
  }

  /** Returns the plaintext value. Callers must never log/echo it. */
  async get(name: string): Promise<string> {
    const r = await this.raw("GET", "/api/entries/" + encodeURIComponent(name));
    if (r.status !== 200) throw new Error(r.json?.error ?? `get failed (${r.status})`);
    return String(r.json?.value ?? "");
  }

  async set(name: string, value: string): Promise<void> {
    const r = await this.raw("POST", "/api/entries", { name, value }, true);
    if (r.status !== 200 && r.status !== 201) {
      throw new Error(r.json?.error ?? `set failed (${r.status})`);
    }
  }

  async del(name: string): Promise<void> {
    const r = await this.raw("DELETE", "/api/entries/" + encodeURIComponent(name), undefined, true);
    if (r.status !== 200) throw new Error(r.json?.error ?? `delete failed (${r.status})`);
  }

  async audit(n = 100): Promise<any[]> {
    const r = await this.raw("GET", `/api/audit?n=${n}`);
    if (r.status !== 200) throw new Error(r.json?.error ?? `audit failed (${r.status})`);
    return (r.json?.entries ?? []) as any[];
  }
}

/** Safe display helpers — derived from the value, never the value itself. */
export function sha256Short(value: string, chars = 10): string {
  const h = new Bun.CryptoHasher("sha256");
  h.update(value);
  return h.digest("hex").slice(0, chars);
}

export function maskedPreview(len: number): string {
  return "•".repeat(Math.max(4, Math.min(len, 24)));
}
