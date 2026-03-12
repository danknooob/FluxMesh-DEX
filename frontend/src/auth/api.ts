const STORAGE_KEY = 'fluxmesh_auth';

const MAX_RETRIES = 3;
const BASE_DELAY_MS = 500;
const RETRYABLE_STATUS = new Set([502, 503, 504]);

function getToken(): string | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return (JSON.parse(raw) as { token: string }).token;
  } catch {
    return null;
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

function isRetryable(method: string): boolean {
  const m = method.toUpperCase();
  return m === 'GET' || m === 'HEAD' || m === 'OPTIONS';
}

/**
 * Wrapper around fetch with JWT auto-attach, 401 redirect, and automatic
 * retry with exponential backoff for transient failures (network errors,
 * 502/503/504). Mutating requests (POST/PUT/DELETE) are only retried on
 * network errors — never on a received server response — to avoid
 * duplicate side-effects.
 */
export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
  const token = getToken();
  const headers = new Headers(init?.headers);
  const method = init?.method ?? 'GET';

  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  let lastError: unknown;

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    try {
      const res = await fetch(input, { ...init, headers });

      if (res.status === 401) {
        localStorage.removeItem(STORAGE_KEY);
        window.location.href = '/login';
        return res;
      }

      if (RETRYABLE_STATUS.has(res.status) && isRetryable(method) && attempt < MAX_RETRIES) {
        const delay = BASE_DELAY_MS * 2 ** attempt + Math.random() * 100;
        console.warn(`[apiFetch] ${res.status} on ${method} ${input}, retry ${attempt + 1}/${MAX_RETRIES} in ${Math.round(delay)}ms`);
        await sleep(delay);
        continue;
      }

      return res;
    } catch (err) {
      lastError = err;

      if (attempt < MAX_RETRIES) {
        const delay = BASE_DELAY_MS * 2 ** attempt + Math.random() * 100;
        console.warn(`[apiFetch] network error on ${method} ${input}, retry ${attempt + 1}/${MAX_RETRIES} in ${Math.round(delay)}ms`, err);
        await sleep(delay);
        continue;
      }
    }
  }

  throw lastError;
}
