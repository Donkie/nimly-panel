<script>
  import { app } from '$lib/store.svelte.js';

  let events = $derived(app.events);

  function fmtTime(t) {
    try {
      return new Date(t).toLocaleString();
    } catch {
      return t;
    }
  }
</script>

<section class="card">
  <h2>Activity log</h2>
  {#if events.length === 0}
    <p class="muted">No activity recorded since the panel started.</p>
  {:else}
    <div class="list">
      {#each events as ev}
        <div class="list-item">
          <div>
            <div class="value">
              {ev.kind === 'unlock' ? '🔓 Unlocked' : '🔒 Locked'} · {ev.source}
            </div>
            <div class="muted" style="font-size:0.8rem">{fmtTime(ev.at)}</div>
          </div>
          {#if ev.user != null}<span class="badge">user {ev.user}</span>{/if}
        </div>
      {/each}
    </div>
  {/if}
</section>

<p class="muted center" style="font-size:0.8rem">
  Activity is derived live from the lock. PIN digits are never shown.
</p>
