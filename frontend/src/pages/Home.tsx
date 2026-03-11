import { Link } from 'react-router-dom';
import { useAuth } from '../auth/AuthContext';

export function Home() {
  const { auth } = useAuth();

  return (
    <div style={{ display: 'grid', gap: '1.8rem', maxWidth: 720 }}>
      <section>
        <h1 style={{ fontSize: '2.2rem', marginBottom: '0.5rem' }}>FluxMesh DEX</h1>
        <p style={{ color: '#94a3b8', marginBottom: '1rem' }}>
          Event-driven order-book DEX with Kafka, EVM settlement, and an MCP server so AI tools can inspect and
          operate the exchange in real time.
        </p>
        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap' }}>
          {auth ? (
            <>
              <Link to="/trade/markets" className="primary-btn">View markets</Link>
              {auth.role === 'admin' && (
                <Link to="/admin" className="secondary-btn">Control plane</Link>
              )}
            </>
          ) : (
            <Link to="/login" className="primary-btn">Sign in to trade</Link>
          )}
        </div>
      </section>
      <section style={{ display: 'grid', gap: '0.5rem' }}>
        <h2 style={{ fontSize: '1.1rem' }}>What&apos;s under the hood</h2>
        <ul style={{ paddingLeft: '1.2rem', color: '#cbd5f5', margin: 0 }}>
          <li>API Gateway with per-user rate limiting and centralized JWT auth</li>
          <li>Go REST API with Kafka topics for orders, matches, and settlements</li>
          <li>EVM smart contracts for batched settlement and market registry events</li>
          <li>React trader UI plus admin console for configuration and health</li>
          <li>MCP (Model Context Protocol) server so AI agents can query markets and system state</li>
        </ul>
      </section>
    </div>
  );
}
