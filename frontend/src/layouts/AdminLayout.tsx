import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ThemeToggle } from '../components/ThemeToggle';

export function AdminLayout() {
  const { auth, logout } = useAuth();
  const navigate = useNavigate();

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
        <NavLink to="/admin" style={{ fontWeight: 700, fontSize: '1.25rem', color: 'var(--text-primary)' }}>FluxMesh Admin</NavLink>
        <nav style={{ display: 'flex', gap: '1rem', marginLeft: '1rem' }}>
          <NavLink to="/admin/markets" style={({ isActive }) => ({ color: isActive ? 'var(--accent)' : 'var(--text-muted)', fontWeight: isActive ? 600 : 400 })}>Config</NavLink>
          <NavLink to="/admin/health" style={({ isActive }) => ({ color: isActive ? 'var(--accent)' : 'var(--text-muted)', fontWeight: isActive ? 600 : 400 })}>Health</NavLink>
        </nav>
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <ThemeToggle />
          {auth && (
            <span style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>
              {auth.email} <span style={{
                marginLeft: '0.3rem',
                padding: '0.15rem 0.4rem',
                borderRadius: 6,
                background: '#7c3aed',
                color: '#fff',
                fontSize: '0.7rem',
                fontWeight: 600,
                textTransform: 'uppercase',
              }}>admin</span>
            </span>
          )}
          <NavLink to="/" className="secondary-btn" style={{ fontSize: '0.8rem', padding: '0.35rem 0.8rem' }}>Home</NavLink>
          <NavLink to="/trade/markets" className="secondary-btn" style={{ fontSize: '0.8rem', padding: '0.35rem 0.8rem' }}>Trader UI</NavLink>
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
