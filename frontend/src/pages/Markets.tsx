import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { apiFetch } from '../auth/api';

type Market = { id: string; base_asset: string; quote_asset: string; tick_size: string; fee_rate: string; enabled: boolean };

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

  if (loading) return <p>Loading markets…</p>;

  return (
    <div>
      <h1 style={{ marginBottom: '1rem' }}>Markets</h1>
      <div style={{ display: 'grid', gap: '0.75rem' }}>
        {markets.length === 0 && <p>No markets yet. Seed via API or Admin.</p>}
        {markets.map((m) => (
          <Link
            key={m.id}
            to={`/trade/markets/${encodeURIComponent(m.id)}`}
            style={{
              display: 'block',
              padding: '1rem',
              border: '1px solid #334155',
              borderRadius: '8px',
              color: 'inherit',
            }}
          >
            <strong>{m.base_asset}/{m.quote_asset}</strong>
            <span style={{ marginLeft: '0.5rem', color: '#64748b' }}>tick {m.tick_size} · fee {m.fee_rate}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}
