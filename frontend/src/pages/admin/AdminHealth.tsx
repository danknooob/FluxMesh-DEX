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
          throw new Error('Failed to load health');
        }
        return (await r.json()) as HealthResponse;
      })
      .then(setHealth)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      <h1 style={{ marginBottom: '1rem' }}>Health</h1>
      <p style={{ color: '#94a3b8', marginBottom: '1rem' }}>
        Service registry and health as reported by the control plane via <code>control.health</code> events.
      </p>

      {loading && <p>Loading health…</p>}
      {error && <p style={{ color: '#f97373' }}>{error}</p>}

      {!loading && !error && health && (
        <div style={{ display: 'grid', gap: '0.5rem', maxWidth: 480 }}>
          {health.services.map((s) => (
            <div
              key={s.name}
              style={{
                border: '1px solid #334155',
                borderRadius: 8,
                padding: '0.6rem 0.8rem',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
            >
              <span>{s.name}</span>
              <span style={{ color: s.status === 'healthy' ? '#4ade80' : '#f97373' }}>{s.status}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
