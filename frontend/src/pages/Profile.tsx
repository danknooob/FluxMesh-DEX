import { FormEvent, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiFetch } from '../auth/api';
import { useAuth } from '../auth/AuthContext';

interface UserProfile {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
  role: string;
  created_at: string;
}

export function Profile() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [avatarUrl, setAvatarUrl] = useState('');

  const [showDelete, setShowDelete] = useState(false);

  useEffect(() => {
    apiFetch('/api/profile')
      .then(async (r) => {
        if (!r.ok) throw new Error('Failed to load profile');
        return r.json() as Promise<UserProfile>;
      })
      .then((p) => {
        setProfile(p);
        setName(p.name || '');
        setEmail(p.email);
        setAvatarUrl(p.avatar_url || '');
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const onSave = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setMessage(null);
    try {
      const res = await apiFetch('/api/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email, avatar_url: avatarUrl }),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || 'Update failed');
      }
      const updated = (await res.json()) as UserProfile;
      setProfile(updated);
      setName(updated.name || '');
      setEmail(updated.email);
      setAvatarUrl(updated.avatar_url || '');
      setMessage('Profile updated successfully');
    } catch (err: any) {
      setError(err?.message ?? 'Update failed');
    } finally {
      setSaving(false);
    }
  };

  const onDelete = async () => {
    try {
      const res = await apiFetch('/api/profile', { method: 'DELETE' });
      if (!res.ok) throw new Error('Delete failed');
      logout();
      navigate('/login');
    } catch (err: any) {
      setError(err?.message ?? 'Delete failed');
    }
  };

  const handleAvatarFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 500_000) {
      setError('Image must be under 500KB');
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      setAvatarUrl(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  if (loading) return <p style={{ color: 'var(--text-muted)' }}>Loading profile...</p>;

  return (
    <div style={{ maxWidth: 520, margin: '0 auto' }}>
      <h1 style={{ fontSize: '1.5rem', marginBottom: '1.5rem', color: 'var(--text-primary)' }}>Profile</h1>

      <div style={{
        display: 'flex', alignItems: 'center', gap: '1.25rem',
        marginBottom: '1.5rem', padding: '1rem',
        border: '1px solid var(--border)', borderRadius: 12, background: 'var(--bg-card)',
      }}>
        <div style={{
          width: 72, height: 72, borderRadius: '50%', overflow: 'hidden',
          background: 'var(--border-subtle)', display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontSize: '1.8rem', color: 'var(--text-muted)', flexShrink: 0,
        }}>
          {avatarUrl ? (
            <img src={avatarUrl} alt="avatar" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
          ) : (
            (name || email || '?').charAt(0).toUpperCase()
          )}
        </div>
        <div>
          <div style={{ fontWeight: 600, fontSize: '1.1rem', color: 'var(--text-primary)' }}>{name || 'No name set'}</div>
          <div style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}>{profile?.email}</div>
          <div style={{ marginTop: '0.25rem' }}>
            <span style={{
              padding: '0.15rem 0.45rem', borderRadius: 6, fontSize: '0.7rem', fontWeight: 600,
              textTransform: 'uppercase',
              background: profile?.role === 'admin' ? '#7c3aed' : '#2563eb', color: '#fff',
            }}>{profile?.role}</span>
          </div>
        </div>
      </div>

      <form onSubmit={onSave} style={{ display: 'grid', gap: '1rem' }}>
        <label style={{ display: 'grid', gap: '0.25rem' }}>
          <span style={{ color: 'var(--text-primary)', fontSize: '0.85rem' }}>Display name</span>
          <input
            type="text" value={name} onChange={(e) => setName(e.target.value)}
            placeholder="Your name"
            style={inputStyle}
          />
        </label>

        <label style={{ display: 'grid', gap: '0.25rem' }}>
          <span style={{ color: '#334155', fontSize: '0.85rem' }}>Email</span>
          <input
            type="email" value={email} onChange={(e) => setEmail(e.target.value)}
            required
            style={inputStyle}
          />
        </label>

        <label style={{ display: 'grid', gap: '0.25rem' }}>
          <span style={{ color: 'var(--text-primary)', fontSize: '0.85rem' }}>Profile photo</span>
          <input
            type="file" accept="image/*" onChange={handleAvatarFile}
            style={{ color: 'var(--text-muted)', fontSize: '0.85rem' }}
          />
          <span style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>Max 500KB. Stored as base64 data URL.</span>
        </label>

        {avatarUrl && (
          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
            <img src={avatarUrl} alt="preview" style={{ width: 40, height: 40, borderRadius: '50%', objectFit: 'cover' }} />
            <button type="button" onClick={() => setAvatarUrl('')}
              style={{ background: 'none', border: 'none', color: 'var(--error)', cursor: 'pointer', fontSize: '0.8rem' }}>
              Remove photo
            </button>
          </div>
        )}

        {error && <p style={{ color: 'var(--error)', fontSize: '0.85rem', margin: 0 }}>{error}</p>}
        {message && <p style={{ color: 'var(--success)', fontSize: '0.85rem', margin: 0 }}>{message}</p>}

        <button type="submit" className="primary-btn" disabled={saving}
          style={{ opacity: saving ? 0.7 : 1 }}>
          {saving ? 'Saving...' : 'Save changes'}
        </button>
      </form>

      <div style={{
        marginTop: '2.5rem', padding: '1rem',
        border: '1px solid #fecaca', borderRadius: 12, background: '#fef2f2',
      }}>
        <h3 style={{ fontSize: '1rem', color: '#dc2626', marginTop: 0, marginBottom: '0.5rem' }}>Danger zone</h3>
        <p style={{ color: '#64748b', fontSize: '0.85rem', marginBottom: '0.75rem' }}>
          Permanently delete your account and all associated data. This action cannot be undone.
        </p>
        {!showDelete ? (
          <button onClick={() => setShowDelete(true)}
            style={{
              background: 'transparent', border: '1px solid var(--error)', color: 'var(--error)',
              padding: '0.45rem 1rem', borderRadius: 8, cursor: 'pointer', fontSize: '0.85rem',
            }}>
            Delete my account
          </button>
        ) : (
          <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
            <span style={{ color: 'var(--error)', fontSize: '0.85rem' }}>Are you sure?</span>
            <button onClick={onDelete}
              style={{
                background: 'var(--error)', border: 'none', color: '#fff',
                padding: '0.45rem 1rem', borderRadius: 8, cursor: 'pointer', fontSize: '0.85rem',
              }}>
              Yes, delete
            </button>
            <button onClick={() => setShowDelete(false)}
              style={{
                background: 'transparent', border: '1px solid var(--border)', color: 'var(--text-muted)',
                padding: '0.45rem 1rem', borderRadius: 8, cursor: 'pointer', fontSize: '0.85rem',
              }}>
              Cancel
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  padding: '0.55rem 0.7rem',
  borderRadius: 8,
  border: '1px solid var(--border)',
  background: 'var(--bg-input)',
  color: 'var(--text-primary)',
  fontSize: '0.95rem',
};
