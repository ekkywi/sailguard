import { useEffect, useState } from 'react'
import { AppShell } from '../components/AppShell'
import { fetchMe } from '../features/auth/api'
import type { Me } from '../features/auth/types'

export default function HomePage() {
  const [me, setMe] = useState<Me | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetchMe()
      .then(setMe)
      .catch((err: unknown) =>
        setError(err instanceof Error ? err.message : 'failed to load profile'),
      )
  }, [])

  const userLabel = me ? me.email : null

  return (
    <AppShell userLabel={userLabel}>
      <div className="stack">
        <header>
          <p className="section-label">Overview</p>
          <h1 className="title">Control plane</h1>
          <p className="lede">
            Signed-in session and role summary for this admin account.
          </p>
        </header>

        {error ? (
          <p className="error" role="alert">
            {error}
          </p>
        ) : null}

        {!me && !error ? <p className="loading">Loading profile…</p> : null}

        {me ? (
          <div className="info-list" aria-label="Account details">
            <div className="info-row">
              <span className="info-key">Name</span>
              <span className="info-val">{me.name}</span>
            </div>
            <div className="info-row">
              <span className="info-key">Email</span>
              <span className="info-val">{me.email}</span>
            </div>
            <div className="info-row">
              <span className="info-key">Roles</span>
              <span className="info-val">
                {me.roles.length ? (
                  <span className="badge-row">
                    {me.roles.map((role) => (
                      <span key={role} className="badge">
                        {role}
                      </span>
                    ))}
                  </span>
                ) : (
                  '—'
                )}
              </span>
            </div>
            <div className="info-row">
              <span className="info-key">Permissions</span>
              <span className="info-val muted">
                {me.permissions.length
                  ? `${me.permissions.length} granted`
                  : '—'}
              </span>
            </div>
          </div>
        ) : null}
      </div>
    </AppShell>
  )
}
