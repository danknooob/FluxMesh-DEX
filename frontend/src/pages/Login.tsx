import { FormEvent, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

export function Login() {
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const [isRegister, setIsRegister] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const from = (location.state as { from?: { pathname: string } })?.from?.pathname || '/';

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);

    if (isRegister && password !== confirm) {
      setError('Passwords do not match');
      return;
    }
    if (password.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    setLoading(true);
    try {
      if (isRegister) {
        await register(email, password);
      } else {
        await login(email, password);
      }
      navigate(from, { replace: true });
    } catch (err: any) {
      setError(err?.message ?? 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: '#0f172a',
    }}>
      <div style={{
        width: '100%',
        maxWidth: 400,
        padding: '2.5rem 2rem',
        border: '1px solid #334155',
        borderRadius: 16,
        background: '#1e293b',
      }}>
        <h1 style={{ fontSize: '1.6rem', marginTop: 0, marginBottom: '0.3rem', textAlign: 'center' }}>
          FluxMesh DEX
        </h1>
        <p style={{ color: '#94a3b8', textAlign: 'center', marginBottom: '1.5rem', fontSize: '0.9rem' }}>
          {isRegister ? 'Create an account to start trading' : 'Sign in to access the exchange'}
        </p>

        <form onSubmit={onSubmit} style={{ display: 'grid', gap: '1rem' }}>
          <label style={{ display: 'grid', gap: '0.25rem' }}>
            <span style={{ color: '#cbd5f5', fontSize: '0.85rem' }}>Email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="you@example.com"
              required
              style={inputStyle}
            />
          </label>

          <label style={{ display: 'grid', gap: '0.25rem' }}>
            <span style={{ color: '#cbd5f5', fontSize: '0.85rem' }}>Password</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              required
              minLength={6}
              style={inputStyle}
            />
          </label>

          {isRegister && (
            <label style={{ display: 'grid', gap: '0.25rem' }}>
              <span style={{ color: '#cbd5f5', fontSize: '0.85rem' }}>Confirm password</span>
              <input
                type="password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                placeholder="••••••••"
                required
                minLength={6}
                style={inputStyle}
              />
            </label>
          )}

          {error && (
            <p style={{ color: '#f97373', fontSize: '0.85rem', margin: 0 }}>{error}</p>
          )}

          <button
            type="submit"
            className="primary-btn"
            disabled={loading}
            style={{ width: '100%', padding: '0.65rem', opacity: loading ? 0.7 : 1 }}
          >
            {loading
              ? (isRegister ? 'Creating account…' : 'Signing in…')
              : (isRegister ? 'Create account' : 'Sign in')}
          </button>
        </form>

        <p style={{ textAlign: 'center', marginTop: '1.2rem', fontSize: '0.85rem', color: '#94a3b8' }}>
          {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
          <button
            onClick={() => { setIsRegister(!isRegister); setError(null); }}
            style={{
              background: 'none',
              border: 'none',
              color: '#38bdf8',
              cursor: 'pointer',
              fontSize: '0.85rem',
              textDecoration: 'underline',
              padding: 0,
            }}
          >
            {isRegister ? 'Sign in' : 'Register'}
          </button>
        </p>

        <div style={{
          marginTop: '1rem',
          padding: '0.75rem',
          borderRadius: 8,
          background: '#0f172a',
          fontSize: '0.8rem',
          color: '#64748b',
        }}>
          <strong style={{ color: '#94a3b8' }}>Dev credentials</strong>
          <div style={{ marginTop: '0.3rem' }}>
            Trader: <code>trader@example.com</code> / <code>trader123</code>
          </div>
          <div>
            Admin: <code>admin@example.com</code> / <code>admin123</code>
          </div>
        </div>
      </div>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  padding: '0.55rem 0.7rem',
  borderRadius: 8,
  border: '1px solid #334155',
  background: '#020617',
  color: '#e2e8f0',
  fontSize: '0.95rem',
};
