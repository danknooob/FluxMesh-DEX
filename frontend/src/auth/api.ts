const STORAGE_KEY = 'fluxmesh_auth';

function getToken(): string | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return (JSON.parse(raw) as { token: string }).token;
  } catch {
    return null;
  }
}

/**
 * Thin wrapper around fetch that auto-attaches the JWT Bearer token.
 * If the response is 401 it clears stored auth and redirects to /login.
 */
export async function apiFetch(input: string, init?: RequestInit): Promise<Response> {
  const token = getToken();
  const headers = new Headers(init?.headers);

  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const res = await fetch(input, { ...init, headers });

  if (res.status === 401) {
    localStorage.removeItem(STORAGE_KEY);
    window.location.href = '/login';
  }

  return res;
}
