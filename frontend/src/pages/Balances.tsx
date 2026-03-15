import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '../auth/api';
import { useNotifications } from '../components/NotificationProvider';

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
  const [authError, setAuthError] = useState(false);
  const { subscribe } = useNotifications();

  const fetchBalances = useCallback(() => {
    if (authError) return;
    setError(null);
    apiFetch('/api/balances')
      .then(async (r) => {
        if (r.status === 401) {
          setAuthError(true);
          throw new Error('Unauthorized');
        }
        if (!r.ok) throw new Error(await r.text());
        setAuthError(false);
        return r.json() as Promise<Balance[] | null>;
      })
      .then((data) => setBalances(Array.isArray(data) ? data : []))
      .catch((err) => {
        setError(err?.message ?? 'Failed to load balances');
      })
      .finally(() => setLoading(false));
  }, [authError]);

  useEffect(() => { fetchBalances(); }, [fetchBalances]);

  useEffect(() => {
    return subscribe((msg) => {
      if (authError) return;
      if (msg.type === 'balance_updated' || msg.type === 'order_filled') {
        fetchBalances();
      }
    });
  }, [subscribe, fetchBalances, authError]);

  const total = (balances ?? []).reduce((acc, b) => acc + parseFloat(b.available ?? '0') + parseFloat(b.locked ?? '0'), 0);

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '1rem', marginBottom: '1.5rem' }}>
        <h1 style={{ margin: 0, color: 'var(--text-primary)' }}>Balances</h1>
        {!loading && (balances ?? []).length > 0 && (
          <span style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>
            {(balances ?? []).length} asset{(balances ?? []).length !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {loading && <p style={{ color: 'var(--text-muted)' }}>Loading balances…</p>}
      {error && <p style={{ color: 'var(--error)' }}>{error}</p>}

      {!loading && !error && balances.length === 0 && (
        <div style={{
          border: '1px solid var(--border)',
          borderRadius: 12,
          padding: '2rem',
          textAlign: 'center',
          color: 'var(--text-muted)',
          background: 'var(--bg-card)',
        }}>
          <p style={{ fontSize: '1.1rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>No balances yet</p>
          <p style={{ fontSize: '0.85rem' }}>
            Balances are populated when trades settle through the matching engine and settlement pipeline.
            Place an order to get started.
          </p>
        </div>
      )}

      {(balances ?? []).length > 0 && (
        <>
          <div style={{
            display: 'grid',
            gridTemplateColumns: '1fr 1fr',
            gap: '1rem',
            marginBottom: '1.5rem',
          }}>
            <div style={{
              border: '1px solid var(--border)',
              borderRadius: 12,
              padding: '1rem 1.25rem',
              background: 'var(--bg-card)',
            }}>
              <div style={{ color: 'var(--text-muted)', fontSize: '0.8rem', marginBottom: '0.25rem' }}>Total Value</div>
              <div style={{ fontSize: '1.5rem', fontWeight: 700, color: 'var(--text-primary)' }}>
                {total.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
              </div>
            </div>
            <div style={{
              border: '1px solid var(--border)',
              borderRadius: 12,
              padding: '1rem 1.25rem',
              background: 'var(--bg-card)',
            }}>
              <div style={{ color: 'var(--text-muted)', fontSize: '0.8rem', marginBottom: '0.25rem' }}>Assets Held</div>
              <div style={{ fontSize: '1.5rem', fontWeight: 700, color: 'var(--text-primary)' }}>{(balances ?? []).length}</div>
            </div>
          </div>

          <div style={{
            border: '1px solid var(--border)',
            borderRadius: 12,
            overflow: 'hidden',
            background: 'var(--bg-card)',
          }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-page)' }}>
                  <th style={thStyle}>Asset</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Available</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Locked</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Total</th>
                </tr>
              </thead>
              <tbody>
                {(balances ?? []).map((b) => {
                  const avail = parseFloat(b.available ?? '0');
                  const locked = parseFloat(b.locked ?? '0');
                  return (
                    <tr key={b.asset ?? ''} style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                      <td style={tdStyle}>
                        <span style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{b.asset ?? '—'}</span>
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: 'var(--text-primary)' }}>
                        {avail.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: locked > 0 ? 'var(--success)' : 'var(--text-muted)' }}>
                        {locked.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
                      </td>
                      <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', fontWeight: 600, color: 'var(--text-primary)' }}>
                        {(avail + locked).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <p style={{ color: 'var(--text-muted)', fontSize: '0.8rem', marginTop: '0.75rem' }}>
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
  color: 'var(--text-muted)',
  fontWeight: 500,
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
};

const tdStyle: React.CSSProperties = {
  padding: '0.75rem 1rem',
  fontSize: '0.95rem',
};
