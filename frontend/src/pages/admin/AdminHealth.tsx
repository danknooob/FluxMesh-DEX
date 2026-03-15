import { useEffect, useState } from 'react';
import { apiFetch } from '../../auth/api';

type ServiceHealth = {
  name: string;
  status: string;
};

type HealthResponse = {
  services: ServiceHealth[];
};

export function AdminHealth() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    setError(null);
    apiFetch('/control/admin/health')
      .then(async (r) => {
        if (!r.ok) {
          throw new Error(r.status === 502 || r.status === 503 ? 'Control plane unreachable. Start the MCP/control service (e.g. cd mcp && go run ./cmd/mcp).' : 'Failed to load health');
        }
        return (await r.json()) as HealthResponse;
      })
      .then(setHealth)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load health'))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      <h1 style={{ marginBottom: '1rem', color: 'var(--text-primary)' }}>Health</h1>
      <p style={{ color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
        Quick view of which backend services are up. The control plane probes each service’s <code style={{ background: 'var(--border-subtle)', padding: '0.1rem 0.35rem', borderRadius: 4 }}>/health</code> endpoint (when available).
      </p>
      <p style={{ color: 'var(--text-muted)', fontSize: '0.85rem', marginBottom: '1rem' }}>
        <strong>Unknown</strong> = not probed (no /health endpoint or not configured). It does not mean the service is down — only that this dashboard does not check it yet.
      </p>

      {loading && <p style={{ color: 'var(--text-muted)' }}>Loading health…</p>}
      {error && <p style={{ color: 'var(--error)' }}>{error}</p>}

      {!loading && !error && health && (
        <div style={{ display: 'grid', gap: '0.5rem', maxWidth: 520 }}>
          {health.services.map((s) => (
            <div
              key={s.name}
              style={{
                border: '1px solid var(--border)',
                borderRadius: 10,
                padding: '0.6rem 0.8rem',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                background: 'var(--bg-card)',
              }}
            >
              <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{s.name}</span>
              <span
                style={{
                  color: s.status === 'healthy' ? 'var(--success)' : s.status === 'unknown' ? 'var(--text-muted)' : 'var(--error)',
                  fontWeight: 500,
                }}
                title={s.status === 'unknown' ? 'Not probed by control plane' : undefined}
              >
                {s.status}
                {s.status === 'unknown' && (
                  <span style={{ fontSize: '0.75rem', fontWeight: 400, marginLeft: '0.35rem', opacity: 0.9 }}>(not probed)</span>
                )}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
