/**
 * clipboard.ts — best-effort clipboard access, no external JS deps.
 *
 * Order: native helper (wl-copy/xclip/xsel/pbcopy) -> OSC 52 escape sequence.
 * OSC 52 goes through the pty, so it never lands in the rendered screen.
 */
const COPY_CMDS: string[][] = [
  ["wl-copy"],
  ["xclip", "-selection", "clipboard"],
  ["xsel", "--clipboard", "--input"],
  ["pbcopy"],
];
const PASTE_CMDS: string[][] = [
  ["wl-paste", "--no-newline"],
  ["xclip", "-selection", "clipboard", "-o"],
  ["xsel", "--clipboard", "--output"],
  ["pbpaste"],
];

function which(bin: string): boolean {
  try {
    return !!Bun.which(bin);
  } catch {
    return false;
  }
}

async function tryCmds(cmds: string[][], input: string | null): Promise<string | null> {
  for (const argv of cmds) {
    if (!which(argv[0])) continue;
    try {
      const proc = Bun.spawn(argv, {
        stdin: input === null ? "ignore" : new Blob([input]),
        stdout: "pipe",
        stderr: "ignore",
      });
      const out = await new Response(proc.stdout).text();
      const code = await proc.exited;
      if (code === 0) return input === null ? out : "";
    } catch {
      /* try next */
    }
  }
  return null;
}

/** Copy via helper tool, else OSC 52. Returns how it was copied. */
export async function copyText(text: string): Promise<"helper" | "osc52" | "failed"> {
  const viaHelper = await tryCmds(COPY_CMDS, text);
  if (viaHelper !== null) return "helper";
  try {
    const b64 = Buffer.from(text, "utf8").toString("base64");
    process.stdout.write(`\x1b]52;c;${b64}\x07`);
    return "osc52";
  } catch {
    return "failed";
  }
}

/** Read clipboard via helper tool. Returns null when no backend exists. */
export async function pasteText(): Promise<string | null> {
  const out = await tryCmds(PASTE_CMDS, null);
  if (out === null) return null;
  return out.replace(/\r?\n$/, "");
}
