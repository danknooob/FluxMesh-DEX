import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

export function TraderLayout() {
  const { auth, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <header style={{
        padding: '1rem 1.5rem',
        borderBottom: '1px solid #334155',
        display: 'flex',
        alignItems: 'center',
        gap: '1.5rem',
      }}>
        <NavLink to="/" style={{ fontWeight: 700, fontSize: '1.25rem' }}>FluxMesh DEX</NavLink>
        <nav style={{ display: 'flex', gap: '1rem' }}>
          <NavLink to="/trade/markets" style={({ isActive }) => ({ color: isActive ? '#38bdf8' : '#94a3b8' })}>Markets</NavLink>
          <NavLink to="/trade/balances" style={({ isActive }) => ({ color: isActive ? '#38bdf8' : '#94a3b8' })}>Balances</NavLink>
        </nav>

        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
          {auth && (
            <span style={{ color: '#64748b', fontSize: '0.85rem' }}>
              {auth.email} <span style={{
                marginLeft: '0.3rem',
                padding: '0.15rem 0.4rem',
                borderRadius: 4,
                background: auth.role === 'admin' ? '#7c3aed' : '#0ea5e9',
                color: '#fff',
                fontSize: '0.7rem',
                fontWeight: 600,
                textTransform: 'uppercase',
              }}>{auth.role}</span>
            </span>
          )}
          {auth?.role === 'admin' && (
            <NavLink to="/admin" className="secondary-btn" style={{ fontSize: '0.8rem', padding: '0.35rem 0.8rem' }}>
              Admin
            </NavLink>
          )}
          <button
            onClick={handleLogout}
            style={{
              background: 'transparent',
              border: '1px solid #475569',
              color: '#94a3b8',
              padding: '0.35rem 0.8rem',
              borderRadius: 8,
              fontSize: '0.8rem',
              cursor: 'pointer',
              transition: 'border-color 0.15s, color 0.15s',
            }}
            onMouseEnter={(e) => { e.currentTarget.style.borderColor = '#f97373'; e.currentTarget.style.color = '#f97373'; }}
            onMouseLeave={(e) => { e.currentTarget.style.borderColor = '#475569'; e.currentTarget.style.color = '#94a3b8'; }}
          >
            Logout
          </button>
        </div>
      </header>
      <main style={{ flex: 1, padding: '1.5rem' }}>
        <Outlet />
      </main>
    </div>
  );
}
