<script>
  import { app, showToast } from '$lib/store.svelte.js';
  import { api } from '$lib/api.js';

  // ── Derived lock limits ─────────────────────────────────
  let maxSlots  = $derived(/** @type {number} */(app.lock?.max_pin_users)  ?? 10);
  let minLen    = $derived(/** @type {number} */(app.lock?.min_pin_length) ?? 4);
  let maxLen    = $derived(/** @type {number} */(app.lock?.max_pin_length) ?? 8);

  let sortedPins = $derived(
    [...(/** @type {any[]} */(app.pins) ?? [])].sort((a, b) => a.user - b.user)
  );

  let usedSlots = $derived(new Set((/** @type {any[]} */(app.pins) ?? []).map(p => p.user)));

  let nextFreeSlot = $derived(
    Array.from({ length: maxSlots }, (_, i) => i).find(i => !usedSlots.has(i)) ?? 0
  );

  // ── Sheet state ─────────────────────────────────────────
  let showSheet   = $state(false);
  /** @type {any | null} */
  let editingPin  = $state(null);

  let formSlot    = $state(0);
  let formName    = $state('');
  let formPin     = $state('');
  let formType    = $state('unrestricted');
  let formEnabled = $state(true);

  // ── Validation ──────────────────────────────────────────
  let pinOnlyDigits  = $derived(!formPin || /^\d+$/.test(formPin));
  let pinRightLength = $derived(!formPin || (formPin.length >= minLen && formPin.length <= maxLen));
  let pinError = $derived(
    !formPin          ? '' :
    !pinOnlyDigits    ? 'Digits only' :
    !pinRightLength   ? `Must be ${minLen}–${maxLen} digits` : ''
  );
  let pinValid = $derived(!pinError);

  let isRenameOnly = $derived(editingPin !== null && formPin === '');
  let canSave      = $derived(pinValid && (editingPin ? true : formPin.length >= minLen));

  // ── Open sheet helpers ───────────────────────────────────
  function openAdd() {
    editingPin  = null;
    formSlot    = nextFreeSlot;
    formName    = '';
    formPin     = '';
    formType    = 'unrestricted';
    formEnabled = true;
    showSheet   = true;
  }

  /** @param {any} pin */
  function openEdit(pin) {
    editingPin  = pin;
    formSlot    = pin.user;
    formName    = pin.name ?? '';
    formPin     = '';
    formType    = pin.user_type ?? 'unrestricted';
    formEnabled = pin.user_enabled ?? true;
    showSheet   = true;
  }

  function closeSheet() {
    showSheet  = false;
    editingPin = null;
  }

  // ── Save ─────────────────────────────────────────────────
  let saving = $state(false);

  async function savePin() {
    if (!canSave || saving) return;
    saving = true;
    const body = {
      name:         formName,
      user_type:    formType,
      user_enabled: formEnabled,
      ...(formPin ? { pin_code: formPin } : {})
    };
    try {
      await api.put(`/api/pins/${formSlot}`, body);
      showToast(editingPin ? (isRenameOnly ? 'Name updated' : 'PIN updated') : 'PIN slot added');
      closeSheet();
    } catch (/** @type {any} */ e) {
      showToast(e.message, true);
    } finally {
      saving = false;
    }
  }

  // ── Delete ───────────────────────────────────────────────
  /** @param {any} pin */
  async function deletePin(pin) {
    const label = pin.name ? `"${pin.name}"` : `Slot ${pin.user}`;
    if (!confirm(`Delete PIN ${label}? This cannot be undone.`)) return;
    try {
      await api.delete(`/api/pins/${pin.user}`);
      showToast(`PIN ${label} deleted`);
    } catch (/** @type {any} */ e) {
      showToast(e.message, true);
    }
  }

  // ── Type label ───────────────────────────────────────────
  /** @param {string} t */
  function typeLabel(t) {
    return (
      { unrestricted: 'Unrestricted', master: 'Master', week_day_schedule: 'Weekday schedule',
        year_day_schedule: 'Date-range schedule', non_access: 'No access' }[t] ?? t
    );
  }
</script>

