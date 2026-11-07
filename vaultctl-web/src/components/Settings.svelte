<script>
  import { api } from '../lib/api.js';
  import { unlocked } from '../lib/stores.js';

  let current = $state('');
  let newPw = $state('');
  let confirm = $state('');
  let loading = $state(false);
  let auditEntries = $state([]);
  let showCur = $state(false);
  let showNew = $state(false);
  let showConfirm = $state(false);
  let toastMsg = $state('');
  let toastKind = $state('');
  let toastTimer;

  function showToast(msg, kind = '') {
    toastMsg = msg;
    toastKind = kind;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toastMsg = ''; toastKind = ''; }, 2600);
  }

  async function loadAudit() {
    const r = await api('/api/audit?n=100');
    if (r.ok) auditEntries = r.data.entries || [];
    else showToast(r.data?.error || 'Failed to load audit', 'err');
  }

  $effect(() => {
    if ($unlocked) loadAudit();
  });

  async function changePw() {
    if (newPw !== confirm) { showToast('New passwords do not match', 'err'); return; }
    if (newPw.length < 8) { showToast('Password must be at least 8 characters', 'err'); return; }
    loading = true;
    const r = await api('/api/change-password', 'POST', { current, new: newPw, confirm });
    loading = false;
    if (r.ok) {
      showToast('Password changed', 'ok');
      current = '';
      newPw = '';
      confirm = '';
    } else {
      showToast(r.data?.error || 'Failed', 'err');
    }
  }

  async function lock() {
    await api('/api/lock', 'POST');
    $unlocked = false;
    window.location.hash = '';
  }
</script>

<div class="settings">
  <div class="auth-card" style="max-width:620px;margin:0 auto">
    <div class="mark" style="width:56px;height:56px;border:1px solid var(--line);background:var(--bg2);display:grid;place-items:center;border-radius:var(--radius-sm);margin-bottom:24px">
      <img src="/favicon.svg" alt="" style="width:40px;height:40px;object-fit:contain" />
    </div>
    <h2>Settings</h2>
    <p class="sub">Manage your vault configuration.</p>

    <div class="field" style="margin-top:20px">
      <label>Current password</label>
      <div class="input-wrap">
        <input class="input" type={showCur ? 'text' : 'password'} bind:value={current} placeholder="current password" />
        <button type="button" class="pw-toggle" onclick={() => (showCur = !showCur)}>{showCur ? 'hide' : 'show'}</button>
      </div>
    </div>
    <div class="field">
      <label>New password</label>
      <div class="input-wrap">
        <input class="input" type={showNew ? 'text' : 'password'} bind:value={newPw} placeholder="new password" minlength="8" />
        <button type="button" class="pw-toggle" onclick={() => (showNew = !showNew)}>{showNew ? 'hide' : 'show'}</button>
      </div>
    </div>
    <div class="field">
      <label>Confirm new</label>
      <div class="input-wrap">
        <input class="input" type={showConfirm ? 'text' : 'password'} bind:value={confirm} placeholder="confirm" minlength="8" />
        <button type="button" class="pw-toggle" onclick={() => (showConfirm = !showConfirm)}>{showConfirm ? 'hide' : 'show'}</button>
      </div>
    </div>
    <button class="btn cta" style="width:100%" disabled={loading} onclick={changePw}>Change password</button>

    <div class="label" style="margin:28px 0 12px">Session</div>
    <p class="sub" style="margin-bottom:16px">Sessions persist across restarts. Tick "Remember me" on the unlock screen to stay signed in for 30 days.</p>
    <button class="btn danger" style="width:100%" onclick={lock}>Lock vault</button>
    <div style="margin-top:14px"><button class="btn ghost sm" onclick={() => (window.location.hash = '')}>← Back to vault</button></div>
  </div>

  <div style="margin-top:24px">
    <div class="label" style="margin-bottom:10px">Audit log (last 100)</div>
    <div class="audit-list">
      {#if auditEntries.length === 0}
        <div class="empty-state" style="padding:24px">
          <div class="ico">📜</div>
          <p>No audit entries.</p>
        </div>
      {/if}
      {#each auditEntries as row}
        <div class="audit-row">
          <div class="ts">{row.ts}</div>
          <div class="act">{row.action}</div>
          <div class="name">{row.name ?? ''}</div>
          <div class="reason">{row.decision} {row.reason || ''}</div>
        </div>
      {/each}
    </div>
  </div>
</div>

{#if toastMsg}
  <div class="toast" class:show={toastMsg} class:err={toastKind === 'err'} class:ok={toastKind === 'ok'}>
    {#if toastKind === 'err'}✕{/if}
    {#if toastKind === 'ok'}✓{/if}
    {toastMsg}
  </div>
{/if}