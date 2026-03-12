import { useEffect, useRef, useCallback, useState } from 'react';

const STORAGE_KEY = 'fluxmesh_auth';
const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 30000;

function getUserIdFromToken(): string | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const { token } = JSON.parse(raw) as { token: string };
    const payload = JSON.parse(atob(token.split('.')[1]));
    return payload.sub ?? null;
  } catch {
    return null;
  }
}

export type WSMessage = {
  type?: string;
  [key: string]: unknown;
};

type Listener = (msg: WSMessage) => void;

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const listenersRef = useRef<Set<Listener>>(new Set());
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);
  const [connected, setConnected] = useState(false);

  const connect = useCallback(() => {
    const uid = getUserIdFromToken();
    if (!uid) return;

    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${proto}//${window.location.host}/ws?user_id=${uid}`);

    ws.onopen = () => {
      attemptRef.current = 0;
      setConnected(true);
    };

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data) as WSMessage;
        listenersRef.current.forEach((fn) => fn(msg));
      } catch {
        // ignore non-JSON frames
      }
    };

    ws.onclose = () => {
      setConnected(false);
      scheduleReconnect();
    };

    ws.onerror = () => {
      ws.close();
    };

    wsRef.current = ws;
  }, []);

  const scheduleReconnect = useCallback(() => {
    const delay = Math.min(RECONNECT_BASE_MS * 2 ** attemptRef.current, RECONNECT_MAX_MS);
    attemptRef.current += 1;
    reconnectTimer.current = setTimeout(() => {
      if (getUserIdFromToken()) connect();
    }, delay);
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimer.current !== null) clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const subscribe = useCallback((fn: Listener) => {
    listenersRef.current.add(fn);
    return () => { listenersRef.current.delete(fn); };
  }, []);

  return { connected, subscribe };
}
