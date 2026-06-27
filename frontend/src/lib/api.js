// Thin fetch wrapper for the panel API. Sends the session cookie automatically,
// attaches the CSRF token to unsafe requests, and redirects to the OIDC login
// flow on 401.

let csrfToken = '';

export function setCsrfToken(t) {
  csrfToken = t || '';
}

export function loginRedirect() {
  window.location.href = '/api/auth/login';
}

async function request(method, path, body) {
  /** @type {RequestInit} */
  const opts = {
    method,
    credentials: 'same-origin',
    headers: {}
  };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  if (method !== 'GET' && method !== 'HEAD') {
    opts.headers['X-CSRF-Token'] = csrfToken;
  }

  const res = await fetch(path, opts);
  if (res.status === 401) {
    loginRedirect();
    throw new Error('unauthorized');
  }
  if (!res.ok) {
    let msg = `request failed (${res.status})`;
    try {
      const data = await res.json();
      if (data && data.error) msg = data.error;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  const ct = res.headers.get('content-type') || '';
  return ct.includes('application/json') ? res.json() : null;
}

export const api = {
  me: () => request('GET', '/api/me'),
  getLock: () => request('GET', '/api/lock'),
  setLockState: (state) => request('POST', '/api/lock/state', { state }),
  setPin: (user, payload) => request('PUT', `/api/pins/${user}`, payload),
  deletePin: (user) => request('DELETE', `/api/pins/${user}`),
  setSoundVolume: (sound_volume) => request('POST', '/api/settings/sound-volume', { sound_volume }),
  setAutoRelock: (enabled) => request('POST', '/api/settings/auto-relock', { enabled }),
  refresh: () => request('POST', '/api/refresh'),
  logout: () => request('POST', '/api/auth/logout')
};
