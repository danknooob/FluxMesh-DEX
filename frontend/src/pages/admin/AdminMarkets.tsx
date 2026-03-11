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
    apiFetch('/control/admin/markets')
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
      <h1 style={{ marginBottom: '1rem' }}>Config</h1>
      <p style={{ color: '#94a3b8', marginBottom: '1rem' }}>
        Markets, risk limits, feature flags. Desired state is owned by the control plane and broadcast on{' '}
        <code>control.config</code>.
      </p>

      {loading && <p>Loading markets…</p>}
      {error && <p style={{ color: '#f97373' }}>{error}</p>}

      {!loading && !error && (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9rem' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid #334155' }}>
              <th style={{ textAlign: 'left', padding: '0.4rem' }}>Market</th>
              <th style={{ textAlign: 'left', padding: '0.4rem' }}>Tick</th>
              <th style={{ textAlign: 'left', padding: '0.4rem' }}>Min size</th>
              <th style={{ textAlign: 'left', padding: '0.4rem' }}>Fee</th>
              <th style={{ textAlign: 'left', padding: '0.4rem' }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {markets.map((m) => (
              <tr key={m.id} style={{ borderBottom: '1px solid #1e293b' }}>
                <td style={{ padding: '0.4rem' }}>
                  <strong>
                    {m.base_asset}/{m.quote_asset}
                  </strong>{' '}
                  <span style={{ color: '#64748b' }}>({m.id})</span>
                </td>
                <td style={{ padding: '0.4rem' }}>{m.tick_size}</td>
                <td style={{ padding: '0.4rem' }}>{m.min_size ?? '—'}</td>
                <td style={{ padding: '0.4rem' }}>{m.fee_rate}</td>
                <td style={{ padding: '0.4rem', color: m.enabled ? '#4ade80' : '#f97373' }}>
                  {m.enabled ? 'Enabled' : 'Disabled'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
