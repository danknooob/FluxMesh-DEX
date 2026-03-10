import { Outlet, NavLink } from 'react-router-dom';

export function AdminLayout() {
  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column' }}>
      <header style={{
        padding: '1rem 1.5rem',
        borderBottom: '1px solid #334155',
        display: 'flex',
        alignItems: 'center',
        gap: '1.5rem',
      }}>
        <NavLink to="/admin" style={{ fontWeight: 700, fontSize: '1.25rem' }}>FluxMesh Admin</NavLink>
        <nav style={{ display: 'flex', gap: '1rem', marginLeft: '1rem' }}>
          <NavLink to="/admin/markets" style={({ isActive }) => ({ color: isActive ? '#38bdf8' : '#94a3b8' })}>Config</NavLink>
          <NavLink to="/admin/health" style={({ isActive }) => ({ color: isActive ? '#38bdf8' : '#94a3b8' })}>Health</NavLink>
        </nav>
        <div style={{ marginLeft: 'auto', display: 'flex', gap: '0.5rem' }}>
          <NavLink to="/" className="secondary-btn">Home</NavLink>
          <NavLink to="/markets" className="secondary-btn">Trader UI</NavLink>
        </div>
      </header>
      <main style={{ flex: 1, padding: '1.5rem' }}>
        <Outlet />
      </main>
    </div>
  );
}
