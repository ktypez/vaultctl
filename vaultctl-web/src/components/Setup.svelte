<script>
  import { api } from '../lib/api.js';
  import { initialized, unlocked, csrf } from '../lib/stores.js';

  let password = $state('');
  let confirm = $state('');
  let error = $state('');
  let loading = $state(false);
  let showPw = $state(false);
  let showPw2 = $state(false);

  async function submit() {
    error = '';
    if (password.length < 8) { error = 'Password must be at least 8 characters'; return; }
    if (password !== confirm) { error = 'Passwords do not match'; return; }
    loading = true;
    const r = await api('/api/setup', 'POST', { password, confirm });
    loading = false;
    if (r.ok) {
      $csrf = r.data.csrf;
      $initialized = true;
      $unlocked = true;
    } else {
      error = r.data?.error || 'Setup failed';
    }
  }
</script>

<div class="auth">
  <div class="auth-card">
    <div class="mark" style="width:56px;height:56px;border:1px solid var(--line);background:var(--bg2);display:grid;place-items:center;border-radius:var(--radius-sm);margin-bottom:24px">
      <img src="/favicon.svg" alt="" style="width:40px;height:40px;object-fit:contain" />
    </div>
    <h2>Create your vault</h2>
    <p class="sub">Choose a strong master password. It is never stored — lose it and the data is unrecoverable.</p>
    <div class="field">
      <label>Master password</label>
      <div class="input-wrap">
        <input class="input" type={showPw ? 'text' : 'password'} bind:value={password} autocomplete="new-password" placeholder="at least 8 characters" minlength="8" />
        <button type="button" class="pw-toggle" onclick={() => (showPw = !showPw)}>{showPw ? 'hide' : 'show'}</button>
      </div>
    </div>
    <div class="field">
      <label>Confirm password</label>
      <div class="input-wrap">
        <input class="input" type={showPw2 ? 'text' : 'password'} bind:value={confirm} autocomplete="new-password" placeholder="repeat password" minlength="8" />
        <button type="button" class="pw-toggle" onclick={() => (showPw2 = !showPw2)}>{showPw2 ? 'hide' : 'show'}</button>
      </div>
    </div>
    <button class="btn cta" disabled={loading} onclick={submit}>Create vault</button>
    {#if error}<div class="err-msg">{error}</div>{/if}
  </div>
</div>