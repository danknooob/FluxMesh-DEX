import { useEffect, useState } from 'react';
import { apiFetch } from '../../auth/api';

type Market = {
  id: string;
  base_asset: string;
  quote_asset: string;
  tick_size: string;
  min_size?: string;
  fee_rate: string;
  enabled: boolean;
};

export function AdminMarkets() {
  const [markets, setMarkets] = useState<Market[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    // Use API /markets (same data as control plane, works when API is running and has seeded markets)
    apiFetch('/api/markets')
      .then(async (r) => {
        if (!r.ok) {
          throw new Error('Failed to load markets config');
        }
        return (await r.json()) as Market[];
      })
      .then(setMarkets)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      <h1 style={{ marginBottom: '1rem', color: 'var(--text-primary)' }}>Config</h1>
      <p style={{ color: 'var(--text-muted)', marginBottom: '1rem' }}>
        Markets, risk limits, feature flags. Desired state is owned by the control plane and broadcast on{' '}
        <code style={{ background: 'var(--border-subtle)', padding: '0.1rem 0.35rem', borderRadius: 4 }}>control.config</code>.
      </p>

      {loading && <p style={{ color: 'var(--text-muted)' }}>Loading markets…</p>}
      {error && <p style={{ color: 'var(--error)' }}>{error}</p>}

      {!loading && !error && (
        markets.length === 0 ? (
          <p style={{ color: 'var(--text-muted)' }}>
            No markets yet. Start the API service so it can seed default markets (BTC-USDC, ETH-USDC, etc.).
          </p>
        ) : (
          <div style={{ border: '1px solid var(--border)', borderRadius: 12, overflow: 'hidden', background: 'var(--bg-card)' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9rem' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-page)' }}>
                  <th style={{ textAlign: 'left', padding: '0.6rem 0.8rem', color: 'var(--text-muted)', fontWeight: 500 }}>Market</th>
                  <th style={{ textAlign: 'left', padding: '0.6rem 0.8rem', color: 'var(--text-muted)', fontWeight: 500 }}>Tick</th>
                  <th style={{ textAlign: 'left', padding: '0.6rem 0.8rem', color: 'var(--text-muted)', fontWeight: 500 }}>Min size</th>
                  <th style={{ textAlign: 'left', padding: '0.6rem 0.8rem', color: 'var(--text-muted)', fontWeight: 500 }}>Fee</th>
                  <th style={{ textAlign: 'left', padding: '0.6rem 0.8rem', color: 'var(--text-muted)', fontWeight: 500 }}>Status</th>
                </tr>
              </thead>
              <tbody>
                {markets.map((m) => (
                  <tr key={m.id} style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                    <td style={{ padding: '0.6rem 0.8rem', color: 'var(--text-primary)' }}>
                      <strong>
                        {m.base_asset}/{m.quote_asset}
                      </strong>{' '}
                      <span style={{ color: 'var(--text-muted)' }}>({m.id})</span>
                    </td>
                    <td style={{ padding: '0.6rem 0.8rem', color: 'var(--text-primary)' }}>{m.tick_size}</td>
                    <td style={{ padding: '0.6rem 0.8rem', color: 'var(--text-primary)' }}>{m.min_size ?? '—'}</td>
                    <td style={{ padding: '0.6rem 0.8rem', color: 'var(--text-primary)' }}>{m.fee_rate}</td>
                    <td style={{ padding: '0.6rem 0.8rem', color: m.enabled ? 'var(--success)' : 'var(--error)' }}>
                      {m.enabled ? 'Enabled' : 'Disabled'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}
    </div>
  );
}
