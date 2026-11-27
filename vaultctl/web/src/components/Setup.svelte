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
  <div class="card auth-card">
    <div class="mono-mark" aria-hidden="true">V</div>
    <div class="eyebrow"><span class="eyebrow-mark">✳</span> YOUR PRIVATE KEYRING</div>
    <h2>Create your vault</h2>
    <p class="sub">Choose a strong master password. It is never stored — lose it and the data is unrecoverable.</p>
    <div class="field">
      <label for="setupPassword">Master password</label>
      <div class="input-wrap">
        <input id="setupPassword" class="input" type={showPw ? 'text' : 'password'} bind:value={password} autocomplete="new-password" placeholder="at least 8 characters" minlength="8" />
        <button type="button" class="pw" onclick={() => (showPw = !showPw)}>{showPw ? 'hide' : 'show'}</button>
      </div>
    </div>
    <div class="field">
      <label for="setupConfirm">Confirm password</label>
      <div class="input-wrap">
        <input id="setupConfirm" class="input" type={showPw2 ? 'text' : 'password'} bind:value={confirm} autocomplete="new-password" placeholder="repeat password" minlength="8" />
        <button type="button" class="pw" onclick={() => (showPw2 = !showPw2)}>{showPw2 ? 'hide' : 'show'}</button>
      </div>
    </div>
    <button class="btn cta" disabled={loading} onclick={submit}>Create vault</button>
    {#if error}<div class="err-msg">{error}</div>{/if}
  </div>
</div>
