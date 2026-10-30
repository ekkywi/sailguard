import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { TableSkeleton } from '../../components/TableSkeleton'
import type { AppOutletContext } from '../../layouts/AppLayout'
import {
  devicePresence,
  formatRelativeTime,
  presenceBadgeClass,
  presenceLabel,
  statusBadgeClass,
} from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { listDevices } from './api'
import type { Device, DeviceListQuery, DeviceOSFilter, DeviceStatusFilter } from './types'

const PAGE_SIZE = 25

const STATUS_CHIPS: { id: DeviceStatusFilter; label: string }[] = [
  { id: '', label: 'All' },
  { id: 'active', label: 'Active' },
  { id: 'disabled', label: 'Disabled' },
]

const OS_CHIPS: { id: DeviceOSFilter; label: string }[] = [
  { id: '', label: 'All OS' },
  { id: 'windows', label: 'Windows' },
  { id: 'linux', label: 'Linux' },
]

export default function DevicesPage() {
  useDocumentTitle('Devices')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'device.read')
  const navigate = useNavigate()

  const [devices, setDevices] = useState<Device[] | null>(null)
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  const [status, setStatus] = useState<DeviceStatusFilter>('')
  const [os, setOS] = useState<DeviceOSFilter>('')
  const [q, setQ] = useState('')
  const [qDraft, setQDraft] = useState('')

  useEffect(() => {
    const t = window.setTimeout(() => {
      setQ(qDraft.trim())
      setPage(1)
    }, 300)
    return () => window.clearTimeout(t)
  }, [qDraft])

  const query: DeviceListQuery = useMemo(
    () => ({
      status: status || undefined,
      os: os || undefined,
      q: q || undefined,
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
    }),
    [status, os, q, page],
  )

  const hasFilters = Boolean(status || os || q)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listDevices(query)
      .then((data) => {
        setDevices(data.items)
        setTotal(data.total)
      })
      .catch((err: unknown) => {
        setDevices(null)
        setTotal(0)
        setError(err instanceof Error ? err.message : 'failed to load devices')
      })
      .finally(() => setLoading(false))
  }, [canRead, query])

  useEffect(() => {
    load()
  }, [load])

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  function clearFilters() {
    setStatus('')
    setOS('')
    setQDraft('')
    setQ('')
    setPage(1)
  }

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
            <span className="page-meta">
              {' '}
              · {total.toLocaleString()} matching
            </span>
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

      <div className="audit-toolbar" role="search">
        <div className="audit-chips" aria-label="Activation status">
          {STATUS_CHIPS.map((c) => (
            <button
              key={c.id || 'all-status'}
              type="button"
              className={
                status === c.id ? 'audit-chip audit-chip-on' : 'audit-chip'
              }
              onClick={() => {
                setStatus(c.id)
                setPage(1)
              }}
            >
              {c.label}
            </button>
          ))}
        </div>
        <div className="audit-chips" aria-label="Operating system">
          {OS_CHIPS.map((c) => (
            <button
              key={c.id || 'all-os'}
              type="button"
              className={os === c.id ? 'audit-chip audit-chip-on' : 'audit-chip'}
              onClick={() => {
                setOS(c.id)
                setPage(1)
              }}
            >
              {c.label}
            </button>
          ))}
        </div>
        <div className="audit-filters">
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="devices-q">
              Search
            </label>
            <input
              id="devices-q"
              className="input"
              value={qDraft}
              onChange={(e) => setQDraft(e.target.value)}
              placeholder="Hostname, PIC, location, asset tag…"
              autoComplete="off"
            />
          </div>
          {hasFilters ? (
            <div className="field audit-filter-actions">
              <span className="label" aria-hidden="true">
                &nbsp;
              </span>
              <button
                type="button"
                className="btn btn-ghost"
                onClick={clearFilters}
              >
                Clear
              </button>
            </div>
          ) : null}
        </div>
      </div>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      <section className="list-panel" aria-labelledby="device-list-heading">
        <div className="list-panel-header">
          <h2 id="device-list-heading" className="panel-title">
            Fleet
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !devices ? (
          <TableSkeleton label="Loading devices…" rows={4} />
        ) : null}

        {!loading && devices && devices.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">
              {hasFilters ? 'No matching devices' : 'No devices enrolled yet'}
            </p>
            <p className="muted">
              {hasFilters
                ? 'Try another search or clear filters.'
                : 'Create a token, download an agent package, then enroll a host.'}
            </p>
            <div className="empty-actions">
              {hasFilters ? (
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={clearFilters}
                >
                  Clear filters
                </button>
              ) : (
                <>
                  <Link className="btn btn-ghost" to="/tokens">
                    Enrollment tokens
                  </Link>
                  <Link className="btn btn-ghost" to="/agent-download">
                    Download agent
                  </Link>
                </>
              )}
            </div>
          </div>
        ) : null}

        {devices && devices.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Hostname</th>
                    <th>OS</th>
                    <th>PIC</th>
                    <th>Location</th>
                    <th>IP</th>
                    <th>Status</th>
                    <th>Connectivity</th>
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
                        {d.display_name ? (
                          <div className="muted">{d.display_name}</div>
                        ) : d.machine_guid ? (
                          <div className="muted mono">{d.machine_guid}</div>
                        ) : null}
                      </td>
                      <td>
                        {d.os}
                        {d.os_version ? (
                          <span className="muted"> {d.os_version}</span>
                        ) : null}
                      </td>
                      <td className="muted">{d.pic || '—'}</td>
                      <td className="muted">{d.location || '—'}</td>
                      <td className="muted mono">{d.primary_ip || '—'}</td>
                      <td>
                        <span className={statusBadgeClass(d.status)}>
                          {d.status}
                        </span>
                      </td>
                      <td>
                        {(() => {
                          const p = devicePresence(d.last_seen_at)
                          return (
                            <span
                              className={presenceBadgeClass(p)}
                              title={
                                p === 'online'
                                  ? 'Heartbeat within online window'
                                  : p === 'stale'
                                    ? 'No recent heartbeat'
                                    : 'Never seen'
                              }
                            >
                              {presenceLabel(p)}
                            </span>
                          )
                        })()}
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

            <div className="pagination">
              <p className="muted pagination-meta">
                Page {page} of {pageCount}
              </p>
              <div className="pagination-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={page <= 1 || loading}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={page >= pageCount || loading}
                  onClick={() => setPage((p) => Math.min(pageCount, p + 1))}
                >
                  Next
                </button>
              </div>
            </div>
          </>
        ) : null}
      </section>
    </div>
  )
}
