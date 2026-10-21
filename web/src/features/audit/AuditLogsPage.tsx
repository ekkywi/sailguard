import { useCallback, useEffect, useMemo, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatDateTime, formatRelativeTime } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { listAuditLogs } from './api'
import type { AuditLog } from './types'

const PAGE_SIZE = 15

export default function AuditLogsPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'audit.read')
  const [logs, setLogs] = useState<AuditLog[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listAuditLogs()
      .then((data) => setLogs(data.items))
      .catch((err: unknown) => {
        setLogs(null)
        setError(
          err instanceof Error ? err.message : 'failed to load audit logs',
        )
      })
      .finally(() => setLoading(false))
  }, [canRead])

  useEffect(() => {
    load()
  }, [load])

  const total = logs?.length ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    if (!logs) return []
    const start = (page - 1) * PAGE_SIZE
    return logs.slice(start, start + PAGE_SIZE)
  }, [logs, page])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  if (!canRead) {
    return <PermissionDenied permission="audit.read" title="Audit log" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Security</p>
          <h1 className="title">Audit log</h1>
          <p className="lede">
            Administrative actions such as policy publish and device overrides.
            {logs ? (
              <span className="page-meta">
                {' '}
                · {logs.length} entr{logs.length === 1 ? 'y' : 'ies'}
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

      <section className="list-panel" aria-labelledby="audit-list-heading">
        <div className="list-panel-header">
          <h2 id="audit-list-heading" className="panel-title">
            Recent activity
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !logs ? <p className="loading">Loading audit log…</p> : null}

        {!loading && logs && logs.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No audit entries yet</p>
            <p className="muted">
              Publish a policy or create a device override to record the first
              entry.
            </p>
          </div>
        ) : null}

        {logs && logs.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>When</th>
                    <th>Actor</th>
                    <th>Action</th>
                    <th>Summary</th>
                    <th>Resource</th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((e) => (
                    <tr key={e.id}>
                      <td
                        className="muted"
                        title={formatDateTime(e.created_at)}
                      >
                        {formatRelativeTime(e.created_at)}
                      </td>
                      <td>
                        <div className="cell-primary">
                          {e.actor_email || '—'}
                        </div>
                      </td>
                      <td className="mono">{e.action}</td>
                      <td>{e.summary || '—'}</td>
                      <td>
                        <div className="mono muted">
                          {e.resource_type || '—'}
                        </div>
                        <div className="mono muted">
                          {e.resource_id
                            ? e.resource_id.length > 12
                              ? `${e.resource_id.slice(0, 8)}…`
                              : e.resource_id
                            : '—'}
                        </div>
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
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </button>
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={page >= pageCount}
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
