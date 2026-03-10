import { FormEvent, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

type Market = {
  id: string;
  base_asset: string;
  quote_asset: string;
  tick_size: string;
  min_size?: string;
  fee_rate: string;
  enabled: boolean;
};

export function OrderBook() {
  const { marketId } = useParams<{ marketId: string }>();

  const [market, setMarket] = useState<Market | null>(null);
  const [loading, setLoading] = useState(true);
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState('');
  const [size, setSize] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!marketId) return;
    setLoading(true);
    fetch(`/api/markets/${encodeURIComponent(marketId)}`)
      .then(async (r) => {
        if (!r.ok) {
          throw new Error('Market not found');
        }
        return (await r.json()) as Market;
      })
      .then(setMarket)
      .catch(() => setMarket(null))
      .finally(() => setLoading(false));
  }, [marketId]);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!marketId) return;
    setSubmitting(true);
    setError(null);
    setMessage(null);
    try {
      const res = await fetch('/api/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          market_id: marketId,
          side,
          price,
          size,
        }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Order rejected');
      }
      setMessage('Order accepted and queued to the matching engine.');
      setPrice('');
      setSize('');
    } catch (err: any) {
      setError(err?.message ?? 'Failed to place order');
    } finally {
      setSubmitting(false);
    }
  };

  const title =
    market && market.base_asset && market.quote_asset
      ? `${market.base_asset}/${market.quote_asset}`
      : marketId ?? '…';

  return (
    <div style={{ display: 'grid', gap: '1.5rem', gridTemplateColumns: 'minmax(0, 2fr) minmax(0, 3fr)' }}>
      <section style={{ border: '1px solid #334155', borderRadius: 12, padding: '1rem 1.25rem' }}>
        <h1 style={{ margin: 0, marginBottom: '0.5rem', fontSize: '1.4rem' }}>{title}</h1>
        {loading && <p style={{ color: '#94a3b8' }}>Loading market…</p>}
        {!loading && !market && (
          <p style={{ color: '#f97373' }}>Market not found. Go back to the markets list.</p>
        )}
        {market && (
          <>
            <p style={{ color: '#94a3b8', marginBottom: '0.75rem' }}>
              Limit order placement for this market. Matching, settlement, and balances are driven by Kafka events and
              the EVM backend.
            </p>
            <dl style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', rowGap: '0.25rem', columnGap: '0.75rem' }}>
              <dt style={{ color: '#64748b' }}>Tick size</dt>
              <dd style={{ margin: 0 }}>{market.tick_size}</dd>
              {market.min_size && (
                <>
                  <dt style={{ color: '#64748b' }}>Min size</dt>
                  <dd style={{ margin: 0 }}>{market.min_size}</dd>
                </>
              )}
              <dt style={{ color: '#64748b' }}>Fee rate</dt>
              <dd style={{ margin: 0 }}>{market.fee_rate}</dd>
              <dt style={{ color: '#64748b' }}>Status</dt>
              <dd style={{ margin: 0 }}>{market.enabled ? 'Trading enabled' : 'Disabled'}</dd>
            </dl>
          </>
        )}
      </section>

      <section style={{ border: '1px solid #334155', borderRadius: 12, padding: '1rem 1.25rem' }}>
        <h2 style={{ fontSize: '1.1rem', marginTop: 0, marginBottom: '0.75rem' }}>Place limit order</h2>
        <form onSubmit={onSubmit} style={{ display: 'grid', gap: '0.75rem', maxWidth: 420 }}>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <button
              type="button"
              className="primary-btn"
              style={{
                flex: 1,
                background: side === 'buy' ? '#22c55e' : '#1e293b',
                color: side === 'buy' ? '#0f172a' : '#e2e8f0',
                boxShadow: side === 'buy' ? '0 6px 18px rgba(34, 197, 94, 0.4)' : 'none',
              }}
              onClick={() => setSide('buy')}
            >
              Buy
            </button>
            <button
              type="button"
              className="primary-btn"
              style={{
                flex: 1,
                background: side === 'sell' ? '#f97373' : '#1e293b',
                color: side === 'sell' ? '#0f172a' : '#e2e8f0',
                boxShadow: side === 'sell' ? '0 6px 18px rgba(248, 113, 113, 0.4)' : 'none',
              }}
              onClick={() => setSide('sell')}
            >
              Sell
            </button>
          </div>

          <label style={{ display: 'grid', gap: '0.25rem' }}>
            <span style={{ color: '#cbd5f5', fontSize: '0.9rem' }}>Price</span>
            <input
              type="number"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              placeholder="e.g. 62000.10"
              style={{
                padding: '0.5rem 0.6rem',
                borderRadius: 8,
                border: '1px solid #334155',
                background: '#020617',
                color: '#e2e8f0',
              }}
              required
              step="any"
            />
          </label>

          <label style={{ display: 'grid', gap: '0.25rem' }}>
            <span style={{ color: '#cbd5f5', fontSize: '0.9rem' }}>Size</span>
            <input
              type="number"
              value={size}
              onChange={(e) => setSize(e.target.value)}
              placeholder="e.g. 0.01"
              style={{
                padding: '0.5rem 0.6rem',
                borderRadius: 8,
                border: '1px solid #334155',
                background: '#020617',
                color: '#e2e8f0',
              }}
              required
              min="0"
              step="any"
            />
          </label>

          <button
            type="submit"
            className="primary-btn"
            disabled={submitting || !market}
            style={{ opacity: submitting || !market ? 0.7 : 1 }}
          >
            {submitting ? 'Placing order…' : 'Place order'}
          </button>

          {error && <p style={{ color: '#f97373', fontSize: '0.9rem' }}>{error}</p>}
          {message && <p style={{ color: '#4ade80', fontSize: '0.9rem' }}>{message}</p>}
          <p style={{ color: '#64748b', fontSize: '0.8rem', marginTop: '0.25rem' }}>
            This is a limit order: matching and risk checks happen asynchronously via the matching engine and settlement
            services on Kafka.
          </p>
        </form>
      </section>
    </div>
  );
}
