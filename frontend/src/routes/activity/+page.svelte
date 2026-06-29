<script>
  import { app } from '$lib/store.svelte.js';

  const SOURCE_LABELS = /** @type {Record<string, string>} */ ({
    keypad:      'Keypad',
    fingerprint: 'Fingerprint',
    rfid:        'RFID card',
    zigbee:      'App (zigbee)',
    self:        'Auto-relock',
    unknown:     'Unknown',
  });

  /** @param {string | null | undefined} iso */
  function formatDate(iso) {
    if (!iso) return '—';
    const d = new Date(iso);
    const now = new Date();
    const diffMs = now.getTime() - d.getTime();
    const diffMin = Math.floor(diffMs / 60000);
    const diffHr  = Math.floor(diffMin / 60);

    if (diffMin < 1)   return 'Just now';
    if (diffMin < 60)  return `${diffMin}m ago`;
    if (diffHr  < 24)  return `${diffHr}h ago`;

    return new Intl.DateTimeFormat(undefined, {
      month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit'
    }).format(d);
  }
</script>

<div class="page">
  <p class="section-heading">Activity Log</p>

  {#if !app.events?.length}
    <div class="empty-state">
      <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" style="color: var(--t3)">
        <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
      </svg>
      <div class="empty-state-title">No activity recorded yet.</div>
      <div class="empty-state-sub">Events will appear here as the lock is used.</div>
    </div>
  {:else}
    <div class="card">
      {#each app.events as ev}
        {@const isLock   = /** @type {any} */(ev).kind === 'lock'}
        {@const evColor  = isLock ? '#34d174' : '#f0a52a'}
        {@const evBg     = isLock ? 'rgba(52,209,116,0.10)' : 'rgba(240,165,42,0.10)'}
        {@const src      = SOURCE_LABELS[/** @type {any} */(ev).source] ?? /** @type {any} */(ev).source}
        {@const slot     = /** @type {any} */(ev).user}
        <div class="ev-row">
          <div class="ev-icon" style="background: {evBg}; color: {evColor}" aria-hidden="true">
            {#if isLock}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
            {:else}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <rect x="3" y="11" width="18" height="11" rx="2"/>
                <path d="M7 11V7a5 5 0 0 1 9.9-1"/>
              </svg>
            {/if}
          </div>
          <div class="ev-body">
            <span class="ev-title">
              {isLock ? 'Locked' : 'Unlocked'}
              <span class="ev-source">via {src}{slot != null ? ` · Slot ${slot}` : ''}</span>
            </span>
            <span class="ev-time">{formatDate(/** @type {any} */(ev).at)}</span>
          </div>
        </div>
      {/each}
    </div>
    <p class="disclaimer">Events are kept in memory and reset when the server restarts.</p>
  {/if}
</div>

<style>
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

  .ev-icon {
    width: 30px;
    height: 30px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .ev-body {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .ev-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--t1);
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 5px;
  }

  .ev-source {
    font-size: 12px;
    font-weight: 400;
    color: var(--t3);
  }

  .ev-time {
    font-size: 12px;
    color: var(--t3);
    font-variant-numeric: tabular-nums;
  }

  .disclaimer {
    margin-top: 14px;
    font-size: 12px;
    color: var(--t3);
    text-align: center;
    padding: 0 4px;
    line-height: 1.5;
  }
</style>
