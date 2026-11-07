<script>
  import { api } from '../lib/api.js';
  import { unlocked, csrf } from '../lib/stores.js';

  let password = $state('');
  let remember = $state(false);
  let error = $state('');
  let loading = $state(false);
  let showPw = $state(false);

  async function submit() {
    error = '';
    loading = true;
    const r = await api('/api/unlock', 'POST', { password, remember });
    loading = false;
    if (r.ok) {
      $csrf = r.data.csrf;
      $unlocked = true;
    } else {
      error = r.data?.error || 'Unlock failed';
    }
  }
</script>

<div class="auth">
  <div class="auth-card">
    <div class="mark" style="width:56px;height:56px;border:1px solid var(--line);background:var(--bg2);display:grid;place-items:center;border-radius:var(--radius-sm);margin-bottom:24px">
      <img src="/favicon.svg" alt="" style="width:40px;height:40px;object-fit:contain" />
    </div>
    <h2>Unlock vault</h2>
    <p class="sub">Enter your master password to access your credentials.</p>
    <div class="field">
      <label>Master password</label>
      <div class="input-wrap">
        <input class="input" type={showPw ? 'text' : 'password'} bind:value={password} autocomplete="current-password" placeholder="master password" onkeydown={(e) => { if (e.key === 'Enter') submit(); }} />
        <button type="button" class="pw-toggle" onclick={() => (showPw = !showPw)}>{showPw ? 'hide' : 'show'}</button>
      </div>
    </div>
    <label class="remember"><input type="checkbox" bind:checked={remember} />Remember me (30 days)</label>
    <button class="btn cta" disabled={loading} onclick={submit}>Unlock</button>
    {#if error}<div class="err-msg">{error}</div>{/if}
  </div>
</div>