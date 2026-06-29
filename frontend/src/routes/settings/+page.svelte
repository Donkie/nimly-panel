<script>
  import { app, showToast } from '$lib/store.svelte.js';
  import { api } from '$lib/api.js';

  const VOLUMES = [
    { value: 'silent_mode', label: 'Silent',  icon: '🔇' },
    { value: 'low_volume',  label: 'Low',     icon: '🔉' },
    { value: 'high_volume', label: 'High',    icon: '🔊' },
  ];

  let autoRelockPending = $state(false);
  let soundPending      = $state(false);
  let refreshPending    = $state(false);
  let signOutPending    = $state(false);

  /** @param {string} vol */
  async function setVolume(vol) {
    if (soundPending || app.lock?.sound_volume === vol) return;
    soundPending = true;
    try {
      await api.post('/api/settings/sound-volume', { sound_volume: vol });
      showToast('Sound setting sent');
    } catch (/** @type {any} */ e) {
      showToast(e.message, true);
    } finally {
      soundPending = false;
    }
  }

  /** @param {boolean} enabled */
  async function setAutoRelock(enabled) {
    if (autoRelockPending) return;
    autoRelockPending = true;
    try {
      await api.post('/api/settings/auto-relock', { enabled });
      showToast(enabled ? 'Auto-relock enabled' : 'Auto-relock disabled');
    } catch (/** @type {any} */ e) {
      showToast(e.message, true);
    } finally {
      autoRelockPending = false;
    }
  }

  async function refreshLock() {
    if (refreshPending) return;
    refreshPending = true;
    try {
      await api.post('/api/refresh', undefined);
      showToast('Refresh requested');
    } catch (/** @type {any} */ e) {
      showToast(e.message, true);
    } finally {
      refreshPending = false;
    }
  }

  async function signOut() {
    if (signOutPending) return;
    signOutPending = true;
    try {
      await api.post('/api/auth/logout', undefined);
    } catch (_) { /* ignore */ }
    window.location.href = '/api/auth/login';
  }
</script>

<div class="page">

  <!-- ── Sound volume ──────────────────────────────────── -->
  <p class="section-heading">Sound Volume</p>
  <div class="vol-grid">
    {#each VOLUMES as vol}
      <button
        class="vol-btn"
        class:active={app.lock?.sound_volume === vol.value}
        onclick={() => setVolume(vol.value)}
        disabled={soundPending}
        aria-pressed={app.lock?.sound_volume === vol.value}
      >
        <span class="vol-icon" aria-hidden="true">{vol.icon}</span>
        <span class="vol-label">{vol.label}</span>
      </button>
    {/each}
  </div>

  <!-- ── Auto-relock ───────────────────────────────────── -->
  <p class="section-heading">Auto-relock</p>
  <div class="card">
    <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
    <div
      class="setting-row"
      onclick={() => setAutoRelock(!(app.lock?.auto_relock === true))}
      role="button"
      tabindex="0"
      onkeydown={(e) => e.key === 'Enter' && setAutoRelock(!(app.lock?.auto_relock === true))}
    >
      <div class="setting-info">
        <span class="setting-name">Auto-relock</span>
        {#if app.lock?.auto_relock === true && app.lock?.auto_relock_time}
          <span class="setting-sub">Relocks after {app.lock.auto_relock_time}s</span>
        {:else}
          <span class="setting-sub">Automatically lock after unlocking</span>
        {/if}
      </div>
      <label class="toggle" onclick={(e) => e.stopPropagation()}>
        <input
          type="checkbox"
          checked={app.lock?.auto_relock === true}
          onchange={(e) => setAutoRelock(/** @type {HTMLInputElement} */(e.target).checked)}
          disabled={autoRelockPending}
        />
        <div class="toggle-track" class:on={app.lock?.auto_relock === true}></div>
        <div class="toggle-thumb" class:on={app.lock?.auto_relock === true}></div>
      </label>
    </div>
  </div>

  <!-- ── Device ────────────────────────────────────────── -->
  <p class="section-heading">Device</p>
  <div class="card">
    <button
      class="setting-row setting-btn"
      onclick={refreshLock}
      disabled={refreshPending}
    >
      <div class="setting-info">
        <span class="setting-name">Refresh from lock</span>
        <span class="setting-sub">Re-fetch current state via MQTT</span>
      </div>
      {#if refreshPending}
        <span class="spinner-sm"></span>
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color: var(--t3)">
          <polyline points="23 4 23 10 17 10"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      {/if}
    </button>
  </div>

  <!-- ── Account ───────────────────────────────────────── -->
  <p class="section-heading">Account</p>
  <div class="card">
    <div class="account-info">
      <div class="account-avatar" aria-hidden="true">
        {(app.me?.name ?? app.me?.email ?? '?').charAt(0).toUpperCase()}
      </div>
      <div class="account-text">
        <span class="account-name">{app.me?.name ?? '—'}</span>
        <span class="account-email">{app.me?.email ?? '—'}</span>
        {#if app.me?.groups?.length}
          <span class="account-groups">{app.me.groups.join(', ')}</span>
        {/if}
      </div>
    </div>

    <div class="card-divider"></div>

    <div class="setting-row setting-btn-row">
      <button
        class="btn btn-ghost btn-sm sign-out-btn"
        onclick={signOut}
        disabled={signOutPending}
      >
        {#if signOutPending}<span class="spinner-sm"></span>{/if}
        Sign out
      </button>
    </div>
  </div>

</div>

<style>
  /* ── Volume grid ── */
  .vol-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    margin-bottom: 4px;
  }

  .vol-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 14px 8px;
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: var(--r-lg);
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
    min-height: 76px;
    -webkit-tap-highlight-color: transparent;
  }

  .vol-btn.active {
    background: var(--accent-bg);
    border-color: var(--accent-border);
  }

  .vol-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .vol-icon {
    font-size: 20px;
    line-height: 1;
  }

  .vol-label {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.03em;
    text-transform: uppercase;
    color: var(--t2);
  }

  .vol-btn.active .vol-label {
    color: var(--accent);
  }

  /* ── Settings rows ── */
  .setting-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 14px 16px;
  }

  .setting-btn {
    width: 100%;
    background: none;
    border: none;
    cursor: pointer;
    text-align: left;
    font: inherit;
    color: inherit;
    -webkit-tap-highlight-color: transparent;
  }

  .setting-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .setting-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .setting-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--t1);
  }

  .setting-sub {
    font-size: 12px;
    color: var(--t3);
  }

  /* ── Account ── */
  .account-info {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
  }

  .account-avatar {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--accent-bg);
    border: 1px solid var(--accent-border);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16px;
    font-weight: 700;
    color: var(--accent);
    flex-shrink: 0;
  }

  .account-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .account-name {
    font-size: 15px;
    font-weight: 500;
    color: var(--t1);
  }

  .account-email {
    font-size: 13px;
    color: var(--t2);
  }

  .account-groups {
    font-size: 11px;
    color: var(--t3);
    letter-spacing: 0.02em;
  }

  .card-divider {
    height: 1px;
    background: var(--border);
    margin: 0 16px;
  }

  .setting-btn-row {
    justify-content: flex-start;
  }

  .sign-out-btn {
    color: var(--error);
    border-color: rgba(240, 76, 82, 0.3);
  }
</style>
