import { writable } from 'svelte/store';

export const theme = writable(localStorage.getItem('theme') || 'light');
export const csrf = writable(sessionStorage.getItem('csrf') || '');
export const initialized = writable(false);
export const unlocked = writable(false);

theme.subscribe((v) => localStorage.setItem('theme', v));
csrf.subscribe((v) => {
  if (v) sessionStorage.setItem('csrf', v);
  else sessionStorage.removeItem('csrf');
});