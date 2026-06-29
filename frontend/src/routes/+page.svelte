<script>
  import { app, showToast } from '$lib/store.svelte.js';
  import { api } from '$lib/api.js';

  // ── State ──────────────────────────────────────────────
  let inFlight = $state(false);

  // ── Lock state config ───────────────────────────────────
  const STATE_CONFIG = {
    locked: {
      label: 'Locked',
      color: '#34d174',
      bg: 'rgba(52,209,116,0.10)',
      glow: 'rgba(52,209,116,0.22)',
    },
    unlocked: {
      label: 'Unlocked',
      color: '#f0a52a',
      bg: 'rgba(240,165,42,0.10)',
      glow: 'rgba(240,165,42,0.22)',
    },
    not_fully_locked: {
      label: 'Not Fully Locked',
      color: '#f04c52',
      bg: 'rgba(240,76,82,0.10)',
      glow: 'rgba(240,76,82,0.22)',
    },
    unknown: {
      label: 'Unknown',
      color: '#3e4a5c',
      bg: 'rgba(62,74,92,0.10)',
      glow: 'transparent',
    },
  };

  let stateInfo = $derived(
    STATE_CONFIG[/** @type {string} */ (app.lock?.lock_state)] ?? STATE_CONFIG.unknown
  );

  let recentEvents = $derived((app.events ?? []).slice(0, 5));

  // ── Helpers ─────────────────────────────────────────────
  /** @param {string | null | undefined} iso */
  function relTime(iso) {
    if (!iso) return '—';
    const diff = Date.now() - new Date(iso).getTime();
    const m = Math.floor(diff / 60000);
    const h = Math.floor(m / 60);
    const d = Math.floor(h / 24);
    if (m < 1)  return 'Just now';
    if (m < 60) return `${m}m ago`;
    if (h < 24) return `${h}h ago`;
    if (d === 1) return 'Yesterday';
    return `${d}d ago`;
  }

  /** @param {string} s */
  function srcLabel(s) {
    return (
      { keypad: 'Keypad', fingerprint: 'Fingerprint', rfid: 'RFID card',
        zigbee: 'App (zigbee)', self: 'Auto-relock', unknown: 'Unknown' }[s] ?? s
    );
  }

  // ── Actions ─────────────────────────────────────────────
  /** @param {'lock'|'unlock'} state */
  async function sendCommand(state) {
    if (inFlight) return;
    inFlight = true;
    try {
      await api.post('/api/lock/state', { state });
      showToast(state === 'lock' ? 'Lock command sent' : 'Unlock command sent');
    } catch (/** @type {any} */ e) {
      showToast(e.message, true);
    } finally {
      inFlight = false;
    }
  }
</script>

