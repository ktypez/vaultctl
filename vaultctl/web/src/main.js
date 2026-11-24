import '@mcky/ui/v6-base.css';
import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import {
  V6_MODE_STORAGE_KEY,
  applyV6Mode,
  onV6ModeChange,
} from '@mcky/ui/theme';

// Existing installations stored the preference under `theme`; move it to the
// shared key once so the V6 ramp and scenery start in the visitor's mode.
try {
  const legacy = localStorage.getItem('theme');
  if ((legacy === 'light' || legacy === 'dark') && !localStorage.getItem(V6_MODE_STORAGE_KEY)) {
    localStorage.setItem(V6_MODE_STORAGE_KEY, legacy);
    localStorage.removeItem('theme');
  }
} catch {
  // Storage may be blocked; the helper still resolves the current OS mode.
}

const themeColor = document.querySelector('meta[name="theme-color"]');
function syncThemeColor(mode) {
  themeColor?.setAttribute('content', mode === 'dark' ? '#1b2440' : '#9dd2f7');
}

// Apply before mounting so the shared CSS ramp and scenery agree on first paint.
syncThemeColor(applyV6Mode());
onV6ModeChange(syncThemeColor);

const target = document.getElementById('app');
if (!target) throw new Error('missing #app mount point');
const app = mount(App, { target });
export default app;
