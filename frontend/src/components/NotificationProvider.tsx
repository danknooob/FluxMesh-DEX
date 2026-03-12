import { createContext, useContext, useState, useCallback, useEffect, useRef, type ReactNode } from 'react';
import { useWebSocket, type WSMessage } from '../hooks/useWebSocket';

interface Toast {
  id: number;
  title: string;
  body: string;
  color: string;
  ts: number;
}

interface NotificationContextValue {
  subscribe: (fn: (msg: WSMessage) => void) => () => void;
  connected: boolean;
}

const NotificationContext = createContext<NotificationContextValue | null>(null);

const TOAST_TTL = 6000;

export function NotificationProvider({ children }: { children: ReactNode }) {
  const { connected, subscribe } = useWebSocket();
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextId = useRef(0);

  const addToast = useCallback((title: string, body: string, color: string) => {
    const id = nextId.current++;
    setToasts((prev) => [...prev, { id, title, body, color, ts: Date.now() }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, TOAST_TTL);
  }, []);

  useEffect(() => {
    return subscribe((msg) => {
      switch (msg.type) {
        case 'order_filled': {
          const role = msg.role === 'maker' ? 'Maker' : 'Taker';
          addToast(
            `Order Filled (${role})`,
            `${msg.size} @ ${msg.price} on ${msg.market_id}`,
            '#22c55e',
          );
          break;
        }
        case 'order_cancelled':
          addToast(
            'Order Cancelled',
            `Order on ${msg.market_id ?? 'unknown market'}` +
              (msg.cancel_fee && msg.cancel_fee !== '0' ? ` · Fee: ${msg.cancel_fee}` : ''),
            '#fbbf24',
          );
          break;
        case 'balance_updated':
          addToast(
            'Balance Updated',
            `${msg.asset ?? 'Asset'} balance changed`,
            '#38bdf8',
          );
          break;
        default:
          break;
      }
    });
  }, [subscribe, addToast]);

  return (
    <NotificationContext.Provider value={{ subscribe, connected }}>
      {children}
      <ToastContainer toasts={toasts} onDismiss={(id) => setToasts((p) => p.filter((t) => t.id !== id))} />
    </NotificationContext.Provider>
  );
}

export function useNotifications(): NotificationContextValue {
  const ctx = useContext(NotificationContext);
  if (!ctx) throw new Error('useNotifications must be inside NotificationProvider');
  return ctx;
}

function ToastContainer({ toasts, onDismiss }: { toasts: Toast[]; onDismiss: (id: number) => void }) {
  if (toasts.length === 0) return null;

  return (
    <div style={{
      position: 'fixed',
      bottom: '1.5rem',
      right: '1.5rem',
      display: 'flex',
      flexDirection: 'column-reverse',
      gap: '0.5rem',
      zIndex: 9999,
      pointerEvents: 'none',
      maxWidth: 380,
    }}>
      {toasts.map((t) => (
        <div
          key={t.id}
          style={{
            pointerEvents: 'auto',
            background: '#1e293b',
            border: `1px solid ${t.color}44`,
            borderLeft: `4px solid ${t.color}`,
            borderRadius: 10,
            padding: '0.7rem 1rem',
            boxShadow: '0 8px 24px rgba(0,0,0,0.5)',
            animation: 'slideIn 0.25s ease-out',
            cursor: 'pointer',
          }}
          onClick={() => onDismiss(t.id)}
        >
          <div style={{ fontWeight: 600, fontSize: '0.85rem', color: t.color, marginBottom: '0.15rem' }}>
            {t.title}
          </div>
          <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>
            {t.body}
          </div>
        </div>
      ))}
    </div>
  );
}
