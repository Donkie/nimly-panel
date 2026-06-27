<script>
  import { app, api, action } from '$lib/store.svelte.js';

  const volumes = [
    { value: 'silent_mode', label: 'Silent' },
    { value: 'low_volume', label: 'Low' },
    { value: 'high_volume', label: 'High' }
  ];

  let lock = $derived(app.lock);
  let busy = $state(false);

  async function setVolume(v) {
    busy = true;
    await action(() => api.setSoundVolume(v), 'Sound volume updated');
    busy = false;
  }

  async function toggleRelock(enabled) {
    busy = true;
    await action(() => api.setAutoRelock(enabled), 'Auto relock updated');
    busy = false;
  }

  async function refresh() {
    busy = true;
    await action(() => api.refresh(), 'Refreshing from lock…');
    busy = false;
  }

  async function logout() {
    await action(() => api.logout());
    window.location.href = '/api/auth/login';
  }
</script>

<section class="card">
  <h2>Sound volume</h2>
  <div class="btn-group">
    {#each volumes as v}
      <button class:primary={lock.sound_volume === v.value} disabled={busy} onclick={() => setVolume(v.value)}>
        {v.label}
      </button>
    {/each}
  </div>
</section>

<section class="card">
  <h2>Auto relock</h2>
  <div class="row" style="border:none;padding:0">
    <div>
      <div class="value">{lock.auto_relock ? 'Enabled' : 'Disabled'}</div>
      {#if lock.auto_relock_time != null}
        <div class="muted" style="font-size:0.8rem">Relocks after {lock.auto_relock_time}s</div>
      {/if}
    </div>
    <div class="btn-group">
      <button class:primary={lock.auto_relock === true} disabled={busy} onclick={() => toggleRelock(true)}>On</button>
      <button class:primary={lock.auto_relock === false} disabled={busy} onclick={() => toggleRelock(false)}>Off</button>
    </div>
  </div>
</section>

<section class="card">
  <h2>Account</h2>
  <div class="row" style="border:none;padding:0 0 12px">
    <span class="label">Signed in as</span>
    <span class="value">{app.me?.name || app.me?.email || app.me?.subject || '—'}</span>
  </div>
  <div class="stack">
    <button class="block" disabled={busy} onclick={refresh}>Refresh from lock</button>
    <button class="block danger" onclick={logout}>Sign out</button>
  </div>
</section>
