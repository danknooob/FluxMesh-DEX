import { Outlet, NavLink } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

export function PublicLayout() {
  const { auth } = useAuth();

  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <header style={{
        padding: '1rem 1.5rem',
        borderBottom: '1px solid #334155',
        display: 'flex',
        alignItems: 'center',
        gap: '1.5rem',
      }}>
        <NavLink to="/" style={{ fontWeight: 700, fontSize: '1.25rem', color: '#e2e8f0' }}>FluxMesh DEX</NavLink>
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          {auth ? (
            <>
              <NavLink to="/trade" className="primary-btn" style={{ fontSize: '0.85rem', padding: '0.4rem 1rem' }}>
                Open Exchange
              </NavLink>
              {auth.role === 'admin' && (
                <NavLink to="/admin" className="secondary-btn" style={{ fontSize: '0.85rem', padding: '0.4rem 1rem' }}>
                  Admin
                </NavLink>
              )}
            </>
          ) : (
            <NavLink to="/login" className="primary-btn" style={{ fontSize: '0.85rem', padding: '0.4rem 1rem' }}>
              Sign in
            </NavLink>
          )}
        </div>
      </header>
      <main style={{ flex: 1, padding: '1.5rem' }}>
        <Outlet />
      </main>
    </div>
  );
}
