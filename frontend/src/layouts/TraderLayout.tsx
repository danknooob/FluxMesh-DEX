import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { useNotifications } from '../components/NotificationProvider';
import { ThemeToggle } from '../components/ThemeToggle';

export function TraderLayout() {
  const { auth, logout } = useAuth();
  const navigate = useNavigate();
  const { connected } = useNotifications();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', background: 'var(--bg-page)' }}>
      <header style={{
        padding: '1rem 1.5rem',
        borderBottom: '1px solid var(--border)',
        display: 'flex',
        alignItems: 'center',
        gap: '1.5rem',
        background: 'var(--bg-header)',
      }}>
        <NavLink to="/" style={{ fontWeight: 700, fontSize: '1.25rem', color: 'var(--text-primary)' }}>FluxMesh DEX</NavLink>
        <nav style={{ display: 'flex', gap: '1rem' }}>
          <NavLink to="/trade/markets" style={({ isActive }) => ({ color: isActive ? 'var(--accent)' : 'var(--text-muted)', fontWeight: isActive ? 600 : 400 })}>Markets</NavLink>
          <NavLink to="/trade/balances" style={({ isActive }) => ({ color: isActive ? 'var(--accent)' : 'var(--text-muted)', fontWeight: isActive ? 600 : 400 })}>Balances</NavLink>
        </nav>

        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          <ThemeToggle />
          <span title={connected ? 'Live: real-time updates' : 'Reconnecting…'}
            style={{
              width: 8, height: 8, borderRadius: '50%',
              background: connected ? '#22c55e' : '#ef4444',
              boxShadow: connected ? '0 0 6px #22c55e' : 'none',
              flexShrink: 0,
            }} />
          {auth && (
            <span style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>
              {auth.email} <span style={{
                marginLeft: '0.3rem',
                padding: '0.15rem 0.4rem',
                borderRadius: 6,
                background: auth.role === 'admin' ? '#7c3aed' : 'var(--accent)',
                color: '#fff',
                fontSize: '0.7rem',
                fontWeight: 600,
                textTransform: 'uppercase',
              }}>{auth.role}</span>
            </span>
          )}
          <NavLink to="/trade/profile" style={({ isActive }) => ({
            fontSize: '0.8rem', padding: '0.35rem 0.8rem', borderRadius: 8,
            border: '1px solid ' + (isActive ? 'var(--accent)' : 'var(--border)'),
            color: isActive ? 'var(--accent)' : 'var(--text-muted)',
            background: isActive ? 'var(--success-bg)' : 'transparent',
          })}>
            Profile
          </NavLink>
          {auth?.role === 'admin' && (
            <NavLink to="/admin" className="secondary-btn" style={{ fontSize: '0.8rem', padding: '0.35rem 0.8rem' }}>
              Admin
            </NavLink>
          )}
          <button
            onClick={handleLogout}
            style={{
              background: 'transparent',
              border: '1px solid var(--border)',
              color: 'var(--text-muted)',
              padding: '0.35rem 0.8rem',
              borderRadius: 8,
              fontSize: '0.8rem',
              cursor: 'pointer',
              transition: 'border-color 0.15s, color 0.15s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.borderColor = 'var(--error)'; e.currentTarget.style.color = 'var(--error)'; }}
            onMouseLeave={(e) => { e.currentTarget.style.borderColor = 'var(--border)'; e.currentTarget.style.color = 'var(--text-muted)'; }}
          >
            Logout
          </button>
        </div>
      </header>
      <main style={{ flex: 1, padding: '1.5rem', background: 'var(--bg-page)' }}>
        <Outlet />
      </main>
    </div>
  );
}
