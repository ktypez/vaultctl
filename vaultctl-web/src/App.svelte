<script>
  import { onMount } from 'svelte';
  import { fade, fly, scale } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { quintOut, cubicOut } from 'svelte/easing';
  import { api, getCsrf, setCsrf, clearCsrf } from './lib/api.js';

  let theme = $state('light');
  let initialized = $state(false);
  let unlocked = $state(false);
  let route = $state('vault');

  let entries = $state([]);
  let revealed = $state({});
  let values = $state({});
  let search = $state('');

  let toast = $state({ msg: '', kind: '', visible: false });
  let toastTimer;

  // auth forms
  let showSetupPw = $state(false);
  let showSetupPw2 = $state(false);
  let setupPw = $state('');
  let setupPw2 = $state('');
  let setupErr = $state('');

  let showUnlockPw = $state(false);
  let unlockPw = $state('');
  let remember = $state(false);
  let unlockErr = $state('');

  // add modal
  let addModalOpen = $state(false);
  let newName = $state('');
  let newValue = $state('');

  // settings
  let showCpCur = $state(false);
  let showCpNew = $state(false);
  let showCpNew2 = $state(false);
  let cpCur = $state('');
  let cpNew = $state('');
  let cpNew2 = $state('');
  let audit = $state([]);

  let loadingEntries = $state(false);

  function showToast(msg, kind = '') {
    toast = { msg, kind, visible: true };
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
      toast = { ...toast, visible: false };
    }, 2600);
  }

  function applyTheme(t) {
    theme = t;
    document.documentElement.setAttribute('data-theme', t);
    try { localStorage.setItem('theme', t); } catch (e) { /* ignore */ }
  }
  function toggleTheme() {
    applyTheme(theme === 'dark' ? 'light' : 'dark');
  }

  function syncRoute() {
    route = (typeof location !== 'undefined' && location.hash === '#/settings') ? 'settings' : 'vault';
    if (unlocked && route === 'settings') loadAudit();
  }

  async function refreshStatus() {
    const s = await api('/api/status');
    if (s.data) {
      initialized = !!s.data.initialized;
      unlocked = !!s.data.unlocked;
    }
  }

  async function refreshEntries() {
    loadingEntries = true;
    const res = await api('/api/entries');
    entries = (res.data && res.data.entries) || [];
    loadingEntries = false;
  }

  async function doSetup() {
    setupErr = '';
    if (setupPw.length < 8) { setupErr = 'Password must be at least 8 characters'; return; }
    if (setupPw !== setupPw2) { setupErr = 'Passwords do not match'; return; }
    const r = await api('/api/setup', 'POST', { password: setupPw, confirm: setupPw2 });
    if (r.ok) {
      if (r.data && r.data.csrf) setCsrf(r.data.csrf);
      initialized = true;
      unlocked = true;
      setupPw = ''; setupPw2 = '';
      await refreshEntries();
      showToast('Vault created', 'ok');
    } else {
      setupErr = (r.data && r.data.error) || 'error';
    }
  }

  async function doUnlock() {
    unlockErr = '';
    const r = await api('/api/unlock', 'POST', { password: unlockPw, remember });
    if (r.ok) {
      if (r.data && r.data.csrf) setCsrf(r.data.csrf);
      unlocked = true;
      unlockPw = '';
      await refreshEntries();
      showToast('Unlocked', 'ok');
    } else {
      unlockErr = (r.data && r.data.error) || 'error';
    }
  }

  async function doLock() {
    await api('/api/lock', 'POST');
    clearCsrf();
    unlocked = false;
    revealed = {};
    values = {};
    entries = [];
    audit = [];
    if (location.hash === '#/settings') location.hash = '';
    showToast('Locked', 'ok');
  }

  let filtered = $derived.by(() => {
    const q = search.trim().toLowerCase();
    if (!q) return entries;
    return entries.filter((name) => {
      if (name.toLowerCase().includes(q)) return true;
      if (revealed[name] && (values[name] || '').toLowerCase().includes(q)) return true;
      return false;
    });
  });

  async function toggleReveal(name) {
    if (revealed[name]) {
      revealed = { ...revealed, [name]: false };
      return;
    }
    const r = await api('/api/entries/' + encodeURIComponent(name));
    if (r.ok) {
      values = { ...values, [name]: r.data.value };
      revealed = { ...revealed, [name]: true };
    } else {
      showToast((r.data && r.data.error) || 'error', 'err');
    }
  }

  async function copyValue(name) {
    const value = values[name];
    if (!value) return;
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(value);
      } else {
        const ta = document.createElement('textarea');
        ta.value = value;
        ta.setAttribute('readonly', '');
        ta.style.position = 'absolute';
        ta.style.left = '-9999px';
        ta.style.top = '0';
        document.body.appendChild(ta);
        ta.focus();
        ta.select();
        ta.setSelectionRange(0, value.length);
        const ok = document.execCommand('copy');
        document.body.removeChild(ta);
        if (!ok) throw new Error('execCommand returned false');
      }
      showToast('copied to clipboard', 'ok');
    } catch (e) {
      // Last resort: let the user copy manually
      try { window.prompt('Copy this value:', value); } catch (_) { /* ignore */ }
      showToast('copy failed — value shown for manual copy', 'err');
    }
  }

  async function deleteEntry(name) {
    if (!confirm("Delete entry '" + name + "'?")) return;
    const r = await api('/api/entries/' + encodeURIComponent(name), 'DELETE');
    if (r.ok) {
      const nr = { ...revealed }; delete nr[name]; revealed = nr;
      const nv = { ...values }; delete nv[name]; values = nv;
      showToast('deleted', 'ok');
      await refreshEntries();
    } else {
      showToast((r.data && r.data.error) || 'error', 'err');
    }
  }

  async function saveEntry() {
    const name = newName.trim();
    if (!name) { showToast('Name is required', 'err'); return; }
    if (newValue === '') { showToast('Value is required', 'err'); return; }
    const csrf = getCsrf();
    if (!csrf) {
      showToast('Session expired — please unlock again', 'err');
      return;
    }
    const r = await api('/api/entries', 'POST', { name, value: newValue });
    if (r.ok) {
      addModalOpen = false;
      newName = '';
      newValue = '';
      showToast('Saved', 'ok');
      await refreshEntries();
    } else {
      showToast((r.data && r.data.error) || 'error', 'err');
    }
  }

  async function changePassword() {
    if (cpNew !== cpNew2) { showToast('New passwords do not match', 'err'); return; }
    const r = await api('/api/change-password', 'POST', { current: cpCur, new: cpNew, confirm: cpNew2 });
    if (r.ok) {
      cpCur = ''; cpNew = ''; cpNew2 = '';
      showToast('Password changed', 'ok');
    } else {
      showToast((r.data && r.data.error) || 'error', 'err');
    }
  }

  async function loadAudit() {
    const r = await api('/api/audit?n=100');
    if (r.ok) audit = (r.data && r.data.entries) || [];
    else showToast((r.data && r.data.error) || 'error', 'err');
  }

  onMount(async () => {
    try {
      const saved = localStorage.getItem('theme');
      if (saved) theme = saved;
      else if (window.matchMedia('(prefers-color-scheme: dark)').matches) theme = 'dark';
    } catch (e) { /* ignore */ }
    applyTheme(theme);

    syncRoute();
    window.addEventListener('hashchange', syncRoute);

    await refreshStatus();
    if (unlocked) await refreshEntries();
  });
