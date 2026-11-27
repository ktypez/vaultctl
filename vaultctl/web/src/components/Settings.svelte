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

<div class="page">
  <div class="settings-page">
    <div class="page-heading">
      <div class="heading-copy">
        <div class="eyebrow"><span class="eyebrow-mark">✳</span> VAULT CONFIGURATION</div>
        <div class="title-line"><h1>Settings</h1><span class="count-badge">SEC / control</span></div>
      </div>
    </div>

    <div class="settings-card">
      <h2>Vault settings</h2>
      <p class="sub">Manage your vault configuration.</p>

      <div class="section">
        <div class="field">
          <label for="settingsCurrent">Current password</label>
          <div class="input-wrap">
            <input id="settingsCurrent" class="input" type={showCur ? 'text' : 'password'} bind:value={current} placeholder="current password" />
            <button type="button" class="pw" onclick={() => (showCur = !showCur)}>{showCur ? 'hide' : 'show'}</button>
          </div>
        </div>
        <div class="field">
          <label for="settingsNew">New password</label>
          <div class="input-wrap">
            <input id="settingsNew" class="input" type={showNew ? 'text' : 'password'} bind:value={newPw} placeholder="new password" minlength="8" />
            <button type="button" class="pw" onclick={() => (showNew = !showNew)}>{showNew ? 'hide' : 'show'}</button>
          </div>
        </div>
        <div class="field">
          <label for="settingsConfirm">Confirm new</label>
          <div class="input-wrap">
            <input id="settingsConfirm" class="input" type={showConfirm ? 'text' : 'password'} bind:value={confirm} placeholder="confirm" minlength="8" />
            <button type="button" class="pw" onclick={() => (showConfirm = !showConfirm)}>{showConfirm ? 'hide' : 'show'}</button>
          </div>
        </div>
        <button class="btn cta" style="width:100%" disabled={loading} onclick={changePw}>Change password</button>
      </div>

      <div class="section">
        <div class="label">Session</div>
        <p class="muted">Sessions are held in memory only and expire after 30 minutes of inactivity. Restarting the service locks the vault.</p>
        <button class="btn danger" style="width:100%" onclick={lock}>Lock vault</button>
        <div style="margin-top:14px"><button class="btn ghost sm" onclick={() => (window.location.hash = '')}>← Back to vault</button></div>
      </div>
    </div>

    <div style="margin-top:24px">
      <div class="label" style="margin-bottom:10px">Audit log (last 100)</div>
      <div class="audit-list">
        {#if auditEntries.length === 0}
          <div class="empty-state" style="padding:24px">
            <div class="ico">✳</div>
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
</div>

{#if toastMsg}
  <div class="toast show" class:err={toastKind === 'err'} class:ok={toastKind === 'ok'} role="status">
    <span aria-hidden="true">{toastKind === 'err' ? '✕' : toastKind === 'ok' ? '✓' : 'ℹ'}</span>
    {toastMsg}
  </div>
{/if}
