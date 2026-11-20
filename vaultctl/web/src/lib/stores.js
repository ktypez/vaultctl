import { writable } from 'svelte/store';

export const theme = writable(localStorage.getItem('theme') || 'light');
export const initialized = writable(false);
export const unlocked = writable(false);

theme.subscribe((v) => localStorage.setItem('theme', v));
