import { useEffect, useState } from 'react';
import { apiFetch } from '../auth/api';

type Balance = {
  user_id: string;
  asset: string;
  available: string;
  locked: string;
  updated_at: string;
};

export function Balances() {
  const [balances, setBalances] = useState<Balance[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetch('/api/balances')
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json() as Promise<Balance[]>;
      })
      .then(setBalances)
      .catch((err) => setError(err?.message ?? 'Failed to load balances'))
      .finally(() => setLoading(false));
  }, []);

  const total = balances.reduce((acc, b) => acc + parseFloat(b.available || '0') + parseFloat(b.locked || '0'), 0);

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '1rem', marginBottom: '1.5rem' }}>
        <h1 style={{ margin: 0 }}>Balances</h1>
        {!loading && balances.length > 0 && (
          <span style={{ color: '#64748b', fontSize: '0.9rem' }}>
            {balances.length} asset{balances.length !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {loading && <p style={{ color: '#94a3b8' }}>Loading balances…</p>}
      {error && <p style={{ color: '#f97373' }}>{error}</p>}

      {!loading && !error && balances.length === 0 && (
        <div style={{
          border: '1px solid #334155',
          borderRadius: 12,
          padding: '2rem',
          textAlign: 'center',
          color: '#94a3b8',
        }}>
          <p style={{ fontSize: '1.1rem', marginBottom: '0.5rem' }}>No balances yet</p>
          <p style={{ fontSize: '0.85rem' }}>
            Balances are populated when trades settle through the matching engine and settlement pipeline.
            Place an order to get started.
          </p>
        </div>
      )}

      {balances.length > 0 && (
        <>
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '1rem',
            marginBottom: '1.5rem',
          }}>
            <div style={{
              border: '1px solid #334155',
              borderRadius: 12,
              padding: '1rem 1.25rem',
            }}>
              <div style={{ color: '#64748b', fontSize: '0.8rem', marginBottom: '0.25rem' }}>Total Value</div>
              <div style={{ fontSize: '1.5rem', fontWeight: 700 }}>
                {total.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
              </div>
            </div>
            <div style={{
              border: '1px solid #334155',
              borderRadius: 12,
              padding: '1rem 1.25rem',
            }}>
              <div style={{ color: '#64748b', fontSize: '0.8rem', marginBottom: '0.25rem' }}>Assets Held</div>
              <div style={{ fontSize: '1.5rem', fontWeight: 700 }}>{balances.length}</div>
            </div>
          </div>

          <div style={{
            border: '1px solid #334155',
            borderRadius: 12,
            overflow: 'hidden',
          }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #334155', background: '#0f172a' }}>
                  <th style={thStyle}>Asset</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Available</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Locked</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Total</th>
                </tr>
              </thead>
              <tbody>
                {balances.map((b) => {
                  const avail = parseFloat(b.available || '0');
                  const locked = parseFloat(b.locked || '0');
                  return (
                    <tr key={b.asset} style={{ borderBottom: '1px solid #1e293b' }}>
                      <td style={tdStyle}>
                        <span style={{ fontWeight: 600 }}>{b.asset}</span>
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>
                        {avail.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: locked > 0 ? '#fbbf24' : '#475569' }}>
                        {locked.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', fontWeight: 600 }}>
                        {(avail + locked).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <p style={{ color: '#475569', fontSize: '0.8rem', marginTop: '0.75rem' }}>
            Balances are updated asynchronously via Kafka as trades settle. Locked amounts represent funds reserved in open orders.
          </p>
        </>
      )}
    </div>
  );
}

const thStyle: React.CSSProperties = {
  padding: '0.75rem 1rem',
  textAlign: 'left',
  fontSize: '0.8rem',
  color: '#94a3b8',
  fontWeight: 500,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
};

const tdStyle: React.CSSProperties = {
  padding: '0.75rem 1rem',
  fontSize: '0.95rem',
};
