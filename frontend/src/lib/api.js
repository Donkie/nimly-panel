/** @type {string} */
let csrfToken = '';

/** @param {string} token */
export function setCsrfToken(token) {
  csrfToken = token;
}

/**
 * @param {'GET'|'POST'|'PUT'|'DELETE'} method
 * @param {string} path
 * @param {unknown} [body]
 * @returns {Promise<unknown>}
 */
async function request(method, path, body) {
  /** @type {Record<string, string>} */
  const headers = {};

  if (method !== 'GET') {
    headers['Content-Type'] = 'application/json';
    headers['X-CSRF-Token'] = csrfToken;
  }

  const res = await fetch(path, {
    method,
    credentials: 'include',
    headers,
    ...(body !== undefined ? { body: JSON.stringify(body) } : {})
  });

  if (res.status === 401) {
    window.location.href = '/api/auth/login';
    return null;
  }

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const json = await res.json();
      msg = json.message || json.error || msg;
    } catch (_) { /* ignore */ }
    throw new Error(msg);
  }

  return res.json();
}

export const api = {
  /** @param {string} path */
  get: (path) => request('GET', path),
  /** @param {string} path @param {unknown} body */
  post: (path, body) => request('POST', path, body),
  /** @param {string} path @param {unknown} body */
  put: (path, body) => request('PUT', path, body),
  /** @param {string} path */
  delete: (path) => request('DELETE', path)
};
