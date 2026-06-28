<script>
  import { app, api, action } from '$lib/store.svelte.js';

  const userTypes = [
    { value: 'unrestricted', label: 'Unrestricted' },
    { value: 'master', label: 'Master (can program)' },
    { value: 'week_day_schedule', label: 'Weekly schedule' },
    { value: 'year_day_schedule', label: 'Date-range schedule' },
    { value: 'non_access', label: 'No access' }
  ];

  let sheetOpen = $state(false);
  let editing = $state(null); // existing pin or null
  let busy = $state(false);

  let form = $state({ user: 0, name: '', user_type: 'unrestricted', user_enabled: true, pin_code: '' });

  let lock = $derived(app.lock);
  let minLen = $derived(lock.min_pin_length ?? 4);
  let maxLen = $derived(lock.max_pin_length ?? 8);
  let maxUsers = $derived(lock.max_pin_users ?? null);
  let pins = $derived([...app.pins].sort((a, b) => a.user - b.user));

  function nextFreeUser() {
    const used = new Set(app.pins.map((p) => p.user));
    let i = 0;
    while (used.has(i)) i++;
    return i;
  }

  function openAdd() {
    editing = null;
    form = { user: nextFreeUser(), name: '', user_type: 'unrestricted', user_enabled: true, pin_code: '' };
    sheetOpen = true;
  }

  function openEdit(pin) {
    editing = pin;
    form = {
      user: pin.user,
      name: pin.name || '',
      user_type: pin.user_type || 'unrestricted',
      user_enabled: pin.user_enabled,
      pin_code: ''
    };
    sheetOpen = true;
  }

  function close() {
    sheetOpen = false;
  }

  let pinError = $derived.by(() => {
    if (form.pin_code === '') return null; // empty handled at submit
    if (!/^\d+$/.test(form.pin_code)) return 'Digits only';
    if (form.pin_code.length < minLen || form.pin_code.length > maxLen)
      return `Must be ${minLen}–${maxLen} digits`;
    return null;
  });

  // When editing, the code is optional: leaving it blank just renames the slot
  // (the lock isn't touched). When adding, a code is required.
  let renameOnly = $derived(!!editing && form.pin_code.length === 0);
  let canSave = $derived(
    (maxUsers == null || form.user < maxUsers) &&
      (renameOnly ? form.name.trim().length > 0 : form.pin_code.length > 0 && !pinError)
  );

  async function save() {
    busy = true;
    const ok = await action(
      () =>
        api.setPin(form.user, {
          name: form.name.trim(),
          user_type: form.user_type,
          user_enabled: form.user_enabled,
          pin_code: form.pin_code
        }),
      renameOnly ? 'Name updated' : editing ? 'PIN updated' : 'PIN added'
    );
    busy = false;
    if (ok) close();
  }

  async function remove(pin) {
    if (!confirm(`Delete PIN for user ${pin.user}? This cannot be undone.`)) return;
    await action(() => api.deletePin(pin.user), 'PIN deleted');
  }

  function typeLabel(v) {
    return userTypes.find((t) => t.value === v)?.label ?? v ?? '—';
  }
</script>

<section class="card">
  <h2>PIN codes</h2>
  <p class="muted" style="margin-top:-4px">
    {pins.length}{maxUsers != null ? ` / ${maxUsers}` : ''} slots used · {minLen}–{maxLen} digits
  </p>
  {#if pins.length === 0}
    <p class="muted">No PIN codes programmed. Tap “Add PIN” to create one.</p>
  {:else}
    <div class="list">
      {#each pins as pin}
        <div class="list-item">
          <div>
            <div class="value">{pin.name || `User ${pin.user}`}</div>
            <div class="muted" style="font-size:0.8rem">slot {pin.user} · {typeLabel(pin.user_type)}</div>
          </div>
          <div style="display:flex;align-items:center;gap:8px">
            <span class="badge {pin.user_enabled ? 'on' : 'off'}">{pin.user_enabled ? 'enabled' : 'disabled'}</span>
            <button onclick={() => openEdit(pin)}>Edit</button>
            <button class="danger" onclick={() => remove(pin)}>✕</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</section>

<button class="primary block big" onclick={openAdd}>＋ Add PIN</button>

{#if sheetOpen}
  <div class="sheet-backdrop" onclick={close} onkeydown={(e) => e.key === 'Escape' && close()} role="presentation">
    <div class="sheet" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex="-1">
      <h3>{editing ? `Edit PIN · user ${form.user}` : 'Add PIN'}</h3>

      {#if !editing}
        <label class="field">
          User slot
          <input type="number" min="0" max={maxUsers != null ? maxUsers - 1 : undefined} bind:value={form.user} />
        </label>
      {/if}

      <label class="field">
        Name
        <input type="text" autocomplete="off" placeholder="e.g. Cleaner, Guest, Sarah" bind:value={form.name} />
      </label>

      <label class="field">
        PIN code ({minLen}–{maxLen} digits){#if editing}<span class="muted"> — leave blank to keep current</span>{/if}
        <input
          type="tel"
          inputmode="numeric"
          autocomplete="off"
          placeholder={editing ? 'Leave blank to rename only' : 'e.g. 1234'}
          bind:value={form.pin_code}
        />
        {#if pinError}<span style="color:var(--danger)">{pinError}</span>{/if}
      </label>

      <label class="field">
        Access type
        <select bind:value={form.user_type}>
          {#each userTypes as t}<option value={t.value}>{t.label}</option>{/each}
        </select>
      </label>

      <label class="row" style="border:none;padding:0">
        <span class="label">Enabled</span>
        <input type="checkbox" style="width:auto;min-height:auto" bind:checked={form.user_enabled} />
      </label>

      <div class="btn-group">
        <button onclick={close}>Cancel</button>
        <button class="primary" disabled={!canSave || busy} onclick={save}>
          {renameOnly ? 'Save name' : editing ? 'Save' : 'Add PIN'}
        </button>
      </div>
      {#if editing && !renameOnly}
        <p class="muted center" style="font-size:0.8rem">Entering a code overwrites the code for this slot on the lock.</p>
      {:else if renameOnly}
        <p class="muted center" style="font-size:0.8rem">Updates the name only — the lock isn't changed.</p>
      {/if}
    </div>
  </div>
{/if}