</script>

<header class="topbar">
  <div class="inner">
    <div class="brand">
      <span class="seal">V</span>
      <div class="name">vaultctl <span class="jp">web</span></div>
    </div>
    <div class="actions">
      {#if initialized}
        <span class="status {unlocked ? 'ok' : 'warn'}">
          <span class="dot"></span>{unlocked ? 'Unlocked' : 'Locked'}
        </span>
      {/if}
      <button class="ghost" onclick={toggleTheme}>{theme === 'dark' ? 'Light' : 'Dark'}</button>
      {#if unlocked}
        <a class="ghost" href="#/settings">Settings</a>
        <button class="ghost danger" onclick={doLock}>Lock</button>
      {/if}
    </div>
  </div>
</header>

<main>
  {#if !initialized}
    <section class="auth auth-setup" transition:fade={{ duration: 200 }}>
      <div class="auth-floater floater-one">PRIVATE / LOCAL</div>
      <div class="auth-floater floater-two">✳ AES-256</div>
      <div class="card auth-card">
        <div class="auth-stamp"><span>V</span><i>✳</i></div>
        <div class="eyebrow"><span class="eyebrow-mark">✳</span> YOUR PRIVATE KEYRING</div>
        <h2>Create your vault</h2>
        <svg class="squiggle auth-squiggle" viewBox="0 0 148 8" aria-hidden="true"><path d="M2 5c10-5 16-5 26 0s16 5 26 0 16-5 26 0 16 5 26 0 16-5 26 0 12 4 14 2" /></svg>
        <p class="sub">Choose a strong master password. It is never stored — lose it and the data is unrecoverable.</p>
        <div class="field">
          <label for="setupPw">Master password</label>
          <div class="wrap">
            <input id="setupPw" type={showSetupPw ? 'text' : 'password'} bind:value={setupPw} class="input" autocomplete="new-password" placeholder="at least 8 characters" />
            <button type="button" class="pw" onclick={() => (showSetupPw = !showSetupPw)}>{showSetupPw ? 'hide' : 'show'}</button>
          </div>
        </div>
        <div class="field">
          <label for="setupPw2">Confirm password</label>
          <div class="wrap">
            <input id="setupPw2" type={showSetupPw2 ? 'text' : 'password'} bind:value={setupPw2} class="input" autocomplete="new-password" placeholder="repeat password" />
            <button type="button" class="pw" onclick={() => (showSetupPw2 = !showSetupPw2)}>{showSetupPw2 ? 'hide' : 'show'}</button>
          </div>
        </div>
        <button class="btn cta" onclick={doSetup}>Create vault</button>
        {#if setupErr}<div class="err">{setupErr}</div>{/if}
      </div>
    </section>

  {:else if !unlocked}
    <section class="auth auth-unlock" transition:fade={{ duration: 200 }}>
      <div class="auth-floater floater-one">PRIVATE / LOCAL</div>
      <div class="auth-floater floater-two">✳ AES-256</div>
      <div class="card auth-card">
        <div class="auth-stamp"><span>V</span><i>✳</i></div>
        <div class="eyebrow"><span class="eyebrow-mark">✳</span> SECURE ACCESS / MASTER KEY</div>
        <h2>Open your vault</h2>
        <svg class="squiggle auth-squiggle" viewBox="0 0 148 8" aria-hidden="true"><path d="M2 5c10-5 16-5 26 0s16 5 26 0 16-5 26 0 16 5 26 0 16-5 26 0 12 4 14 2" /></svg>
        <p class="sub">Enter your master password to access your credentials.</p>
        <div class="field">
          <label for="unlockPw">Master password</label>
          <div class="wrap">
            <input id="unlockPw" type={showUnlockPw ? 'text' : 'password'} bind:value={unlockPw} class="input" autocomplete="current-password" placeholder="master password" onkeydown={(e) => { if (e.key === 'Enter') doUnlock(); }} />
            <button type="button" class="pw" onclick={() => (showUnlockPw = !showUnlockPw)}>{showUnlockPw ? 'hide' : 'show'}</button>
          </div>
        </div>
        <label class="remember"><input type="checkbox" bind:checked={remember} />Remember me (30 days)</label>
        <button class="btn cta" onclick={doUnlock}>Unlock</button>
        {#if unlockErr}<div class="err">{unlockErr}</div>{/if}
      </div>
    </section>

  {:else if route === 'settings'}
    <section class="page" transition:fade={{ duration: 200 }}>
      <div class="settings-page">
        <div class="page-heading settings-heading"><div class="heading-copy"><div class="eyebrow"><span class="eyebrow-mark">✳</span> VAULT CONFIGURATION</div><div class="title-line"><h1>Settings</h1><span class="count-sticker"><strong>SEC</strong><span>control</span></span></div><svg class="squiggle" viewBox="0 0 148 8" aria-hidden="true"><path d="M2 5c10-5 16-5 26 0s16 5 26 0 16-5 26 0 16-5 26 0 16-5 26 0 12 4 14 2" /></svg></div></div>
        <div class="settings-card">
        <h2>Vault settings</h2>
        <p class="sub">Manage your vault configuration and audit history.</p>

        <div class="section">
          <div class="label">Change master password</div>
          <div class="field">
            <label for="cpCur">Current password</label>
            <div class="wrap">
              <input id="cpCur" type={showCpCur ? 'text' : 'password'} bind:value={cpCur} class="input" placeholder="current password" />
              <button type="button" class="pw" onclick={() => (showCpCur = !showCpCur)}>{showCpCur ? 'hide' : 'show'}</button>
            </div>
          </div>
          <div class="field">
            <label for="cpNew">New password</label>
            <div class="wrap">
              <input id="cpNew" type={showCpNew ? 'text' : 'password'} bind:value={cpNew} class="input" placeholder="new password" />
              <button type="button" class="pw" onclick={() => (showCpNew = !showCpNew)}>{showCpNew ? 'hide' : 'show'}</button>
            </div>
          </div>
          <div class="field">
            <label for="cpNew2">Confirm new</label>
            <div class="wrap">
              <input id="cpNew2" type={showCpNew2 ? 'text' : 'password'} bind:value={cpNew2} class="input" placeholder="confirm" />
              <button type="button" class="pw" onclick={() => (showCpNew2 = !showCpNew2)}>{showCpNew2 ? 'hide' : 'show'}</button>
            </div>
          </div>
          <button class="btn cta" onclick={changePassword} style="width:100%">Change password</button>
        </div>

        <div class="section">
          <div class="label">Session</div>
          <p class="muted">Sessions persist across restarts. Tick "Remember me" on the unlock screen to stay signed in for 30 days.</p>
          <button class="btn danger" onclick={doLock} style="width:100%">Lock vault</button>
        </div>

        <div class="section">
          <div class="label">Audit log (last 100)</div>
          <div class="entries">
            {#each audit as a (a.ts + (a.name || '') + (a.action || ''))}
              <div class="row" in:fly={{ y: 6, duration: 180 }}>
                <div class="time">{a.ts}</div>
                <div class="desc">{a.action} {a.name ? '· ' + a.name : ''} <span class="decision">— {a.decision}{a.reason ? ' (' + a.reason + ')' : ''}</span></div>
              </div>
            {/each}
            {#if audit.length === 0}
              <div class="empty"><div class="ico">📜</div><p>No audit entries.</p></div>
            {/if}
          </div>
        </div>

        <div style="margin-top:18px"><a class="btn ghost" href="#/">← Back to vault</a></div>
        </div>
      </div>
    </section>

  {:else}
    <section class="page vault-page" transition:fade={{ duration: 200 }}>
      <div class="page-heading">
        <div class="heading-copy">
          <div class="eyebrow"><span class="eyebrow-mark">✳</span> PRIVATE CREDENTIALS / LOCAL VAULT</div>
          <div class="title-line">
            <h1>Keys to the vault</h1>
            <span class="count-sticker"><strong>{entries.length}</strong><span>entries</span></span>
          </div>
          <svg class="squiggle" viewBox="0 0 148 8" aria-hidden="true"><path d="M2 5c10-5 16-5 26 0s16 5 26 0 16-5 26 0 16 5 26 0 16-5 26 0 12 4 14 2" /></svg>
          <p class="heading-sub">Encrypted credentials, kept in one quiet place.</p>
        </div>
        <button class="btn cta add-button" onclick={() => (addModalOpen = true)}><span class="plus">＋</span> Add entry</button>
      </div>

      <div class="vault-summary" aria-label="Vault summary">
        <div class="summary-cell"><span class="summary-label">Stored entries</span><strong>{entries.length}</strong></div>
        <span class="summary-divider"></span>
        <div class="summary-cell"><span class="summary-label">Visible now</span><strong>{Object.values(revealed).filter(Boolean).length}</strong></div>
        <span class="summary-divider"></span>
        <div class="summary-cell encryption"><span class="summary-label">Storage</span><strong><span class="live-dot"></span> AES-256 / ENCRYPTED</strong></div>
      </div>

      <div class="vault-workspace">
        <div class="vault-main">
          <div class="toolbar">
            <div class="search">
              <span class="search-mark" aria-hidden="true">⌕</span>
              <input type="text" bind:value={search} class="input" placeholder="Search entries..." autocomplete="off" aria-label="Search entries" />
              {#if search}<button class="clear" onclick={() => (search = '')} aria-label="Clear search">✕</button>{/if}
            </div>
            <div class="count">SHOWING <strong>{filtered.length}</strong> / {entries.length}</div>
          </div>

          <div class="list">
            <div class="list-head"><span>ENTRY / KEY</span><span>SECRET VALUE</span><span>ACTIONS</span></div>
            {#each filtered as name, i (name)}
              <div class="row" animate:flip={{ duration: 250 }} in:fly={{ y: 8, duration: 200, delay: i * 20 }} out:fade={{ duration: 120 }}>
                <div class="cell">
                  <span class="key-mark" aria-hidden="true">K</span>
                  <div class="key-copy"><div class="name">{name}</div><div class="entry-meta">SECRET <span>·</span> ENCRYPTED</div></div>
                </div>
                <div class="value-cell"><div class="value {revealed[name] ? 'show' : ''}">{revealed[name] ? (values[name] ?? '••••••••') : '••••••••••••••••'}</div></div>
                <div class="actions">
                  <button class="icon" title={revealed[name] ? 'hide' : 'reveal'} aria-label={revealed[name] ? 'Hide secret' : 'Reveal secret'} onclick={() => toggleReveal(name)}>{revealed[name] ? '◉' : '◎'}</button>
                  <button class="icon" title="copy" aria-label="Copy secret" disabled={!revealed[name]} onclick={() => copyValue(name)}>⧉</button>
                  <button class="icon danger" title="delete" aria-label="Delete entry" onclick={() => deleteEntry(name)}>×</button>
                </div>
              </div>
            {/each}
          </div>

          {#if entries.length === 0 || (search.trim() && filtered.length === 0)}
            <div class="empty">
              <div class="empty-sticker">✳</div>
              <strong>{search.trim() && filtered.length === 0 ? 'No matching entries' : 'The vault is still empty'}</strong>
              <p>{search.trim() && filtered.length === 0 ? 'Try a different key name.' : 'Add your first credential to start the collection.'}</p>
              {#if entries.length === 0}<button class="btn line-button" onclick={() => (addModalOpen = true)}>＋ Add first entry</button>{/if}
            </div>
          {/if}
        </div>

        <aside class="vault-aside">
          <div class="aside-card vault-sticker">
            <div class="aside-topline"><span>VAULT STATUS</span><span class="aside-star">✳</span></div>
            <div class="seal-graphic" aria-hidden="true"><span>V</span><i></i></div>
            <h2>Locked by design.</h2>
            <p>Secrets stay masked until you choose to reveal them. Lock the vault when you’re finished.</p>
            <div class="aside-rule"></div>
            <div class="aside-fact"><span>ENCRYPTION</span><strong>AES-256</strong></div>
            <div class="aside-fact"><span>SESSION</span><strong>{unlocked ? 'UNLOCKED' : 'LOCKED'}</strong></div>
          </div>
          <div class="aside-note"><span class="note-pin">✳</span><span>Keep your master password safe. It cannot be recovered.</span></div>
        </aside>
      </div>
    </section>
  {/if}
</main>

{#if addModalOpen}
  <div class="modal">
    <div class="backdrop" role="button" tabindex="0" onclick={() => (addModalOpen = false)} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') addModalOpen = false; }} transition:fade={{ duration: 150 }}></div>
    <div class="card modal-card" transition:scale={{ duration: 200, start: 0.96, opacity: 0 }}>
      <div class="vault-head">
        <div class="title">Add / update entry</div>
        <button class="icon" onclick={() => (addModalOpen = false)} title="close">✕</button>
      </div>
      <div class="body">
        <div class="field">
          <label for="newName">Name</label>
          <input id="newName" type="text" bind:value={newName} class="input mono" placeholder="e.g. github_api_key" />
        </div>
        <div class="field">
          <label for="newValue">Value</label>
          <input id="newValue" type="text" bind:value={newValue} class="input mono" placeholder="secret value" />
        </div>
        <div class="actions">
          <button class="btn cta" onclick={saveEntry}>Save entry</button>
          <button class="btn ghost" onclick={() => (addModalOpen = false)}>Cancel</button>
        </div>
      </div>
    </div>
  </div>
{/if}

<div class="toast {toast.visible ? 'show' : ''} {toast.kind}">{#if toast.visible}<span>{toast.kind === 'err' ? '✕' : toast.kind === 'ok' ? '✓' : 'ℹ'}</span><span>{toast.msg}</span>{/if}</div>
