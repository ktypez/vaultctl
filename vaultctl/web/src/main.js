import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import { initTheme } from './lib/design/theme.js';

// index.html paints the saved palette/mode before CSS loads; re-apply here so
// the controller, <html> attributes and theme-color meta stay in sync.
initTheme();

const target = document.getElementById('app');
if (!target) throw new Error('missing #app mount point');
const app = mount(App, { target });
export default app;
