import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { apiFetch } from '../auth/api';

type Market = {
  id: string;
  base_asset: string;
  quote_asset: string;
  tick_size: string;
  fee_rate: string;
  enabled: boolean;
};

const cardStyle: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 12,
  padding: '1.25rem 1.5rem',
  color: 'inherit',
  textDecoration: 'none',
  display: 'flex',
  flexDirection: 'column',
  gap: '0.75rem',
  transition: 'border-color 0.2s ease, box-shadow 0.2s ease, transform 0.15s ease',
};
const cardHover = {
  borderColor: 'var(--accent)',
  boxShadow: '0 4px 20px rgba(37, 99, 235, 0.15)',
  transform: 'translateY(-2px)',
};

export function Markets() {
  const [markets, setMarkets] = useState<Market[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiFetch('/api/markets')
      .then((r) => r.json())
      .then(setMarkets)
      .catch(() => setMarkets([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div style={{ maxWidth: 960, margin: '0 auto' }}>
        <div style={{ marginBottom: '1.5rem' }}>
          <div style={{ height: 28, width: 140, background: '#e2e8f0', borderRadius: 6, marginBottom: 6 }} />
          <div style={{ height: 16, width: 320, background: '#e2e8f0', borderRadius: 4, opacity: 0.7 }} />
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1rem' }}>
          {[1, 2, 3, 4].map((i) => (
            <div
              key={i}
              style={{
                ...cardStyle,
                minHeight: 120,
                background: '#f8fafc',
                border: '1px solid #e2e8f0',
              }}
            >
              <div style={{ height: 24, width: '60%', background: '#e2e8f0', borderRadius: 4 }} />
              <div style={{ height: 14, width: '80%', background: '#e2e8f0', borderRadius: 4, opacity: 0.6 }} />
              <div style={{ height: 36, width: 100, background: '#e2e8f0', borderRadius: 8, marginTop: 8 }} />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 960, margin: '0 auto' }}>
      <header style={{ marginBottom: '1.75rem' }}>
        <h1
          style={{
            fontSize: '1.75rem',
            fontWeight: 700,
            letterSpacing: '-0.02em',
            margin: 0,
            marginBottom: '0.35rem',
            color: 'var(--text-primary)',
          }}
        >
          Markets
        </h1>
        <p
          style={{
            margin: 0,
            fontSize: '0.95rem',
            color: 'var(--text-muted)',
            fontWeight: 400,
          }}
        >
          Spot pairs on FluxMesh DEX · Query via MCP in Cursor
        </p>
      </header>

      {markets.length === 0 && (
        <div
          style={{
            padding: '2.5rem',
            textAlign: 'center',
            border: '1px dashed var(--border)',
            borderRadius: 12,
            color: 'var(--text-muted)',
            background: 'var(--bg-card)',
          }}
        >
          No markets yet. Seed via API or Admin.
        </div>
      )}

      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
          gap: '1rem',
        }}
      >
        {markets.map((m) => (
          <Link
            key={m.id}
            to={`/trade/markets/${encodeURIComponent(m.id)}`}
            style={cardStyle}
            onMouseEnter={(e) => {
              Object.assign(e.currentTarget.style, cardHover);
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border)';
              e.currentTarget.style.boxShadow = 'none';
              e.currentTarget.style.transform = 'none';
            }}
          >
            <div style={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', flexWrap: 'wrap', gap: '0.5rem' }}>
              <span style={{ fontSize: '1.25rem', fontWeight: 700, color: 'var(--text-primary)' }}>
                {m.base_asset}<span style={{ color: 'var(--text-muted)', fontWeight: 500 }}>/</span>{m.quote_asset}
              </span>
              {!m.enabled && (
                <span
                  style={{
                    fontSize: '0.7rem',
                    padding: '0.2rem 0.5rem',
                    borderRadius: 6,
                    background: 'var(--error-bg)',
                    color: 'var(--error)',
                    fontWeight: 600,
                    textTransform: 'uppercase',
                  }}
                >
                  Paused
                </span>
              )}
            </div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
              Tick <span style={{ color: 'var(--text-primary)' }}>{m.tick_size}</span>
              <span style={{ margin: '0 0.5rem', color: 'var(--text-muted-2)' }}>·</span>
              Fee <span style={{ color: 'var(--text-primary)' }}>{m.fee_rate}</span>
            </div>
            <div style={{ marginTop: '0.25rem' }}>
              <span
                style={{
                  display: 'inline-block',
                  fontSize: '0.85rem',
                  fontWeight: 600,
                  padding: '0.5rem 1rem',
                  borderRadius: 8,
                  background: 'var(--success-bg)',
                  color: 'var(--accent)',
                  transition: 'background 0.15s, color 0.15s',
                }}
              >
                Trade →
              </span>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}
