import { useEffect, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import type { AppOutletContext } from '../layouts/AppLayout'
import { listDevices } from '../features/devices/api'

export default function HomePage() {
  const { me, meLoading, meError } = useOutletContext<AppOutletContext>()
  const [deviceCount, setDeviceCount] = useState<number | null>(null)
  const [devicesError, setDevicesError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    listDevices()
      .then((data) => {
        if (!cancelled) {
          setDeviceCount(data.items.length)
          setDevicesError(null)
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setDeviceCount(null)
          setDevicesError(
            err instanceof Error ? err.message : 'failed to load devices',
          )
        }
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="stack">
      <header>
        <p className="section-label">Overview</p>
        <h1 className="title">Control plane</h1>
        <p className="lede">
          Fleet snapshot and session summary for this admin account.
        </p>
      </header>

      {meError ? (
        <p className="error" role="alert">
          {meError}
        </p>
      ) : null}

      {devicesError ? (
        <p className="error" role="alert">
          {devicesError}
        </p>
      ) : null}

      <div className="stat-grid" aria-label="Fleet summary">
        <Link to="/devices" className="stat-card">
          <span className="stat-label">Devices</span>
          <span className="stat-value">
            {deviceCount === null ? '—' : deviceCount}
          </span>
          <span className="stat-hint">View inventory →</span>
        </Link>
        <div className="stat-card stat-card-static">
          <span className="stat-label">Roles</span>
          <span className="stat-value">
            {meLoading || !me ? '—' : me.roles.length}
          </span>
          <span className="stat-hint">Assigned to you</span>
        </div>
        <div className="stat-card stat-card-static">
          <span className="stat-label">Permissions</span>
          <span className="stat-value">
            {meLoading || !me ? '—' : me.permissions.length}
          </span>
          <span className="stat-hint">Granted codes</span>
        </div>
      </div>

      {meLoading && !me ? <p className="loading">Loading profile…</p> : null}

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
  )
}
