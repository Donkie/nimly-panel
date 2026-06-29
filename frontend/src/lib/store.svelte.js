import { api, setCsrfToken } from './api.js';

/**
 * Central reactive app state — Svelte 5 universal reactivity.
 * Import this object in any component; reads are tracked automatically.
 */
export const app = $state({
  /** True after the first SSE state event is received. */
  ready: false,
  /** Authenticated user info, populated from GET /api/me. */
  me: /** @type {{ subject: string, name: string, email: string, groups: string[], csrf_token: string } | null} */ (null),
  /** Whether the SSE stream is currently live. */
  connected: false,
  /** Current lock state object from the most recent SSE event. */
  lock: /** @type {Record<string, unknown>} */ ({}),
  /** PIN slot array from the most recent SSE event. */
  pins: /** @type {unknown[]} */ ([]),
  /** Activity log (newest first, max 100). */
  events: /** @type {unknown[]} */ ([]),
  /** Active toast, or null. */
  toast: /** @type {{ message: string, isError: boolean } | null} */ (null)
});

/** @type {EventSource | null} */
let _sse = null;
/** @type {ReturnType<typeof setTimeout> | null} */
let _reconnectTimer = null;
/** @type {ReturnType<typeof setTimeout> | null} */
let _toastTimer = null;

/**
 * Display a transient toast that auto-dismisses after 3.5 s.
 * @param {string} message
 * @param {boolean} [isError]
 */
export function showToast(message, isError = false) {
  if (_toastTimer) clearTimeout(_toastTimer);
  app.toast = { message, isError };
  _toastTimer = setTimeout(() => { app.toast = null; }, 3500);
}

function connectSSE() {
  if (_sse) _sse.close();

  _sse = new EventSource('/api/stream', { withCredentials: true });

  _sse.addEventListener('state', (e) => {
    try {
      const data = JSON.parse(/** @type {MessageEvent} */(e).data);
      app.lock    = data.state  ?? {};
      app.pins    = data.pins   ?? [];
      app.events  = (data.events ?? []).slice(0, 100);
      app.connected = true;
      if (!app.ready) app.ready = true;
    } catch (err) {
      console.error('[nimlypanel] SSE parse error', err);
    }
  });

  _sse.onerror = () => {
    app.connected = false;
    _sse?.close();
    _sse = null;
    if (_reconnectTimer) clearTimeout(_reconnectTimer);
    _reconnectTimer = setTimeout(connectSSE, 3000);
  };
}

/**
 * Bootstrap the app. Call once in the root layout's onMount.
 * 1. Fetches /api/me (redirects to login on 401).
 * 2. Opens the SSE stream.
 */
export async function bootstrap() {
  let me;
  try {
    me = await api.get('/api/me');
  } catch {
    return; // 401 → redirect is handled inside api.get
  }
  if (!me) return;
  app.me = me;
  setCsrfToken(/** @type {any} */(me).csrf_token);
  connectSSE();
}
