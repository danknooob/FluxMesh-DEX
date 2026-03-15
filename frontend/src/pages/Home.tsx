import { Link } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

const features = [
  {
    icon: 'candlestick_chart',
    title: 'Real-time Order Book',
    description: 'Live depth and instant execution. WebSocket updates keep the book in sync across all clients.',
  },
  {
    icon: 'lock',
    title: 'JWT-Secured API',
    description: 'Per-user rate limiting and centralized auth at the gateway. Build on a production-ready REST API.',
  },
  {
    icon: 'psychology',
    title: 'MCP for AI',
    description: 'Model Context Protocol tools so AI assistants can query markets, balances, and system health.',
  },
];

export function Home() {
  const { auth } = useAuth();

  return (
    <div className="landing-font" style={{ fontFamily: 'Inter, sans-serif', paddingBottom: 100 }}>
      {/* Hero */}
      <section
        style={{
          position: 'relative',
          minHeight: 520,
          display: 'flex',
          flexDirection: 'column',
          gap: '1.5rem',
          overflow: 'hidden',
          borderRadius: 12,
          background: '#0f172a',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '1.5rem',
          textAlign: 'center',
          marginBottom: '2rem',
          boxShadow: '0 25px 50px -12px rgba(0,0,0,0.25)',
        }}
      >
        <div
          style={{
            position: 'absolute',
            inset: 0,
            opacity: 0.4,
            background: 'linear-gradient(to bottom right, rgba(37, 99, 235, 0.4), #0f172a)',
          }}
        />
        <div style={{ position: 'relative', zIndex: 1, display: 'flex', flexDirection: 'column', gap: '1rem', alignItems: 'center' }}>
          <span
            style={{
              display: 'inline-block',
              padding: '0.25rem 0.75rem',
              background: 'rgba(37, 99, 235, 0.2)',
              color: 'var(--accent)',
              fontSize: '0.75rem',
              fontWeight: 700,
              borderRadius: 9999,
              letterSpacing: '0.1em',
              textTransform: 'uppercase',
            }}
          >
            Event-Driven DEX
          </span>
          <h1
            style={{
              color: '#fff',
              fontSize: 'clamp(2rem, 5vw, 3.5rem)',
              fontWeight: 900,
              lineHeight: 1.15,
              letterSpacing: '-0.02em',
              margin: 0,
            }}
          >
            Trade on-chain.
            <br />
            <span style={{ color: 'var(--accent)' }}>Settle in real time.</span>
          </h1>
          <p
            style={{
              color: 'rgba(226, 232, 240, 0.9)',
              fontSize: '1rem',
              maxWidth: 420,
              margin: 0,
              lineHeight: 1.6,
            }}
          >
            Production-grade order-book exchange with Kafka, EVM settlement, and an MCP server for AI tools.
          </p>
        </div>
        <div
          style={{
            position: 'relative',
            zIndex: 1,
            display: 'flex',
            flexDirection: 'column',
            gap: '0.75rem',
            width: '100%',
            maxWidth: 360,
          }}
        >
          {auth ? (
            <Link
              to="/trade/markets"
              className="primary-btn"
              style={{
                height: 56,
                paddingLeft: '2rem',
                paddingRight: '2rem',
                fontSize: '1.125rem',
                borderRadius: 10,
                textDecoration: 'none',
              }}
            >
              Open Exchange
            </Link>
          ) : (
            <Link
              to="/login"
              className="primary-btn"
              style={{
                height: 56,
                paddingLeft: '2rem',
                paddingRight: '2rem',
                fontSize: '1.125rem',
                borderRadius: 10,
                textDecoration: 'none',
              }}
            >
              Open Account
            </Link>
          )}
          <Link
            to="/trade/markets"
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: 56,
              paddingLeft: '2rem',
              paddingRight: '2rem',
              background: 'rgba(255,255,255,0.1)',
              color: '#fff',
              fontSize: '1.125rem',
              fontWeight: 700,
              borderRadius: 10,
              border: '1px solid rgba(255,255,255,0.2)',
              textDecoration: 'none',
            }}
          >
            View Markets
          </Link>
        </div>
      </section>

      {/* Value prop */}
      <section
        style={{
          padding: '3rem 1rem',
          background: 'var(--bg-card)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div style={{ maxWidth: 480, margin: '0 auto', display: 'flex', flexDirection: 'column', alignItems: 'center', textAlign: 'center', gap: '1rem' }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: '0.25rem' }}>
            <span style={{ fontSize: '3rem', fontWeight: 900, color: 'var(--accent)' }}>Event-driven</span>
            <span style={{ fontSize: '1.25rem', fontWeight: 700, color: 'var(--text-muted)' }}> architecture</span>
          </div>
          <h3 style={{ fontSize: '1.5rem', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>
            Built for the next wave of on-chain trading
          </h3>
          <p style={{ color: 'var(--text-muted)', margin: 0, lineHeight: 1.6 }}>
            Kafka-backed order flow, Postgres for state, and WebSocket notifications. No polling — just real-time depth and fills.
          </p>
        </div>
      </section>

      {/* Features */}
      <section style={{ padding: '3rem 1rem' }}>
        <h2 style={{ fontSize: '1.5rem', fontWeight: 800, color: 'var(--text-primary)', marginBottom: '2rem' }}>
          Smarter stack for growth
        </h2>
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))',
            gap: '1rem',
          }}
        >
          {features.map((f) => (
            <div
              key={f.icon}
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '1rem',
                padding: '1.5rem',
                borderRadius: 12,
                border: '1px solid var(--border)',
                background: 'var(--bg-card)',
                boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
              }}
            >
              <div
                style={{
                  width: 48,
                  height: 48,
                  borderRadius: '50%',
                  background: 'rgba(37, 99, 235, 0.1)',
                  color: 'var(--accent)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <span className="material-symbols-outlined" style={{ fontSize: 28 }}>
                  {f.icon}
                </span>
              </div>
              <div>
                <h3 style={{ fontSize: '1.125rem', fontWeight: 700, color: 'var(--text-primary)', margin: '0 0 0.5rem 0' }}>
                  {f.title}
                </h3>
                <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', lineHeight: 1.5, margin: 0 }}>
                  {f.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </section>

      {/* Get started (plan-style cards) */}
      <section
        style={{
          padding: '3rem 1rem',
          background: 'var(--border-subtle)',
        }}
      >
        <div style={{ marginBottom: '2rem' }}>
          <h2 style={{ fontSize: '1.5rem', fontWeight: 800, color: 'var(--text-primary)', margin: '0 0 0.25rem 0' }}>
            Get started
          </h2>
          <p style={{ color: 'var(--text-muted)', margin: 0 }}>
            Trade on the exchange or integrate via API.
          </p>
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '1.5rem' }}>
          <div
            style={{
              padding: '1.5rem',
              borderRadius: 12,
              border: '1px solid var(--border)',
              background: 'var(--bg-card)',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '0.75rem' }}>
              <div>
                <h4 style={{ fontSize: '1.25rem', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>Trade</h4>
                <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', margin: '0.25rem 0 0 0' }}>
                  Use the web UI
                </p>
              </div>
            </div>
            <ul style={{ listStyle: 'none', padding: 0, margin: '0 0 1rem 0' }}>
              <li style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
                <span className="material-symbols-outlined" style={{ fontSize: 20, color: 'var(--accent)' }}>check_circle</span>
                Markets, order book, balances
              </li>
              <li style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', color: 'var(--text-muted)' }}>
                <span className="material-symbols-outlined" style={{ fontSize: 20, color: 'var(--accent)' }}>check_circle</span>
                Real-time WebSocket updates
              </li>
            </ul>
            <Link to={auth ? '/trade/markets' : '/login'} className="primary-btn" style={{ display: 'block', textAlign: 'center', textDecoration: 'none', padding: '0.75rem' }}>
              {auth ? 'Open Exchange' : 'Sign in to trade'}
            </Link>
          </div>

          <div
            style={{
              padding: '1.5rem',
              borderRadius: 12,
              border: '2px solid var(--accent)',
              background: 'var(--bg-card)',
              position: 'relative',
            }}
          >
            <div
              style={{
                position: 'absolute',
                top: 0,
                right: 0,
                background: 'var(--accent)',
                color: '#fff',
                fontSize: '0.625rem',
                fontWeight: 800,
                padding: '0.25rem 0.75rem',
                borderBottomLeftRadius: 8,
                textTransform: 'uppercase',
              }}
            >
              For builders
            </div>
            <div style={{ marginBottom: '0.75rem' }}>
              <h4 style={{ fontSize: '1.25rem', fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>Build</h4>
              <p style={{ fontSize: '0.875rem', color: 'var(--text-muted)', margin: '0.25rem 0 0 0' }}>
                API + MCP for AI
              </p>
            </div>
            <ul style={{ listStyle: 'none', padding: 0, margin: '0 0 1rem 0' }}>
              <li style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
                <span className="material-symbols-outlined" style={{ fontSize: 20, color: 'var(--accent)' }}>check_circle</span>
                REST API with JWT auth
              </li>
              <li style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
                <span className="material-symbols-outlined" style={{ fontSize: 20, color: 'var(--accent)' }}>check_circle</span>
                MCP tools for AI assistants
              </li>
              <li style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.875rem', color: 'var(--text-muted)' }}>
                <span className="material-symbols-outlined" style={{ fontSize: 20, color: 'var(--accent)' }}>check_circle</span>
                OpenAPI docs at /docs
              </li>
            </ul>
            <a
              href="/docs"
              target="_blank"
              rel="noopener noreferrer"
              className="secondary-btn"
              style={{
                display: 'block',
                textAlign: 'center',
                textDecoration: 'none',
                padding: '0.75rem',
                borderWidth: 2,
                borderColor: 'var(--accent)',
                color: 'var(--accent)',
              }}
            >
              View API docs
            </a>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer
        style={{
          padding: '2rem 1rem 4rem',
          borderTop: '1px solid var(--border)',
          background: 'var(--bg-header)',
        }}
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '2rem', marginBottom: '2rem' }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <h5 style={{ fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>Product</h5>
            <Link to="/trade/markets" style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Markets</Link>
            <Link to="/trade/balances" style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Balances</Link>
            <a href="/docs" target="_blank" rel="noopener noreferrer" style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>API Docs</a>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
            <h5 style={{ fontWeight: 700, color: 'var(--text-primary)', margin: 0 }}>Resources</h5>
            <a href="https://github.com/danknooob/FluxMesh-DEX" target="_blank" rel="noopener noreferrer" style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>GitHub</a>
            {auth ? (
              <Link to="/trade/profile" style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Profile</Link>
            ) : (
              <Link to="/login" style={{ fontSize: '0.875rem', color: 'var(--text-muted)' }}>Sign in</Link>
            )}
          </div>
        </div>
        <div style={{ paddingTop: '1.5rem', borderTop: '1px solid var(--border)', textAlign: 'center' }}>
          <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', margin: 0 }}>
            © {new Date().getFullYear()} FluxMesh DEX. Event-driven order-book exchange with Kafka and MCP.
          </p>
        </div>
      </footer>

      {/* Bottom nav (mobile-friendly) */}
      <nav
        style={{
          position: 'fixed',
          bottom: 0,
          left: 0,
          right: 0,
          zIndex: 50,
          display: 'flex',
          gap: '0.5rem',
          padding: '0.5rem 1rem 1.5rem',
          borderTop: '1px solid var(--border)',
          background: 'var(--bg-header)',
        }}
      >
        <Link
          to="/"
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '0.25rem',
            color: 'var(--accent)',
            textDecoration: 'none',
            fontSize: '0.75rem',
            fontWeight: 500,
          }}
        >
          <span className="material-symbols-outlined" style={{ fontSize: 28 }}>home</span>
          Home
        </Link>
        <Link
          to="/trade/markets"
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '0.25rem',
            color: 'var(--text-muted)',
            textDecoration: 'none',
            fontSize: '0.75rem',
            fontWeight: 500,
          }}
        >
          <span className="material-symbols-outlined" style={{ fontSize: 28 }}>candlestick_chart</span>
          Markets
        </Link>
        <Link
          to="/trade/balances"
          style={{
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '0.25rem',
            color: 'var(--text-muted)',
            textDecoration: 'none',
            fontSize: '0.75rem',
            fontWeight: 500,
          }}
        >
          <span className="material-symbols-outlined" style={{ fontSize: 28 }}>account_balance_wallet</span>
          Balances
        </Link>
        {auth ? (
          <Link
            to="/trade/profile"
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '0.25rem',
              color: 'var(--text-muted)',
              textDecoration: 'none',
              fontSize: '0.75rem',
              fontWeight: 500,
            }}
          >
            <span className="material-symbols-outlined" style={{ fontSize: 28 }}>person</span>
            Profile
          </Link>
        ) : (
          <Link
            to="/login"
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '0.25rem',
              color: 'var(--text-muted)',
              textDecoration: 'none',
              fontSize: '0.75rem',
              fontWeight: 500,
            }}
          >
            <span className="material-symbols-outlined" style={{ fontSize: 28 }}>login</span>
            Sign in
          </Link>
        )}
      </nav>
    </div>
  );
}
