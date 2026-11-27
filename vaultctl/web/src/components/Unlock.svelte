<script>
  import { api } from '../lib/api.js';
  import { unlocked, csrf } from '../lib/stores.js';

  let password = $state('');
  let error = $state('');
  let loading = $state(false);
  let showPw = $state(false);

  async function submit() {
    error = '';
    loading = true;
    const r = await api('/api/unlock', 'POST', { password });
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
  <div class="card auth-card">
    <div class="mono-mark" aria-hidden="true">V</div>
    <div class="eyebrow"><span class="eyebrow-mark">✳</span> SECURE ACCESS / MASTER KEY</div>
    <h2>Unlock vault</h2>
    <p class="sub">Enter your master password to access your credentials.</p>
    <div class="field">
      <label for="unlockPassword">Master password</label>
      <div class="input-wrap">
        <input id="unlockPassword" class="input" type={showPw ? 'text' : 'password'} bind:value={password} autocomplete="current-password" placeholder="master password" onkeydown={(e) => { if (e.key === 'Enter') submit(); }} />
        <button type="button" class="pw" onclick={() => (showPw = !showPw)}>{showPw ? 'hide' : 'show'}</button>
      </div>
    </div>
    <button class="btn cta" disabled={loading} onclick={submit}>Unlock</button>
    {#if error}<div class="err-msg">{error}</div>{/if}
  </div>
</div>
