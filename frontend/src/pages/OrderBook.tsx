import { useParams } from 'react-router-dom';

export function OrderBook() {
  const { marketId } = useParams<{ marketId: string }>();

  return (
    <div>
      <h1>Order book — {marketId ?? '…'}</h1>
      <p style={{ color: '#94a3b8' }}>Order book and place order UI (connect to WebSocket for live updates).</p>
    </div>
  );
}
