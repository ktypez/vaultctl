<script>
  import { api } from '../lib/api.js';
  import { unlocked } from '../lib/stores.js';

  let entries = $state([]);
  let search = $state('');
  let revealed = $state(new Set());
  let values = $state({});
  let showAdd = $state(false);
  let newName = $state('');
  let newValue = $state('');
  let toastMsg = $state('');
  let toastKind = $state('');
  let toastTimer;

  const NAME_RE = /^[A-Za-z0-9_.\-]+$/;
  const NAME_MAX = 128;
  function validName(name) {
    return name.length >= 1 && name.length <= NAME_MAX && NAME_RE.test(name);
  }

  function showToast(msg, kind = '') {
    toastMsg = msg;
    toastKind = kind;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toastMsg = ''; toastKind = ''; }, 2600);
  }

  async function refresh() {
    const r = await api('/api/entries');
    if (r.ok) entries = r.data.entries || [];
    else showToast(r.data?.error || 'Failed to load entries', 'err');
  }

  $effect(() => {
    if ($unlocked) refresh();
  });

  let filtered = $derived(entries.filter((name) => {
    if (!search) return true;
    const q = search.toLowerCase();
    if (name.toLowerCase().includes(q)) return true;
    if (revealed.has(name) && (values[name] || '').toLowerCase().includes(q)) return true;
    return false;
  }));

  let countText = $derived(search ? `${filtered.length} of ${entries.length} entries` : `${entries.length} entries`);

  async function reveal(name) {
    if (revealed.has(name)) {
      revealed.delete(name);
      return;
    }
    const r = await api('/api/entries/' + encodeURIComponent(name));
    if (r.ok) {
      values[name] = r.data.value;
      revealed.add(name);
    } else {
      showToast(r.data?.error || 'Failed to reveal', 'err');
    }
  }

  async function copy(name) {
    try {
      await navigator.clipboard.writeText(values[name] || '');
      showToast('copied to clipboard', 'ok');
    } catch (e) {
      showToast('copy failed', 'err');
    }
  }

  async function remove(name) {
    if (!confirm("Delete entry '" + name + "'?")) return;
    const r = await api('/api/entries/' + encodeURIComponent(name), 'DELETE');
    if (r.ok) {
      revealed.delete(name);
      delete values[name];
      showToast('deleted', 'ok');
      refresh();
    } else {
      showToast(r.data?.error || 'Failed to delete', 'err');
    }
  }

  async function save() {
    const name = newName.trim();
    const value = newValue;
    if (!validName(name)) { showToast('Name must be letters, numbers, . - _ (1-128 chars)', 'err'); return; }
    if (value === '') { showToast('Value is required', 'err'); return; }
    const r = await api('/api/entries', 'POST', { name, value });
    if (r.ok) {
      showAdd = false;
      newName = '';
      newValue = '';
      showToast('Saved', 'ok');
      refresh();
    } else {
      showToast(r.data?.error || 'Failed to save', 'err');
    }
  }
</script>

<div class="vault-wrap">
  <div class="page-heading">
    <div class="heading-copy">
      <div class="eyebrow"><span class="eyebrow-mark">✳</span> PRIVATE CREDENTIALS / LOCAL VAULT</div>
      <div class="title-line"><h1>Keys to the vault</h1><span class="count-badge">{countText}</span></div>
    </div>
  </div>
  <div class="card">
    <div class="vault-head">
      <div class="title">Entries</div>
      <div class="search-wrap">
        <input class="input" type="text" placeholder="search entries…" bind:value={search} autocomplete="off" aria-label="Search entries" />
      </div>
      <button class="btn cta sm" onclick={() => (showAdd = true)}>＋ Add entry</button>
    </div>

    {#if entries.length === 0 || (search && filtered.length === 0)}
      <div class="empty-state">
        <div class="ico">✳</div>
        <p>
          {#if entries.length === 0}
            No entries yet. Click "Add entry" to create your first one.
          {:else}
            No entries match "{search}".
          {/if}
        </p>
      </div>
    {:else}
      <div class="list">
        <div class="list-head"><span>ENTRY / KEY</span><span>SECRET VALUE</span><span>ACTIONS</span></div>
        {#each filtered as name, i (name)}
          {@const isRev = revealed.has(name)}
          <div class="row" style="animation-delay:{i * 40}ms">
            <div class="cell">
              <span class="key-mark" aria-hidden="true">K</span>
              <div class="key-copy"><div class="name">{name}</div><div class="entry-meta">SECRET <span>·</span> ENCRYPTED</div></div>
            </div>
            <div class="value-cell"><div class="value {isRev ? 'show' : ''}">{isRev ? (values[name] ?? '') : '••••••••'}</div></div>
            <div class="actions">
              <button class="icon" title={isRev ? 'hide' : 'reveal'} aria-label={isRev ? 'Hide secret' : 'Reveal secret'} onclick={() => reveal(name)}>{isRev ? '◉' : '◎'}</button>
              <button class="icon" title="copy" aria-label="Copy secret" disabled={!isRev} onclick={() => copy(name)}>⧉</button>
              <button class="icon danger" title="delete" aria-label="Delete entry" onclick={() => remove(name)}>×</button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

{#if showAdd}
  <div class="modal">
    <div class="modal-backdrop" role="button" tabindex="0" aria-label="Close dialog" onclick={() => (showAdd = false)} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showAdd = false; }}></div>
    <div class="modal-card">
      <div class="vault-head">
        <div class="title">Add / update entry</div>
        <button class="icon" aria-label="Close dialog" onclick={() => (showAdd = false)}>✕</button>
      </div>
      <div class="body">
        <div class="field">
          <label for="vaultNewName">Name</label>
          <input id="vaultNewName" class="input mono" bind:value={newName} placeholder="e.g. github_api_key" />
        </div>
        <div class="field">
          <label for="vaultNewValue">Value</label>
          <input id="vaultNewValue" type="password" class="input mono" bind:value={newValue} placeholder="secret value" />
        </div>
        <div class="actions">
          <button class="btn cta" onclick={save}>Save entry</button>
          <button class="btn ghost" onclick={() => (showAdd = false)}>Cancel</button>
        </div>
      </div>
    </div>
  </div>
{/if}

{#if toastMsg}
  <div class="toast show" class:err={toastKind === 'err'} class:ok={toastKind === 'ok'} role="status">
    <span aria-hidden="true">{toastKind === 'err' ? '✕' : toastKind === 'ok' ? '✓' : 'ℹ'}</span>
    {toastMsg}
  </div>
{/if}
