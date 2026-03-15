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

const POLL_INTERVAL_MS = 30_000;  // fallback when no WebSocket depth_updated
const POLL_BACKOFF_MS = 60_000;

const COINGECKO_IDS: Record<string, string> = {
  BTC: 'bitcoin',
  ETH: 'ethereum',
  SOL: 'solana',
  ARB: 'arbitrum',
  OP: 'optimism',
};
const GLOBAL_PRICE_URL = 'https://api.coingecko.com/api/v3/simple/price';
const GLOBAL_PRICE_POLL_MS = 60_000; // poll every 60s to avoid rate limits

export function OrderBook() {
  const { marketId } = useParams<{ marketId: string }>();

  const [market, setMarket] = useState<Market | null>(null);
  const [loading, setLoading] = useState(true);
  const [depth, setDepth] = useState<Depth | null>(null);
  const [myOrders, setMyOrders] = useState<Order[]>([]);
  const [cancelling, setCancelling] = useState<string | null>(null);
  const [pollIntervalMs, setPollIntervalMs] = useState(POLL_INTERVAL_MS);
  const [globalPriceUsd, setGlobalPriceUsd] = useState<number | null>(null);

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

  useEffect(() => {
    if (!market?.base_asset) return;
    const cgId = COINGECKO_IDS[market.base_asset.toUpperCase()];
    if (!cgId) {
      setGlobalPriceUsd(null);
      return;
    }
    const fetchGlobal = () => {
      const params = new URLSearchParams({ ids: cgId, vs_currencies: 'usd' });
      fetch(`${GLOBAL_PRICE_URL}?${params}`)
        .then((r) => r.ok ? r.json() : null)
        .then((data: Record<string, { usd?: number }> | null) => {
          const price = data?.[cgId]?.usd;
          setGlobalPriceUsd(price ?? null);
        })
        .catch(() => setGlobalPriceUsd(null));
    };
    fetchGlobal();
    const id = setInterval(fetchGlobal, GLOBAL_PRICE_POLL_MS);
    return () => clearInterval(id);
  }, [market?.base_asset]);

  const fetchDepth = useCallback(() => {
    if (!marketId) return;
    apiFetch(`/api/markets/${encodeURIComponent(marketId)}/depth?limit=15`)
      .then((r) => {
        if (r.ok) {
          setPollIntervalMs(POLL_INTERVAL_MS);
          return r.json() as Promise<Depth>;
        }
        setPollIntervalMs(POLL_BACKOFF_MS);
        return null;
      })
      .then((d) => { if (d) setDepth(d); })
      .catch(() => setPollIntervalMs(POLL_BACKOFF_MS));
  }, [marketId]);

  const fetchMyOrders = useCallback(() => {
    if (!marketId) return;
    apiFetch(`/api/orders?market_id=${encodeURIComponent(marketId)}`)
      .then((r) => {
        if (r.ok) {
          setPollIntervalMs(POLL_INTERVAL_MS);
          return r.json() as Promise<Order[] | null>;
        }
        setPollIntervalMs(POLL_BACKOFF_MS);
        return null;
      })
      .then((data) => setMyOrders(Array.isArray(data) ? data : []))
      .catch(() => {
        setPollIntervalMs(POLL_BACKOFF_MS);
        setMyOrders([]);
      });
  }, [marketId]);

  useEffect(() => {
    fetchDepth();
    fetchMyOrders();
    const id = setInterval(() => {
      fetchDepth();
      fetchMyOrders();
    }, pollIntervalMs);
    return () => clearInterval(id);
  }, [fetchDepth, fetchMyOrders, pollIntervalMs]);

  useEffect(() => {
    return subscribe((msg) => {
      if (msg.type === 'order_filled' || msg.type === 'order_cancelled') {
        fetchDepth();
        fetchMyOrders();
      }
      if (msg.type === 'depth_updated' && msg.market_id === marketId) {
        fetchDepth();
        fetchMyOrders();
      }
    });
  }, [subscribe, fetchDepth, fetchMyOrders, marketId]);

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

  const bestBid = depth?.bids?.[0]?.price;
  const bestAsk = depth?.asks?.[0]?.price;
  const midPrice =
    bestBid != null && bestAsk != null
      ? (parseFloat(bestBid) + parseFloat(bestAsk)) / 2
      : null;

  const maxDepthSize = Math.max(
    ...(depth?.bids ?? []).map((l) => parseFloat(l.total_size) || 0),
    ...(depth?.asks ?? []).map((l) => parseFloat(l.total_size) || 0),
    1,
  );

  const openOrders = (myOrders ?? []).filter((o) => o.status === 'pending' || o.status === 'partial');

  return (
    <div style={{ display: 'grid', gap: '1.25rem' }}>
      {/* Row 1: Market info header + prices */}
      <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'baseline', gap: '1rem' }}>
        <h1 style={{ margin: 0, fontSize: '1.4rem', color: 'var(--text-primary)' }}>{title}</h1>
        {market && (
          <span style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>
            tick {market.tick_size} · fee {market.fee_rate} · {market.enabled ? 'live' : 'disabled'}
          </span>
        )}
        {midPrice != null && !Number.isNaN(midPrice) && (
          <span style={{ fontSize: '0.9rem', color: 'var(--text-muted)' }}>
            Mid <strong style={{ color: 'var(--text-primary)', fontFamily: 'monospace' }}>{midPrice.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</strong> {market?.quote_asset ?? ''}
          </span>
        )}
        {globalPriceUsd != null && (
          <span style={{ fontSize: '0.9rem', color: 'var(--text-muted)' }}>
            Global <strong style={{ color: 'var(--accent)', fontFamily: 'monospace' }}>${globalPriceUsd.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</strong> USD
          </span>
        )}
      </div>

      {loading && <p style={{ color: 'var(--text-muted)' }}>Loading market…</p>}
      {!loading && !market && <p style={{ color: 'var(--error)' }}>Market not found.</p>}

      {market && (
        <>
          {/* Row 2: Depth + Order form */}
          <div style={{ display: 'grid', gap: '1.25rem', gridTemplateColumns: '1fr 1fr minmax(280px, 1fr)' }}>
            {/* Bids (buy side) */}
            <section style={{ ...cardStyle }}>
              <h3 style={sectionTitle}>
                Bids <span style={{ color: '#16a34a', fontWeight: 400, fontSize: '0.8rem' }}>buy orders</span>
              </h3>
              <DepthTable levels={depth?.bids ?? []} side="buy" maxSize={maxDepthSize} />
            </section>

            {/* Asks (sell side) */}
            <section style={{ ...cardStyle }}>
              <h3 style={sectionTitle}>
                Asks <span style={{ color: '#dc2626', fontWeight: 400, fontSize: '0.8rem' }}>sell orders</span>
              </h3>
              <DepthTable levels={depth?.asks ?? []} side="sell" maxSize={maxDepthSize} />
            </section>

            {/* Order form */}
            <section style={{ ...cardStyle }}>
              <h3 style={sectionTitle}>Place limit order</h3>
              <form onSubmit={onSubmit} style={{ display: 'grid', gap: '0.6rem' }}>
                <div style={{ display: 'flex', gap: '0.4rem' }}>
                  <button type="button" className="primary-btn" style={{
                    flex: 1, background: side === 'buy' ? 'var(--success)' : 'var(--border-subtle)',
                    color: side === 'buy' ? '#fff' : 'var(--text-muted)',
                    boxShadow: side === 'buy' ? '0 4px 14px rgba(22,163,74,0.35)' : 'none',
                    border: side === 'buy' ? 'none' : '1px solid var(--border)',
                  }} onClick={() => setSide('buy')}>Buy</button>
                  <button type="button" className="primary-btn" style={{
                    flex: 1, background: side === 'sell' ? 'var(--error)' : 'var(--border-subtle)',
                    color: side === 'sell' ? '#fff' : 'var(--text-muted)',
                    boxShadow: side === 'sell' ? '0 4px 14px rgba(220,38,38,0.35)' : 'none',
                    border: side === 'sell' ? 'none' : '1px solid var(--border)',
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
                {error && <p style={{ color: 'var(--error)', fontSize: '0.85rem', margin: 0 }}>{error}</p>}
                {message && <p style={{ color: 'var(--success)', fontSize: '0.85rem', margin: 0 }}>{message}</p>}
              </form>
            </section>
          </div>

          {/* Row 3: My open orders */}
          <section style={{ ...cardStyle }}>
            <h3 style={sectionTitle}>
              My open orders
              <span style={{ color: 'var(--text-muted)', fontWeight: 400, fontSize: '0.8rem', marginLeft: '0.5rem' }}>
                {openOrders.length} active
              </span>
            </h3>
            {openOrders.length === 0 ? (
              <p style={{ color: 'var(--text-muted)', fontSize: '0.9rem' }}>No open orders for this market.</p>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                  <thead>
                    <tr style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-page)' }}>
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
                      <tr key={o.id} style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                        <td style={{ ...tdStyle, color: o.side === 'buy' ? 'var(--success)' : 'var(--error)', fontWeight: 600 }}>
                          {o.side.toUpperCase()}
                        </td>
                        <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: 'var(--text-primary)' }}>{fmt(o.price)}</td>
                        <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: 'var(--text-primary)' }}>{fmt(o.size)}</td>
                        <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: 'var(--text-primary)' }}>{fmt(o.remaining)}</td>
                        <td style={tdStyle}>
                          <span style={{
                            fontSize: '0.75rem', padding: '2px 8px', borderRadius: 4,
                            background: o.status === 'pending' ? 'var(--success-bg)' : 'var(--border-subtle)',
                            color: o.status === 'pending' ? 'var(--accent)' : 'var(--text-muted)',
                          }}>{o.status}</span>
                        </td>
                        <td style={{ ...tdStyle, textAlign: 'right', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
                          {new Date(o.created_at).toLocaleTimeString()}
                        </td>
                        <td style={{ ...tdStyle, textAlign: 'center' }}>
                          <button onClick={() => onCancel(o.id)} disabled={cancelling === o.id} style={{
                            background: 'transparent', border: '1px solid #e2e8f0', borderRadius: 6,
                            color: '#dc2626', padding: '3px 10px', fontSize: '0.8rem', cursor: 'pointer',
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
            {(myOrders ?? []).filter((o) => o.status !== 'pending' && o.status !== 'partial').length > 0 && (
              <>
                <h4 style={{ fontSize: '0.9rem', color: '#64748b', marginTop: '1rem', marginBottom: '0.5rem' }}>
                  Recent history
                </h4>
                <div style={{ overflowX: 'auto' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-page)' }}>
                        <th style={thStyle}>Side</th>
                        <th style={{ ...thStyle, textAlign: 'right' }}>Price</th>
                        <th style={{ ...thStyle, textAlign: 'right' }}>Size</th>
                        <th style={thStyle}>Status</th>
                        <th style={{ ...thStyle, textAlign: 'right' }}>Time</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(myOrders ?? [])
                        .filter((o) => o.status !== 'pending' && o.status !== 'partial')
                        .slice(0, 10)
                        .map((o) => (
                          <tr key={o.id} style={{ borderBottom: '1px solid var(--border-subtle)' }}>
                            <td style={{ ...tdStyle, color: o.side === 'buy' ? 'var(--success)' : 'var(--error)' }}>
                              {o.side.toUpperCase()}
                            </td>
                            <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: 'var(--text-primary)' }}>{fmt(o.price)}</td>
                            <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: 'var(--text-primary)' }}>{fmt(o.size)}</td>
                            <td style={tdStyle}>
                              <span style={{
                                fontSize: '0.75rem', padding: '2px 8px', borderRadius: 4,
                                background: statusColor(o.status).bg, color: statusColor(o.status).fg,
                              }}>{o.status}</span>
                            </td>
                            <td style={{ ...tdStyle, textAlign: 'right', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
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
  const color = side === 'buy' ? '#16a34a' : '#dc2626';
  const barColor = side === 'buy' ? 'rgba(22,163,74,0.1)' : 'rgba(220,38,38,0.1)';

  if (levels.length === 0) {
    return <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>No {side} orders</p>;
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
              <td style={{ ...tdStyle, textAlign: 'right', fontFamily: 'monospace', color: '#334155' }}>{fmt(l.total_size)}</td>
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
    case 'matched': return { bg: 'var(--success-bg)', fg: 'var(--success)' };
    case 'cancelled': return { bg: 'var(--border-subtle)', fg: 'var(--text-muted)' };
    case 'rejected': return { bg: 'var(--error-bg)', fg: 'var(--error)' };
    default: return { bg: 'var(--border-subtle)', fg: 'var(--text-muted)' };
  }
}

const cardStyle: React.CSSProperties = {
  border: '1px solid var(--border)',
  borderRadius: 12,
  padding: '1rem 1.25rem',
  background: 'var(--bg-card)',
};

const sectionTitle: React.CSSProperties = {
  fontSize: '1rem',
  marginTop: 0,
  marginBottom: '0.6rem',
  color: 'var(--text-primary)',
};

const labelStyle: React.CSSProperties = {
  color: 'var(--text-primary)',
  fontSize: '0.85rem',
};

const inputStyle: React.CSSProperties = {
  padding: '0.45rem 0.55rem',
  borderRadius: 8,
  border: '1px solid var(--border)',
  background: 'var(--bg-input)',
  color: 'var(--text-primary)',
};

const thStyle: React.CSSProperties = {
  padding: '0.5rem 0.6rem',
  textAlign: 'left',
  fontSize: '0.75rem',
  color: 'var(--text-muted)',
  fontWeight: 500,
  textTransform: 'uppercase',
  letterSpacing: '0.04em',
};

const tdStyle: React.CSSProperties = {
  padding: '0.4rem 0.6rem',
  fontSize: '0.9rem',
};
