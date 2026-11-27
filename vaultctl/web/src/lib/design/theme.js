/** Local mochi theme controller (ported from dashboard/src/lib/design/theme.ts).
 *
 * Same attributes and storage keys as DATA Ledger / Truck:
 *   <html data-theme="light|dark" data-palette="mochi|matcha|soda|sakura">
 *   localStorage: mochi.theme (auto | light | dark), mochi.palette
 * Framework-free, safe to import before mount and during SSR.
 */

const KEY_MODE = 'mochi.theme';
const KEY_PALETTE = 'mochi.palette';
export const MODES = ['auto', 'light', 'dark'];
export const PALETTES = ['mochi', 'matcha', 'soda', 'sakura'];

let preference = 'auto';
let painted = 'light';
let palette = 'mochi';
let mediaQuery = null;
let systemListener = null;
const listeners = new Set();

const hasWindow = () => typeof window !== 'undefined';
const hasDocument = () => typeof document !== 'undefined';

function prefersDark() {
	return hasWindow() && typeof window.matchMedia === 'function' &&
		window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function read() {
	if (!hasWindow()) return { mode: preference, palette };
	try {
		const m = window.localStorage.getItem(KEY_MODE);
		const p = window.localStorage.getItem(KEY_PALETTE);
		return {
			mode: MODES.includes(m) ? m : 'auto',
			palette: PALETTES.includes(p) ? p : 'mochi'
		};
	} catch {
		return { mode: 'auto', palette: 'mochi' };
	}
}

function watchSystem() {
	if (mediaQuery && systemListener && typeof mediaQuery.removeEventListener === 'function') {
		mediaQuery.removeEventListener('change', systemListener);
	}
	mediaQuery = null;
	systemListener = null;
	if (preference !== 'auto' || !hasWindow() || typeof window.matchMedia !== 'function') return;
	mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
	systemListener = () => apply(preference);
	if (typeof mediaQuery.addEventListener === 'function') {
		mediaQuery.addEventListener('change', systemListener);
	}
}

/** Paint without persisting — also used after the pre-paint boot script. */
export function apply(mode = preference, nextPalette = palette) {
	preference = mode;
	palette = nextPalette;
	painted = mode === 'auto' ? (prefersDark() ? 'dark' : 'light') : mode;
	if (hasDocument()) {
		const root = document.documentElement;
		root.setAttribute('data-theme', painted);
		root.setAttribute('data-palette', palette);
		root.style.colorScheme = painted;
		const meta = document.querySelector('meta[name="theme-color"]');
		if (meta) meta.setAttribute('content', painted === 'dark' ? '#17131F' : '#FFFDF8');
	}
	watchSystem();
	for (const listener of listeners) listener();
	return painted;
}

/** Persist and paint a mode preference. */
export function setTheme(mode) {
	if (hasWindow()) {
		try { window.localStorage.setItem(KEY_MODE, mode); } catch { /* ignore */ }
	}
	return apply(mode);
}

/** auto → light → dark → auto, matching DATA Ledger's TopBar control. */
export function toggleTheme() {
	const i = MODES.indexOf(preference);
	return setTheme(MODES[(i + 1) % MODES.length]);
}

export function setPalette(next) {
	if (!PALETTES.includes(next)) return;
	if (hasWindow()) {
		try { window.localStorage.setItem(KEY_PALETTE, next); } catch { /* ignore */ }
	}
	apply(preference, next);
}

export function cyclePalette() {
	setPalette(PALETTES[(PALETTES.indexOf(palette) + 1) % PALETTES.length]);
}

export function initTheme() {
	const saved = read();
	return apply(saved.mode, saved.palette);
}

export function currentMode() { return painted; }
export function currentPalette() { return palette; }
export function currentPreference() { return preference; }

export function onThemeChange(listener) {
	listeners.add(listener);
	return () => listeners.delete(listener);
}
