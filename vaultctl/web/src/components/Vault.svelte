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
    if (!name) { showToast('Name is required', 'err'); return; }
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
  <div class="card">
    <div class="vault-head">
      <div>
        <div class="title">Entries</div>
        <div class="sub">{countText}</div>
      </div>
      <div class="search-wrap">
        <input class="input" type="text" placeholder="search entries…" bind:value={search} autocomplete="off" />
      </div>
      <button class="btn cta sm" onclick={() => (showAdd = true)}>＋ Add entry</button>
    </div>

    {#if entries.length === 0 || (search && filtered.length === 0)}
      <div class="empty-state">
        <div class="ico">🔐</div>
        <p>
          {#if entries.length === 0}
            No entries yet. Click "Add entry" to create your first one.
          {:else}
            No entries match "{search}".
          {/if}
        </p>
      </div>
    {:else}
      <div class="entries">
        {#each filtered as name, i (name)}
          {@const isRev = revealed.has(name)}
          <div class="entry" style="animation-delay:{i * 40}ms">
            <div class="entry-top">
              <div class="entry-key">{name}</div>
            </div>
            <div style="display:flex; gap:14px; align-items:center; justify-content:space-between; grid-column: 1 / -1;">
              <div class="entry-val" class:revealed={isRev}>{isRev ? (values[name] ?? '') : '••••••••'}</div>
              <div class="entry-actions">
                <button class="icon-btn" title={isRev ? 'hide' : 'reveal'} onclick={() => reveal(name)}>
                  {isRev ? '🙈' : '👁'}
                </button>
                <button class="icon-btn" title="copy" disabled={!isRev} onclick={() => copy(name)}>⧉</button>
                <button class="icon-btn danger" title="delete" onclick={() => remove(name)}>✕</button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

{#if showAdd}
  <div class="modal">
    <div class="modal-backdrop" onclick={() => (showAdd = false)}></div>
    <div class="modal-card card">
      <div class="vault-head">
        <div class="title">Add / update entry</div>
        <button class="icon-btn" onclick={() => (showAdd = false)}>✕</button>
      </div>
      <div style="padding:22px">
        <div class="field">
          <label>Name</label>
          <input class="input mono" bind:value={newName} placeholder="e.g. github_api_key" />
        </div>
        <div class="field">
          <label>Value</label>
          <input class="input mono" bind:value={newValue} placeholder="secret value" />
        </div>
        <div style="display:flex; gap:10px; margin-top:20px">
          <button class="btn cta" style="flex:1" onclick={save}>Save entry</button>
          <button class="btn ghost" style="flex:1" onclick={() => (showAdd = false)}>Cancel</button>
        </div>
      </div>
    </div>
  </div>
{/if}

{#if toastMsg}
  <div class="toast" class:show={toastMsg} class:err={toastKind === 'err'} class:ok={toastKind === 'ok'}>
    {#if toastKind === 'err'}✕{/if}
    {#if toastKind === 'ok'}✓{/if}
    {toastMsg}
  </div>
{/if}