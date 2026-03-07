import { Outlet, NavLink } from 'react-router-dom';

export function TraderLayout() {
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
          <NavLink to="/markets" style={({ isActive }) => ({ color: isActive ? '#38bdf8' : '#94a3b8' })}>Markets</NavLink>
          <NavLink to="/balances" style={({ isActive }) => ({ color: isActive ? '#38bdf8' : '#94a3b8' })}>Balances</NavLink>
        </nav>
      </header>
      <main style={{ flex: 1, padding: '1.5rem' }}>
        <Outlet />
      </main>
    </div>
  );
}