<div class="page">

  <!-- ── Broker / availability warnings ─────────────────── -->
  {#if app.lock?.broker_connected === false}
    <div class="banner banner-warn">
      <svg class="banner-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
        <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
      </svg>
      <div>
        <strong>Broker offline</strong> — commands won't reach the lock until the MQTT broker reconnects.
      </div>
    </div>
  {/if}

  {#if app.lock?.available === false}
    <div class="banner banner-warn">
      <svg class="banner-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="1" y1="1" x2="23" y2="23"/>
        <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55M5 12.55a10.94 10.94 0 0 1 5.17-2.39M10.71 5.05A16 16 0 0 1 22.56 9M1.42 9a15.91 15.91 0 0 1 4.7-2.88M8.53 16.11a6 6 0 0 1 6.95 0"/>
        <circle cx="12" cy="20" r="1" fill="currentColor" stroke="none"/>
      </svg>
      <div>
        <strong>Lock offline</strong> — not reachable on Zigbee. Check device power.
      </div>
    </div>
  {/if}

  <!-- ── Hero ────────────────────────────────────────────── -->
  <div
    class="hero-card"
    style="--sc: {stateInfo.color}; --sb: {stateInfo.bg}; --sg: {stateInfo.glow}"
  >
    <div class="lock-ring">
      {#if app.lock?.lock_state === 'unlocked'}
        <svg width="42" height="42" viewBox="0 0 24 24" fill="none" stroke={stateInfo.color} stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
        </svg>
      {:else}
        <svg width="42" height="42" viewBox="0 0 24 24" fill="none" stroke={stateInfo.color} stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="11" width="18" height="11" rx="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      {/if}
    </div>

    <div class="state-label" style="color: {stateInfo.color}">{stateInfo.label}</div>
    <div class="state-updated">Updated {relTime(/** @type {string} */(app.lock?.updated_at))}</div>
  </div>

  <!-- ── Control buttons ─────────────────────────────────── -->
  <div class="action-pair">
    <button
      class="btn btn-primary"
      onclick={() => sendCommand('unlock')}
      disabled={inFlight || !app.connected}
    >
      {#if inFlight}<span class="spinner-sm"></span>{/if}
      <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
      </svg>
      Unlock
    </button>
    <button
      class="btn btn-secondary"
      onclick={() => sendCommand('lock')}
      disabled={inFlight || !app.connected}
    >
      {#if inFlight}<span class="spinner-sm"></span>{/if}
      <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="11" width="18" height="11" rx="2"/>
        <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
      </svg>
      Lock
    </button>
  </div>

  <!-- ── Status ──────────────────────────────────────────── -->
  <p class="section-heading">Status</p>
  <div class="card">
    <div class="status-row">
      <span class="status-label">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="1" y="6" width="16" height="12" rx="2"/>
          <line x1="23" y1="10" x2="23" y2="14"/>
        </svg>
        Battery
      </span>
      <span
        class="status-value"
        style={app.lock?.battery != null && /** @type {number} */(app.lock.battery) < 20
          ? 'color: var(--error)' : ''}
      >
        {app.lock?.battery != null ? `${app.lock.battery}%` : '—'}
      </span>
    </div>

    <div class="status-row">
      <span class="status-label">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
        </svg>
        Signal (LQI)
      </span>
      <span class="status-value">
        {app.lock?.link_quality != null ? app.lock.link_quality : '—'}
      </span>
    </div>

    <div class="status-row">
      <span class="status-label">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
        </svg>
        Auto-relock
      </span>
      <span class="status-value">
        {#if app.lock?.auto_relock === true}
          On{app.lock?.auto_relock_time ? ` · ${app.lock.auto_relock_time}s` : ''}
        {:else if app.lock?.auto_relock === false}
          Off
        {:else}
          —
        {/if}
      </span>
    </div>

    <div class="status-row">
      <span class="status-label">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
          <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
        </svg>
        Sound
      </span>
      <span class="status-value">
        {#if app.lock?.sound_volume === 'silent_mode'}Silent
        {:else if app.lock?.sound_volume === 'low_volume'}Low
        {:else if app.lock?.sound_volume === 'high_volume'}High
        {:else}—{/if}
      </span>
    </div>

    <div class="status-row">
      <span class="status-label">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="2" x2="12" y2="6"/>
          <path d="M12 18a6 6 0 0 0 0-12"/>
          <path d="M12 22a10 10 0 0 0 0-20" opacity=".4"/>
        </svg>
        Voltage
      </span>
      <span class="status-value">
        {app.lock?.voltage != null ? `${Number(app.lock.voltage).toFixed(2)} V` : '—'}
      </span>
    </div>
  </div>

  <!-- ── Recent activity ─────────────────────────────────── -->
  {#if recentEvents.length > 0}
    <p class="section-heading">Recent Activity</p>
    <div class="card">
      {#each recentEvents as ev}
        {@const isLock = /** @type {any} */(ev).kind === 'lock'}
        <div class="ev-row">
          <span
            class="ev-pill"
            style="background: {isLock ? 'rgba(52,209,116,0.12)' : 'rgba(240,165,42,0.12)'}; color: {isLock ? '#34d174' : '#f0a52a'}"
            aria-hidden="true"
          >
            {isLock ? '↑' : '↓'}
          </span>
          <div class="ev-info">
            <span class="ev-action">{isLock ? 'Locked' : 'Unlocked'}</span>
            <span class="ev-meta">
              {srcLabel(/** @type {any} */(ev).source)}{/** @type {any} */(ev).user != null ? ` · Slot ${/** @type {any} */(ev).user}` : ''}
            </span>
          </div>
          <span class="ev-time">{relTime(/** @type {any} */(ev).at)}</span>
        </div>
      {/each}
    </div>
    <a href="/activity" class="see-all-link">See full activity log →</a>
  {/if}

</div>

<style>
  .hero-card {
    background: var(--bg-2);
    border: 1px solid var(--border);
    border-radius: 20px;
    padding: 32px 20px 24px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    text-align: center;
    margin-bottom: 12px;
    position: relative;
    overflow: hidden;
  }

  .hero-card::before {
    content: '';
    position: absolute;
    top: -40px;
    left: 50%;
    transform: translateX(-50%);
    width: 200px;
    height: 200px;
    background: radial-gradient(circle, var(--sg, transparent) 0%, transparent 70%);
    pointer-events: none;
    transition: background 0.5s;
  }

  .lock-ring {
    width: 96px;
    height: 96px;
    border-radius: 50%;
    background: var(--sb);
    border: 1.5px solid var(--sc);
    box-shadow: 0 0 32px var(--sg, transparent);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.4s, border-color 0.4s, box-shadow 0.4s;
    position: relative;
    z-index: 1;
  }

  .lock-ring svg {
    transition: stroke 0.4s;
  }

  .state-label {
    font-size: 28px;
    font-weight: 700;
    letter-spacing: -0.03em;
    transition: color 0.4s;
    position: relative;
    z-index: 1;
  }

  .state-updated {
    font-size: 12px;
    color: var(--t3);
    font-variant-numeric: tabular-nums;
    position: relative;
    z-index: 1;
  }

  .action-pair {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-bottom: 4px;
  }

  /* Activity rows */
  .ev-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 11px 16px;
    border-bottom: 1px solid var(--border);
  }

  .ev-row:last-child {
    border-bottom: none;
  }

  .ev-pill {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 700;
    flex-shrink: 0;
  }

  .ev-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .ev-action {
    font-size: 14px;
    font-weight: 500;
    color: var(--t1);
  }

  .ev-meta {
    font-size: 12px;
    color: var(--t3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ev-time {
    font-size: 12px;
    color: var(--t3);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  .see-all-link {
    display: block;
    text-align: center;
    margin-top: 12px;
    font-size: 13px;
    color: var(--accent);
    font-weight: 500;
    padding: 8px 0;
  }
</style>
