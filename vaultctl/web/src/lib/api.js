const BASE = (import.meta.env.VITE_VAULT_URL || '').replace(/\/$/, '');

let csrfToken = null;
try {
  csrfToken = sessionStorage.getItem('vaultctl_csrf');
} catch (e) {
  csrfToken = null;
}

export function getCsrf() {
  if (!csrfToken) {
    try { csrfToken = sessionStorage.getItem('vaultctl_csrf'); } catch (e) { /* ignore */ }
  }
  return csrfToken;
}

export function setCsrf(token) {
  csrfToken = token || null;
  try {
    if (token) sessionStorage.setItem('vaultctl_csrf', token);
    else sessionStorage.removeItem('vaultctl_csrf');
  } catch (e) { /* ignore */ }
}

export function clearCsrf() {
  setCsrf(null);
}

export async function api(path, method = 'GET', body = null) {
  const url = BASE + path;
  const headers = {};
  if (body !== null) headers['Content-Type'] = 'application/json';
  if (method !== 'GET') {
    const c = getCsrf();
    if (c) headers['X-CSRF-Token'] = c;
  }
  const opts = {
    method,
    headers,
    credentials: 'include',
  };
  if (body !== null) opts.body = JSON.stringify(body);
  try {
    const res = await fetch(url, opts);
    let data = null;
    try { data = await res.json(); } catch (e) { /* ignore */ }
    if (res.status === 401 || res.status === 403) {
      clearCsrf();
      throw new Error('Session expired — please unlock again');
    }
    return { ok: res.ok, status: res.status, data };
  } catch (e) {
    return { ok: false, status: 0, data: { error: e.message || 'network error' } };
  }
}