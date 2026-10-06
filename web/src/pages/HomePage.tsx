import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { clearToken } from '../lib/auth-storage';
import { fetchMe } from '../features/auth/api';
import type { Me } from '../features/auth/types';

export default function HomePage() {
  const navigate = useNavigate()
  const [me, setMe] = useState<Me | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchMe()
      .then(setMe)
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : 'failed to load profile'),
      )
  }, [])

  function logout() {
    clearToken()
    navigate('/login', { replace: true })
  }

  return (
    <main className="page">
      <p className="eyebrow">SailGuard</p>
      <h1>Control plane</h1>
      {error && <p style={{ color: '#b91c1c' }}>{error}</p>}
      {me ? (
        <>
          <p className="lede">
            Signed in as <strong>{me.name}</strong> ({me.email})
          </p>
          <p className="muted">Roles: {me.roles.join(', ') || '—'}</p>
          <button type="button" onClick={logout}>
            Sign out
          </button>
        </>
      ) : (
        !error && <p className="muted">Loading profile…</p>
      )}
    </main>
  )
}