<script>
  import { app, api, action } from '$lib/store.svelte.js';

  let busy = $state(false);

  const labels = {
    locked: 'Locked',
    unlocked: 'Unlocked',
    not_fully_locked: 'Not fully locked',
    unknown: 'Unknown'
  };

  async function setState(state) {
    busy = true;
    await action(() => api.setLockState(state), state === 'lock' ? 'Locking…' : 'Unlocking…');
    busy = false;
  }

  let lock = $derived(app.lock);
  let recent = $derived(app.events.slice(0, 5));

  function fmtTime(t) {
    try {
      return new Date(t).toLocaleString();
    } catch {
      return t;
    }
  }
</script>

<section class="card">
  <div class="lock-hero">
    <div class="lock-state {lock.lock_state}">{labels[lock.lock_state] ?? lock.lock_state}</div>
    <div class="btn-group" style="width:100%">
      <button class="big primary" disabled={busy} onclick={() => setState('unlock')}>Unlock</button>
      <button class="big" disabled={busy} onclick={() => setState('lock')}>Lock</button>
    </div>
    {#if !lock.available}
      <p class="muted center">Lock is currently unreachable over MQTT.</p>
    {/if}
  </div>
</section>

<section class="card">
  <h2>Status</h2>
  <div class="row"><span class="label">Battery</span><span class="value">{lock.battery != null ? lock.battery + '%' : '—'}</span></div>
  <div class="row"><span class="label">Voltage</span><span class="value">{lock.voltage != null ? lock.voltage + ' V' : '—'}</span></div>
  <div class="row"><span class="label">Signal</span><span class="value">{lock.link_quality != null ? lock.link_quality + ' lqi' : '—'}</span></div>
  <div class="row"><span class="label">Sound</span><span class="value">{lock.sound_volume || '—'}</span></div>
  <div class="row"><span class="label">Auto relock</span><span class="value">{lock.auto_relock == null ? '—' : lock.auto_relock ? 'On' : 'Off'}</span></div>
</section>

<section class="card">
  <h2>Recent activity</h2>
  {#if recent.length === 0}
    <p class="muted">No activity recorded yet.</p>
  {:else}
    <div class="list">
      {#each recent as ev}
        <div class="list-item">
          <div>
            <div class="value">{ev.kind === 'unlock' ? 'Unlocked' : 'Locked'} · {ev.source}</div>
            <div class="muted" style="font-size:0.8rem">{fmtTime(ev.at)}</div>
          </div>
          {#if ev.user != null}<span class="badge">user {ev.user}</span>{/if}
        </div>
      {/each}
    </div>
  {/if}
</section>
