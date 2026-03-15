import { FormEvent, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ThemeToggle } from '../components/ThemeToggle';

// Simple inline icons (Sellora-style)
const IconEnvelope = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0, color: '#64748b' }}>
    <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
    <polyline points="22,6 12,13 2,6" />
  </svg>
);
const IconLock = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ flexShrink: 0, color: '#64748b' }}>
    <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
    <path d="M7 11V7a5 5 0 0 1 10 0v4" />
  </svg>
);
const IconEye = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0, color: '#64748b' }}>
    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
    <circle cx="12" cy="12" r="3" />
  </svg>
);
const IconEyeOff = () => (
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0, color: '#64748b' }}>
    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
    <line x1="1" y1="1" x2="23" y2="23" />
  </svg>
);
const IconArrowRight = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0 }}>
    <line x1="5" y1="12" x2="19" y2="12" />
    <polyline points="12 5 19 12 12 19" />
  </svg>
);
const IconShield = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0 }}>
    <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
  </svg>
);
const IconLightning = () => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0 }}>
    <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
  </svg>
);
const LogoIcon = () => (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ flexShrink: 0 }}>
    <path d="M12 2L2 12l10 10 10-10L12 2z" />
    <path d="M12 6v6l4 2" />
  </svg>
);

export function Login() {
  const { login, register } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const [isRegister, setIsRegister] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [agreeTerms, setAgreeTerms] = useState(false);
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
    if (isRegister && !agreeTerms) {
      setError('Please agree to the Terms of Service and Privacy Policy');
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
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      className="login-root"
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'row',
        background: 'var(--bg-page)',
      }}
    >
      {/* Left: blue promotional panel — Sellora-style */}
      <div
        className="login-hero"
        style={{
          flex: '1 1 42%',
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          padding: '2.5rem 3rem',
          background: 'var(--login-hero-bg)',
        }}
      >
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '3rem' }}>
            <span style={{ color: '#fff' }}><LogoIcon /></span>
            <span style={{ fontSize: '1.35rem', fontWeight: 700, color: '#fff', letterSpacing: '-0.02em' }}>FluxMesh DEX</span>
          </div>
          <h1 style={{ fontSize: '2rem', fontWeight: 800, color: '#fff', lineHeight: 1.2, margin: 0, marginBottom: '1rem', letterSpacing: '-0.02em' }}>
            The world's most intuitive
            <br />
            order-book exchange.
          </h1>
          <p style={{ fontSize: '1rem', color: 'rgba(255,255,255,0.9)', margin: 0, marginBottom: '2.5rem', lineHeight: 1.6, maxWidth: 380 }}>
            Join traders on a fast, event-driven DEX. Real-time depth, JWT-secured API, and MCP tools for AI. Simple, fast, and secure.
          </p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '1.25rem' }}>
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '1rem' }}>
              <span style={{ color: 'rgba(255,255,255,0.95)', marginTop: 2 }}><IconLightning /></span>
              <div>
                <div style={{ fontWeight: 700, color: '#fff', fontSize: '1rem', marginBottom: 2 }}>Lightning Fast Setup</div>
                <div style={{ fontSize: '0.9rem', color: 'rgba(255,255,255,0.85)' }}>Start trading in under a minute.</div>
              </div>
            </div>
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '1rem' }}>
              <span style={{ color: 'rgba(255,255,255,0.95)', marginTop: 2 }}><IconShield /></span>
              <div>
                <div style={{ fontWeight: 700, color: '#fff', fontSize: '1rem', marginBottom: 2 }}>Enterprise Security</div>
                <div style={{ fontSize: '0.9rem', color: 'rgba(255,255,255,0.85)' }}>Your data and orders are always protected.</div>
              </div>
            </div>
          </div>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <div style={{ display: 'flex', marginRight: 4 }}>
            {[0, 1, 2].map((i) => (
              <div
                key={i}
                style={{
                  width: 36,
                  height: 36,
                  borderRadius: '50%',
                  background: 'rgba(255,255,255,0.3)',
                  border: '2px solid #1d4ed8',
                  marginLeft: i > 0 ? -10 : 0,
                  zIndex: 3 - i,
                }}
              />
            ))}
          </div>
          <span style={{ fontSize: '0.95rem', fontWeight: 600, color: '#fff' }}>+10k</span>
          <span style={{ fontSize: '0.9rem', color: 'rgba(255,255,255,0.9)', marginLeft: 4 }}>Join traders on FluxMesh today</span>
        </div>
      </div>

      {/* Right: form — theme-aware */}
      <div
        style={{
          flex: '1 1 58%',
          minHeight: '100vh',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '2rem',
          background: 'var(--bg-page)',
          position: 'relative',
        }}
      >
        <div style={{ position: 'absolute', top: '1.5rem', right: '1.5rem' }}>
          <ThemeToggle />
        </div>
        <div style={{ width: '100%', maxWidth: 400 }}>
          <h2 style={{ fontSize: '1.75rem', fontWeight: 700, color: 'var(--text-primary)', margin: 0, marginBottom: '0.25rem' }}>
            {isRegister ? 'Create Account' : 'Welcome back'}
          </h2>
          <p style={{ color: 'var(--text-muted)', marginBottom: '1.75rem', fontSize: '0.95rem' }}>
            {isRegister ? 'Join the FluxMesh community and start trading.' : 'Sign in to access the exchange.'}
          </p>

          <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1.1rem' }}>
            <label style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
              <span style={{ color: 'var(--text-primary)', fontSize: '0.85rem', fontWeight: 500 }}>Email Address</span>
              <div style={{ ...inputWrapStyle, borderColor: error ? 'var(--error)' : undefined }}>
                <IconEnvelope />
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="john@example.com"
                  required
                  style={inputStyle}
                />
              </div>
            </label>

            <label style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
              <span style={{ color: '#334155', fontSize: '0.85rem', fontWeight: 500 }}>Password</span>
              <div style={{ ...inputWrapStyle, borderColor: error ? '#ef4444' : undefined }}>
                <IconLock />
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="••••••••"
                  required
                  minLength={6}
                  style={{ ...inputStyle, paddingRight: 44 }}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((p) => !p)}
                  style={{ position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)', background: 'none', border: 'none', cursor: 'pointer', padding: 4 }}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? <IconEyeOff /> : <IconEye />}
                </button>
              </div>
            </label>

            {isRegister && (
              <>
                <label style={{ display: 'flex', flexDirection: 'column', gap: '0.35rem' }}>
                  <span style={{ color: 'var(--text-primary)', fontSize: '0.85rem', fontWeight: 500 }}>Confirm Password</span>
                  <div style={inputWrapStyle}>
                    <IconLock />
                    <input
                      type={showPassword ? 'text' : 'password'}
                      value={confirm}
                      onChange={(e) => setConfirm(e.target.value)}
                      placeholder="••••••••"
                      required
                      minLength={6}
                      style={inputStyle}
                    />
                  </div>
                </label>
                <label style={{ display: 'flex', alignItems: 'flex-start', gap: '0.5rem', cursor: 'pointer', fontSize: '0.9rem', color: 'var(--text-muted)' }}>
                  <input
                    type="checkbox"
                    checked={agreeTerms}
                    onChange={(e) => setAgreeTerms(e.target.checked)}
                    style={{ marginTop: 3, accentColor: 'var(--accent)' }}
                  />
                  <span>I agree to the <a href="#" style={{ color: 'var(--accent)', textDecoration: 'none' }} onClick={(e) => e.preventDefault()}>Terms of Service</a> and <a href="#" style={{ color: 'var(--accent)', textDecoration: 'none' }} onClick={(e) => e.preventDefault()}>Privacy Policy</a>.</span>
                </label>
              </>
            )}

            {error && (
              <p style={{ color: 'var(--error)', fontSize: '0.85rem', margin: 0, padding: '0.5rem 0' }}>{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="primary-btn"
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '0.5rem',
                padding: '0.75rem 1.25rem',
                marginTop: '0.25rem',
                opacity: loading ? 0.7 : 1,
              }}
            >
              {loading ? (isRegister ? 'Creating account…' : 'Signing in…') : (isRegister ? 'Create Account' : 'Sign in')}
              {!loading && <IconArrowRight />}
            </button>
          </form>

          <p style={{ textAlign: 'center', marginTop: '1.5rem', fontSize: '0.95rem', color: 'var(--text-muted)' }}>
            {isRegister ? 'Already have an account?' : "Don't have an account?"}{' '}
            <button
              type="button"
              onClick={() => { setIsRegister(!isRegister); setError(null); setAgreeTerms(false); }}
              style={{ background: 'none', border: 'none', color: 'var(--accent)', cursor: 'pointer', fontSize: '0.95rem', fontWeight: 600, padding: 0 }}
            >
              {isRegister ? 'Log in' : 'Register'}
            </button>
          </p>

          <div style={{ marginTop: '2rem', padding: '1rem', borderRadius: 10, background: 'var(--bg-card)', border: '1px solid var(--border)', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
            <strong style={{ color: 'var(--text-muted)' }}>Dev credentials</strong>
            <div style={{ marginTop: '0.35rem' }}>Trader: <code style={{ color: 'var(--text-primary)' }}>trader@example.com</code> / <code style={{ color: 'var(--text-primary)' }}>trader123</code></div>
            <div>Admin: <code style={{ color: 'var(--text-primary)' }}>admin@example.com</code> / <code style={{ color: 'var(--text-primary)' }}>admin123</code></div>
          </div>
        </div>

        <p style={{ position: 'absolute', bottom: '1.5rem', left: '50%', transform: 'translateX(-50%)', margin: 0, fontSize: '0.8rem', color: 'var(--text-muted)' }}>
          © {new Date().getFullYear()} FluxMesh DEX. All rights reserved.
        </p>
      </div>

      <style>{`
        @media (max-width: 768px) {
          .login-root { flex-direction: column !important; }
          .login-hero { min-height: auto !important; padding: 2rem 1.5rem !important; }
          .login-hero h1 { font-size: 1.5rem !important; }
        }
      `}</style>
    </div>
  );
}

const inputWrapStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: '0.75rem',
  padding: '0 0.85rem',
  borderRadius: 10,
  border: '1px solid var(--border)',
  background: 'var(--bg-input)',
  position: 'relative',
};
const inputStyle: React.CSSProperties = {
  flex: 1,
  padding: '0.65rem 0',
  border: 'none',
  background: 'transparent',
  color: 'var(--text-primary)',
  fontSize: '0.95rem',
  outline: 'none',
  fontFamily: 'inherit',
};