<div class="page">
  <!-- ── Slot summary ──────────────────────────────────── -->
  <div class="summary-bar">
    <span class="summary-count">
      <strong>{sortedPins.length}</strong> / {maxSlots} slots used
    </span>
    <span class="summary-sep">·</span>
    <span class="summary-digits">{minLen}–{maxLen} digit PINs</span>
  </div>

  <!-- ── PIN list ──────────────────────────────────────── -->
  {#if sortedPins.length === 0}
    <div class="empty-state">
      <div class="empty-state-title">No PIN slots configured</div>
      <div class="empty-state-sub">Add a PIN to get started.</div>
    </div>
  {:else}
    <div class="card pin-list">
      {#each sortedPins as pin (pin.user)}
        <div class="pin-row">
          <span class="slot-badge">#{pin.user}</span>
          <div class="pin-info">
            <span class="pin-name">{pin.name || `User ${pin.user}`}</span>
            <span class="pin-meta">
              {typeLabel(pin.user_type)}
              {#if !pin.user_enabled}
                <span class="badge-disabled">Disabled</span>
              {/if}
              {#if !pin.has_code}
                <span class="badge-nocode">No code</span>
              {/if}
            </span>
          </div>
          <div class="pin-actions">
            <button
              class="icon-btn"
              onclick={() => openEdit(pin)}
              aria-label="Edit PIN slot {pin.user}"
              title="Edit"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
            </button>
            <button
              class="icon-btn icon-btn-danger"
              onclick={() => deletePin(pin)}
              aria-label="Delete PIN slot {pin.user}"
              title="Delete"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/>
                <path d="M10 11v6"/><path d="M14 11v6"/>
                <path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>
              </svg>
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- ── Add PIN button ────────────────────────────────── -->
  <button
    class="btn btn-primary add-btn"
    onclick={openAdd}
    disabled={sortedPins.length >= maxSlots}
  >
    <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
      <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
    </svg>
    Add PIN slot
  </button>
</div>

<!-- ── Bottom sheet ────────────────────────────────────── -->
{#if showSheet}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="sheet-backdrop" onclick={(e) => { if (e.target === e.currentTarget) closeSheet(); }}>
    <div class="sheet" role="dialog" aria-modal="true" aria-label={editingPin ? 'Edit PIN slot' : 'Add PIN slot'}>
      <div class="sheet-handle"></div>
      <h2 class="sheet-title">{editingPin ? 'Edit PIN Slot' : 'Add PIN Slot'}</h2>

      <!-- Slot number -->
      <div class="field">
        <label class="field-label" for="field-slot">Slot number (0–{maxSlots - 1})</label>
        <input
          id="field-slot"
          class="field-input"
          type="number"
          min="0"
          max={maxSlots - 1}
          bind:value={formSlot}
          disabled={!!editingPin}
        />
      </div>

      <!-- Name -->
      <div class="field">
        <label class="field-label" for="field-name">Name <span style="color: var(--t3)">(optional)</span></label>
        <input
          id="field-name"
          class="field-input"
          type="text"
          maxlength="32"
          placeholder="e.g. Front door"
          bind:value={formName}
        />
      </div>

      <!-- PIN code -->
      <div class="field">
        <label class="field-label" for="field-pin">
          PIN code
          {#if editingPin}
            <span style="color: var(--t3)">(leave blank to keep current)</span>
          {/if}
        </label>
        <input
          id="field-pin"
          class="field-input"
          class:invalid={!!pinError}
          type="tel"
          inputmode="numeric"
          pattern="[0-9]*"
          autocomplete="off"
          placeholder="{minLen}–{maxLen} digits"
          bind:value={formPin}
        />
        {#if pinError}
          <span class="field-error">{pinError}</span>
        {:else if editingPin && isRenameOnly}
          <span class="field-hint">No PIN entered — only the name and settings will be updated.</span>
        {:else if !editingPin && !formPin}
          <span class="field-hint">PIN digits are write-only and never displayed after saving.</span>
        {/if}
      </div>

      <!-- Access type -->
      <div class="field">
        <label class="field-label" for="field-type">Access type</label>
        <select id="field-type" class="field-input" bind:value={formType}>
          <option value="unrestricted">Unrestricted</option>
          <option value="master">Master</option>
          <option value="week_day_schedule">Weekday schedule</option>
          <option value="year_day_schedule">Date-range schedule</option>
          <option value="non_access">No access</option>
        </select>
      </div>

      <!-- Enabled toggle -->
      <div class="field enabled-row">
        <span class="field-label" id="enabled-label">Enabled</span>
        <label class="toggle" aria-labelledby="enabled-label">
          <input type="checkbox" bind:checked={formEnabled} />
          <div class="toggle-track" class:on={formEnabled}></div>
          <div class="toggle-thumb" class:on={formEnabled}></div>
        </label>
      </div>

      <!-- Actions -->
      <div class="sheet-actions">
        <button
          class="btn btn-primary"
          onclick={savePin}
          disabled={!canSave || saving}
        >
          {#if saving}<span class="spinner-sm"></span>{/if}
          {editingPin ? (isRenameOnly ? 'Save name' : 'Save changes') : 'Add PIN slot'}
        </button>
        <button class="btn btn-ghost" onclick={closeSheet}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .summary-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 2px;
    margin-bottom: 12px;
    font-size: 13px;
    color: var(--t3);
  }

  .summary-count strong {
    color: var(--t1);
    font-weight: 600;
  }

  .summary-sep {
    color: var(--t3);
  }

  .pin-list {
    margin-bottom: 12px;
  }

  .pin-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
  }

  .pin-row:last-child {
    border-bottom: none;
  }

  .slot-badge {
    width: 32px;
    height: 32px;
    border-radius: 8px;
    background: var(--bg-3);
    border: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 11px;
    font-weight: 700;
    color: var(--t3);
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
    letter-spacing: 0.02em;
  }

  .pin-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .pin-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--t1);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .pin-meta {
    font-size: 12px;
    color: var(--t3);
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }

  .badge-disabled,
  .badge-nocode {
    display: inline-block;
    font-size: 10px;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    letter-spacing: 0.03em;
    text-transform: uppercase;
  }

  .badge-disabled {
    background: var(--warn-bg);
    color: var(--warn);
  }

  .badge-nocode {
    background: var(--bg-4);
    color: var(--t3);
  }

  .pin-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .icon-btn {
    width: 34px;
    height: 34px;
    border-radius: var(--r-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--t3);
    background: transparent;
    border: 1px solid transparent;
    transition: color 0.15s, background 0.15s, border-color 0.15s;
    cursor: pointer;
  }

  .icon-btn:hover {
    color: var(--t1);
    background: var(--bg-3);
    border-color: var(--border);
  }

  .icon-btn-danger:hover {
    color: var(--error);
    background: var(--error-bg);
    border-color: rgba(240, 76, 82, 0.3);
  }

  .add-btn {
    margin-top: 4px;
  }

  .enabled-row {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 20px;
  }

  .sheet-actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 4px;
  }
</style>
