import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { formatDateTime, formatRelativeTime } from '../../lib/format'
import { listEvents } from './api'
import type { DeviceEvent } from './types'

const PAGE_SIZE = 15

function eventTypeLabel(t: string): string {
  if (t === 'violation_detected') return 'Detected'
  if (t === 'violation_blocked') return 'Blocked'
  return t
}

export default function EventsPage() {
  const [events, setEvents] = useState<DeviceEvent[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listEvents()
      .then((data) => setEvents(data.items))
      .catch((err: unknown) => {
        setEvents(null)
        setError(err instanceof Error ? err.message : 'failed to load events')
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const total = events?.length ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    if (!events) return []
    const start = (page - 1) * PAGE_SIZE
    return events.slice(start, start + PAGE_SIZE)
  }, [events, page])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Telemetry</p>
          <h1 className="title">Events</h1>
          <p className="lede">
            Policy violations reported by agents (audit and enforce).
            {events ? (
              <span className="page-meta">
                {' '}
                · {events.length} event{events.length === 1 ? '' : 's'}
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

      <section className="list-panel" aria-labelledby="event-list-heading">
        <div className="list-panel-header">
          <h2 id="event-list-heading" className="panel-title">
            Recent violations
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !events ? <p className="loading">Loading events…</p> : null}

        {!loading && events && events.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No events yet</p>
            <p className="muted">
              Publish an audit policy, then run an agent that matches a block
              rule. Violations will appear here.
            </p>
          </div>
        ) : null}

        {events && events.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>When</th>
                    <th>Device</th>
                    <th>Type</th>
                    <th>Process</th>
                    <th>Match</th>
                    <th>Mode</th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((e) => (
                    <tr key={e.id}>
                      <td
                        className="muted"
                        title={formatDateTime(e.occurred_at)}
                      >
                        {formatRelativeTime(e.occurred_at)}
                      </td>
                      <td>
                        {e.device_id ? (
                          <Link
                            to={`/devices/${e.device_id}`}
                            className="cell-primary"
                          >
                            {e.hostname || e.device_id.slice(0, 8)}
                          </Link>
                        ) : (
                          <span className="muted">—</span>
                        )}
                      </td>
                      <td>
                        <span className="mono">
                          {eventTypeLabel(e.event_type)}
                        </span>
                      </td>
                      <td>
                        <div className="cell-primary mono">
                          {e.process_name || '—'}
                        </div>
                        {e.process_path ? (
                          <div className="muted mono">{e.process_path}</div>
                        ) : null}
                      </td>
                      <td>
                        <div className="mono">
                          {e.match_action || '—'}
                          {e.match_rule_type ? (
                            <span className="muted">
                              {' '}
                              · {e.match_rule_type}
                            </span>
                          ) : null}
                        </div>
                        <div className="muted mono">
                          {e.match_value || '—'}
                        </div>
                      </td>
                      <td className="mono muted">{e.policy_mode || '—'}</td>
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
