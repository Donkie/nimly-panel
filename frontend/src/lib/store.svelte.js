// Central reactive application state (Svelte 5 runes) plus bootstrapping and a
// live SSE subscription to the backend's /api/stream endpoint.

import { api, setCsrfToken, loginRedirect } from './api.js';

export const app = $state({
  ready: false,
  me: null,
  connected: false,
  /** @type {any} */
  lock: { lock_state: 'unknown', available: false },
  /** @type {any[]} */
  pins: [],
  /** @type {any[]} */
  events: [],
  toast: null
});

let toastTimer;
export function showToast(message, isError = false) {
  app.toast = { message, isError };
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (app.toast = null), 3200);
}

function applySnapshot(snap) {
  if (!snap) return;
  if (snap.state) app.lock = snap.state;
  if (Array.isArray(snap.pins)) app.pins = snap.pins;
  if (Array.isArray(snap.events)) app.events = snap.events;
}

let started = false;

/** Bootstrap: load the user, then open the live stream. */
export async function init() {
  if (started) return;
  started = true;
  try {
    const me = await api.me();
    app.me = me;
    setCsrfToken(me.csrf_token);
  } catch (e) {
    // api.me triggers a login redirect on 401; nothing else to do.
    return;
  }
  try {
    applySnapshot(await api.getLock());
  } catch (e) {
    showToast(/** @type {Error} */ (e).message, true);
  }
  app.ready = true;
  openStream();
}

function openStream() {
  const es = new EventSource('/api/stream', { withCredentials: true });
  es.addEventListener('open', () => (app.connected = true));
  es.addEventListener('state', (ev) => {
    try {
      applySnapshot(JSON.parse(/** @type {MessageEvent} */ (ev).data));
    } catch {
      /* ignore malformed frame */
    }
  });
  es.addEventListener('error', () => {
    app.connected = false;
    // EventSource auto-reconnects. If the session expired, a manual API call
    // will surface the 401 and redirect.
  });
}

/** Wrap an API action with toast feedback. */
export async function action(fn, successMsg) {
  try {
    await fn();
    if (successMsg) showToast(successMsg);
    return true;
  } catch (e) {
    showToast(/** @type {Error} */ (e).message, true);
    return false;
  }
}

export { api, loginRedirect };
