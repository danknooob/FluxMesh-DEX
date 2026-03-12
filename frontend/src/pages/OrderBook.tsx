import { FormEvent, useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { apiFetch } from '../auth/api';
import { useNotifications } from '../components/NotificationProvider';

type Market = {
  id: string;
  base_asset: string;
  quote_asset: string;
  tick_size: string;
  min_size?: string;
  fee_rate: string;
  enabled: boolean;
};

type PriceLevel = { price: string; total_size: string; count: number };
type Depth = { bids: PriceLevel[]; asks: PriceLevel[] };

type Order = {
  id: string;
  market_id: string;
  side: 'buy' | 'sell';
  price: string;
  size: string;
  remaining: string;
  status: string;
  created_at: string;
};

const POLL_INTERVAL = 5000;

export function OrderBook() {
  const { marketId } = useParams<{ marketId: string }>();

  const [market, setMarket] = useState<Market | null>(null);
  const [loading, setLoading] = useState(true);
  const [depth, setDepth] = useState<Depth | null>(null);
  const [myOrders, setMyOrders] = useState<Order[]>([]);
  const [cancelling, setCancelling] = useState<string | null>(null);

  const { subscribe } = useNotifications();

  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState('');
  const [size, setSize] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!marketId) return;
    setLoading(true);
    apiFetch(`/api/markets/${encodeURIComponent(marketId)}`)
      .then(async (r) => {
        if (!r.ok) throw new Error('Market not found');
        return (await r.json()) as Market;
      })
      .then(setMarket)
      .catch(() => setMarket(null))
      .finally(() => setLoading(false));
  }, [marketId]);

  const fetchDepth = useCallback(() => {
    if (!marketId) return;
    apiFetch(`/api/markets/${encodeURIComponent(marketId)}/depth?limit=15`)
      .then((r) => (r.ok ? (r.json() as Promise<Depth>) : Promise.resolve(null)))
      .then((d) => { if (d) setDepth(d); })
      .catch(() => {});
  }, [marketId]);

  const fetchMyOrders = useCallback(() => {
    if (!marketId) return;
    apiFetch(`/api/orders?market_id=${encodeURIComponent(marketId)}`)
      .then((r) => (r.ok ? (r.json() as Promise<Order[]>) : Promise.resolve([])))
      .then(setMyOrders)
      .catch(() => {});
  }, [marketId]);

  useEffect(() => {
    fetchDepth();
    fetchMyOrders();
    const id = setInterval(() => { fetchDepth(); fetchMyOrders(); }, POLL_INTERVAL);
    return () => clearInterval(id);
  }, [fetchDepth, fetchMyOrders]);

  useEffect(() => {
    return subscribe((msg) => {
      if (msg.type === 'order_filled' || msg.type === 'order_cancelled') {
        fetchDepth();
        fetchMyOrders();
      }
    });
  }, [subscribe, fetchDepth, fetchMyOrders]);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!marketId) return;
    setSubmitting(true);
    setError(null);
    setMessage(null);
    const idempotencyKey = crypto.randomUUID();
    try {
      const res = await apiFetch('/api/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': idempotencyKey },
        body: JSON.stringify({ market_id: marketId, side, price, size }),
      });
      if (!res.ok) throw new Error((await res.text()) || 'Order rejected');
      setMessage('Order accepted and queued for matching.');
      setPrice('');
      setSize('');
      fetchDepth();
      fetchMyOrders();
    } catch (err: any) {
      setError(err?.message ?? 'Failed to place order');
    } finally {
      setSubmitting(false);
    }
  };

  const onCancel = async (orderId: string) => {
    setCancelling(orderId);
    try {
      await apiFetch(`/api/orders/${orderId}`, { method: 'DELETE' });
      fetchMyOrders();
      fetchDepth();
    } catch {
      // ignore
    } finally {
      setCancelling(null);
    }
  };

  const title =
    market && market.base_asset && market.quote_asset
      ? `${market.base_asset}/${market.quote_asset}`
      : marketId ?? '…';

  const maxDepthSize = Math.max(
    ...(depth?.bids ?? []).map((l) => parseFloat(l.total_size) || 0),
    ...(depth?.asks ?? []).map((l) => parseFloat(l.total_size) || 0),
    1,
  );

  const openOrders = myOrders.filter((o) => o.status === 'pending' || o.status === 'partial');

  return (
    <div style={{ display: 'grid', gap: '1.25rem' }}>
      {/* Row 1: Market info header */}
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '1rem' }}>
        <h1 style={{ margin: 0, fontSize: '1.4rem' }}>{title}</h1>
        {market && (
          <span style={{ color: '#64748b', fontSize: '0.85rem' }}>
            tick {market.tick_size} · fee {market.fee_rate} · {market.enabled ? 'live' : 'disabled'}
          </span>
        )}
      </div>

      {loading && <p style={{ color: '#94a3b8' }}>Loading market…</p>}
      {!loading && !market && <p style={{ color: '#f97373' }}>Market not found.</p>}

      {market && (
        <>
          {/* Row 2: Depth + Order form */}
          <div style={{ display: 'grid', gap: '1.25rem', gridTemplateColumns: '1fr 1fr minmax(280px, 1fr)' }}>
            {/* Bids (buy side) */}
            <section style={{ ...cardStyle }}>
              <h3 style={sectionTitle}>
                Bids <span style={{ color: '#22c55e', fontWeight: 400, fontSize: '0.8rem' }}>buy orders</span>
              </h3>
              <DepthTable levels={depth?.bids ?? []} side="buy" maxSize={maxDepthSize} />
            </section>

            {/* Asks (sell side) */}
            <section style={{ ...cardStyle }}>
              <h3 style={sectionTitle}>
                Asks <span style={{ color: '#f97373', fontWeight: 400, fontSize: '0.8rem' }}>sell orders</span>
              </h3>
              <DepthTable levels={depth?.asks ?? []} side="sell" maxSize={maxDepthSize} />
            </section>

            {/* Order form */}
            <section style={{ ...cardStyle }}>
              <h3 style={sectionTitle}>Place limit order</h3>
              <form onSubmit={onSubmit} style={{ display: 'grid', gap: '0.6rem' }}>
                <div style={{ display: 'flex', gap: '0.4rem' }}>
                  <button type="button" className="primary-btn" style={{
                    flex: 1, background: side === 'buy' ? '#22c55e' : '#1e293b',
                    color: side === 'buy' ? '#0f172a' : '#e2e8f0',
                    boxShadow: side === 'buy' ? '0 4px 14px rgba(34,197,94,0.35)' : 'none',
                  }} onClick={() => setSide('buy')}>Buy</button>
                  <button type="button" className="primary-btn" style={{
                    flex: 1, background: side === 'sell' ? '#f97373' : '#1e293b',
                    color: side === 'sell' ? '#0f172a' : '#e2e8f0',
                    boxShadow: side === 'sell' ? '0 4px 14px rgba(248,113,113,0.35)' : 'none',
                  }} onClick={() => setSide('sell')}>Sell</button>
                </div>
                <label style={{ display: 'grid', gap: '0.2rem' }}>
                  <span style={labelStyle}>Price</span>
                  <input type="number" value={price} onChange={(e) => setPrice(e.target.value)}
                    placeholder="e.g. 62000.10" style={inputStyle} required step="any" />
                </label>
                <label style={{ display: 'grid', gap: '0.2rem' }}>
                  <span style={labelStyle}>Size</span>
                  <input type="number" value={size} onChange={(e) => setSize(e.target.value)}
                    placeholder="e.g. 0.01" style={inputStyle} required min="0" step="any" />
                </label>
                <button type="submit" className="primary-btn" disabled={submitting}
                  style={{ opacity: submitting ? 0.7 : 1 }}>
                  {submitting ? 'Placing…' : 'Place order'}
                </button>
                {error && <p style={{ color: '#f97373', fontSize: '0.85rem', margin: 0 }}>{error}</p>}
                {message && <p style={{ color: '#4ade80', fontSize: '0.85rem', margin: 0 }}>{message}</p>}
              </form>
            </section>
          </div>

          {/* Row 3: My open orders */}
          <section style={{ ...cardStyle }}>
            <h3 style={sectionTitle}>
              My open orders
              <span style={{ color: '#64748b', fontWeight: 400, fontSize: '0.8rem', marginLeft: '0.5rem' }}>
                {openOrders.length} active
              </span>
            </h3>
            {openOrders.length === 0 ? (
              <p style={{ color: '#475569', fontSize: '0.9rem' }}>No open orders for this market.</p>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid #1e293b' }}>
                      <th style={thStyle}>Side</th>
                      <th style={{ ...thStyle, textAlign: 'right' }}>Price</th>
                      <th style={{ ...thStyle, textAlign: 'right' }}>Size</th>
                      <th style={{ ...thStyle, textAlign: 'right' }}>Remaining</th>
                      <th style={thStyle}>Status</th>
                      <th style={{ ...thStyle, textAlign: 'right' }}>Placed</th>
                      <th style={{ ...thStyle, textAlign: 'center' }}></th>
                    </tr>
                  </thead>
                  <tbody>
                    {openOrders.map((o) => (
                      <tr key={o.id} style={{ borderBottom: '1px solid #0f172a' }}>
                        <td style={{ ...tdStyle, color: o.side === 'buy' ? '#22c55e' : '#f97373', fontWeight: 600 }}>
                          {o.side.toUpperCase()}
                        </td>
                        <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>{fmt(o.price)}</td>
                        <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>{fmt(o.size)}</td>
                        <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>{fmt(o.remaining)}</td>
                        <td style={tdStyle}>
                          <span style={{
                            fontSize: '0.75rem', padding: '2px 8px', borderRadius: 4,
                            background: o.status === 'pending' ? '#1e3a5f' : '#1e293b',
                            color: o.status === 'pending' ? '#38bdf8' : '#94a3b8',
                          }}>{o.status}</span>
                        </td>
                        <td style={{ ...tdStyle, textAlign: 'right', color: '#475569', fontSize: '0.8rem' }}>
                          {new Date(o.created_at).toLocaleTimeString()}
                        </td>
                        <td style={{ ...tdStyle, textAlign: 'center' }}>
                          <button onClick={() => onCancel(o.id)} disabled={cancelling === o.id} style={{
                            background: 'transparent', border: '1px solid #475569', borderRadius: 6,
                            color: '#f97373', padding: '3px 10px', fontSize: '0.8rem', cursor: 'pointer',
                            opacity: cancelling === o.id ? 0.5 : 1,
                          }}>
                            {cancelling === o.id ? '…' : 'Cancel'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Recent filled/cancelled orders */}
            {myOrders.filter((o) => o.status !== 'pending' && o.status !== 'partial').length > 0 && (
              <>
                <h4 style={{ fontSize: '0.9rem', color: '#94a3b8', marginTop: '1rem', marginBottom: '0.5rem' }}>
                  Recent history
                </h4>
                <div style={{ overflowX: 'auto' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ borderBottom: '1px solid #1e293b' }}>
                        <th style={thStyle}>Side</th>
                        <th style={{ ...thStyle, textAlign: 'right' }}>Price</th>
                        <th style={{ ...thStyle, textAlign: 'right' }}>Size</th>
                        <th style={thStyle}>Status</th>
                        <th style={{ ...thStyle, textAlign: 'right' }}>Time</th>
                      </tr>
                    </thead>
                    <tbody>
                      {myOrders
                        .filter((o) => o.status !== 'pending' && o.status !== 'partial')
                        .slice(0, 10)
                        .map((o) => (
                          <tr key={o.id} style={{ borderBottom: '1px solid #0f172a' }}>
                            <td style={{ ...tdStyle, color: o.side === 'buy' ? '#22c55e' : '#f97373' }}>
                              {o.side.toUpperCase()}
                            </td>
                            <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>{fmt(o.price)}</td>
                            <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>{fmt(o.size)}</td>
                            <td style={tdStyle}>
                              <span style={{
                                fontSize: '0.75rem', padding: '2px 8px', borderRadius: 4,
                                background: statusColor(o.status).bg, color: statusColor(o.status).fg,
                              }}>{o.status}</span>
                            </td>
                            <td style={{ ...tdStyle, textAlign: 'right', color: '#475569', fontSize: '0.8rem' }}>
                              {new Date(o.created_at).toLocaleTimeString()}
                            </td>
                          </tr>
                        ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function DepthTable({ levels, side, maxSize }: { levels: PriceLevel[]; side: 'buy' | 'sell'; maxSize: number }) {
  const color = side === 'buy' ? '#22c55e' : '#f97373';
  const barColor = side === 'buy' ? 'rgba(34,197,94,0.12)' : 'rgba(248,113,113,0.12)';

  if (levels.length === 0) {
    return <p style={{ color: '#475569', fontSize: '0.85rem' }}>No {side} orders</p>;
  }

  return (
    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
      <thead>
        <tr>
          <th style={{ ...thStyle, textAlign: 'right' }}>Price</th>
          <th style={{ ...thStyle, textAlign: 'right' }}>Size</th>
          <th style={{ ...thStyle, textAlign: 'right' }}>Orders</th>
        </tr>
      </thead>
      <tbody>
        {levels.map((l) => {
          const pct = Math.min((parseFloat(l.total_size) / maxSize) * 100, 100);
          return (
            <tr key={l.price} style={{
              background: `linear-gradient(to right, ${barColor} ${pct}%, transparent ${pct}%)`,
            }}>
              <td style={{ ...tdStyle, textAlign: 'right', color, fontFamily: 'monospace', fontWeight: 600 }}>
                {fmt(l.price)}
              </td>
              <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace' }}>{fmt(l.total_size)}</td>
              <td style={{ ...tdStyle, textAlign: 'right', color: '#64748b' }}>{l.count}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

function fmt(v: string): string {
  const n = parseFloat(v);
  if (isNaN(n)) return v;
  return n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 });
}

function statusColor(s: string): { bg: string; fg: string } {
  switch (s) {
    case 'matched': return { bg: '#14532d', fg: '#4ade80' };
    case 'cancelled': return { bg: '#1e293b', fg: '#94a3b8' };
    case 'rejected': return { bg: '#450a0a', fg: '#f97373' };
    default: return { bg: '#1e293b', fg: '#94a3b8' };
  }
}

const cardStyle: React.CSSProperties = {
  border: '1px solid #334155',
  borderRadius: 12,
  padding: '1rem 1.25rem',
};

const sectionTitle: React.CSSProperties = {
  fontSize: '1rem',
  marginTop: 0,
  marginBottom: '0.6rem',
};

const labelStyle: React.CSSProperties = {
  color: '#cbd5f5',
  fontSize: '0.85rem',
};

const inputStyle: React.CSSProperties = {
  padding: '0.45rem 0.55rem',
  borderRadius: 8,
  border: '1px solid #334155',
  background: '#020617',
  color: '#e2e8f0',
};

const thStyle: React.CSSProperties = {
  padding: '0.5rem 0.6rem',
  textAlign: 'left',
  fontSize: '0.75rem',
  color: '#94a3b8',
  fontWeight: 500,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

const tdStyle: React.CSSProperties = {
  padding: '0.4rem 0.6rem',
  fontSize: '0.9rem',
};
