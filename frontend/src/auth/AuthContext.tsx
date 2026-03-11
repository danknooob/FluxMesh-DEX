import { createContext, useContext, useState, useCallback, useMemo, type ReactNode } from 'react';

interface AuthState {
  token: string;
  role: string;
  email: string;
}

interface AuthContextValue {
  auth: AuthState | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

const STORAGE_KEY = 'fluxmesh_auth';

function loadFromStorage(): AuthState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as AuthState;
  } catch {
    return null;
  }
}

function saveToStorage(state: AuthState) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

function clearStorage() {
  localStorage.removeItem(STORAGE_KEY);
}

async function handleAuthResponse(res: Response, email: string): Promise<AuthState> {
  if (!res.ok) {
    const msg = await res.text();
    throw new Error(msg || 'Authentication failed');
  }
  const data = await res.json();
  return { token: data.access_token, role: data.role, email };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState | null>(loadFromStorage);

  const login = useCallback(async (email: string, password: string) => {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    const state = await handleAuthResponse(res, email);
    saveToStorage(state);
    setAuth(state);
  }, []);

  const register = useCallback(async (email: string, password: string) => {
    const res = await fetch('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, role: 'trader' }),
    });
    const state = await handleAuthResponse(res, email);
    saveToStorage(state);
    setAuth(state);
  }, []);

  const logout = useCallback(() => {
    clearStorage();
    setAuth(null);
  }, []);

  const value = useMemo(() => ({ auth, login, register, logout }), [auth, login, register, logout]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be inside AuthProvider');
  return ctx;
}
