import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { TableSkeleton } from '../../components/TableSkeleton'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatRelativeTime, statusBadgeClass } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { listDevices } from './api'
import type { Device } from './types'

export default function DevicesPage() {
  useDocumentTitle('Devices')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'device.read')
  const navigate = useNavigate()
  const [devices, setDevices] = useState<Device[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listDevices()
      .then((data) => setDevices(data.items))
      .catch((err: unknown) => {
        setDevices(null)
        setError(err instanceof Error ? err.message : 'failed to load devices')
      })
      .finally(() => setLoading(false))
  }, [canRead])

  useEffect(() => {
    load()
  }, [load])

  if (!canRead) {
    return <PermissionDenied permission="device.read" title="Devices" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Inventory</p>
          <h1 className="title">Devices</h1>
          <p className="lede">
            Enrolled endpoints
            {devices ? (
              <span className="page-meta">
                {' '}
                · {devices.length} device{devices.length === 1 ? '' : 's'}
              </span>
            ) : null}
          </p>
        </div>
        <button
          type="button"
          className="btn btn-ghost"
          onClick={load}
          disabled={loading}
        >
          Refresh
        </button>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading && !devices ? (
        <TableSkeleton label="Loading devices…" rows={4} />
      ) : null}

      {!loading && devices && devices.length === 0 ? (
        <div className="empty-state" role="status">
          <p className="empty-title">No devices enrolled yet</p>
          <p className="muted">
            Create a token, download an agent package, then enroll a host.
          </p>
          <div className="empty-actions">
            <Link className="btn btn-ghost" to="/tokens">
              Enrollment tokens
            </Link>
            <Link className="btn btn-ghost" to="/agent-download">
              Download agent
            </Link>
          </div>
        </div>
      ) : null}

      {devices && devices.length > 0 ? (
        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Hostname</th>
                <th>OS</th>
                <th>Status</th>
                <th>Agent</th>
                <th>Last seen</th>
                <th>Enrolled</th>
              </tr>
            </thead>
            <tbody>
              {devices.map((d) => (
                <tr
                  key={d.id}
                  className="table-row-link"
                  tabIndex={0}
                  onClick={() => navigate(`/devices/${d.id}`)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      navigate(`/devices/${d.id}`)
                    }
                  }}
                >
                  <td>
                    <div className="cell-primary">{d.hostname}</div>
                    {d.machine_guid ? (
                      <div className="muted mono">{d.machine_guid}</div>
                    ) : null}
                  </td>
                  <td>
                    {d.os}
                    {d.os_version ? (
                      <span className="muted"> {d.os_version}</span>
                    ) : null}
                  </td>
                  <td>
                    <span className={statusBadgeClass(d.status)}>{d.status}</span>
                  </td>
                  <td className="muted">{d.agent_version || '—'}</td>
                  <td className="muted" title={d.last_seen_at ?? undefined}>
                    {formatRelativeTime(d.last_seen_at)}
                  </td>
                  <td className="muted" title={d.enrolled_at ?? undefined}>
                    {formatRelativeTime(d.enrolled_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {devices && devices.length > 0 ? (
        <p className="muted table-hint">
          Select a row for details.{' '}
          <Link to="/">Back to overview</Link>
        </p>
      ) : null}
    </div>
  )
}
