import { Outlet, NavLink } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';
import { ThemeToggle } from '../components/ThemeToggle';

export function PublicLayout() {
  const { auth } = useAuth();

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
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <div style={{ background: 'var(--accent)', padding: '0.375rem', borderRadius: 8, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <span className="material-symbols-outlined" style={{ fontSize: 24 }}>account_balance_wallet</span>
          </div>
          <NavLink to="/" style={{ fontWeight: 800, fontSize: '1.25rem', color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>FluxMesh DEX</NavLink>
        </div>
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <ThemeToggle />
          {auth ? (
            <>
              <NavLink to="/trade" className="primary-btn" style={{ fontSize: '0.875rem', padding: '0.5rem 1rem' }}>
                Open Exchange
              </NavLink>
              {auth.role === 'admin' && (
                <NavLink to="/admin" className="secondary-btn" style={{ fontSize: '0.875rem', padding: '0.5rem 1rem' }}>
                  Admin
                </NavLink>
              )}
            </>
          ) : (
            <>
              <NavLink to="/login" style={{ fontWeight: 700, fontSize: '0.875rem', color: 'var(--accent)', textDecoration: 'none' }}>Log in</NavLink>
              <NavLink to="/login" className="primary-btn" style={{ fontSize: '0.875rem', padding: '0.5rem 1rem' }}>
                Join
              </NavLink>
            </>
          )}
        </div>
      </header>
      <main style={{ flex: 1, padding: '1.5rem', background: 'var(--bg-page)' }}>
        <Outlet />
      </main>
    </div>
  );
}
