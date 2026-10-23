import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { TableSkeleton } from '../../components/TableSkeleton'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatDateTime, formatRelativeTime } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { exportEventsCSV, listEvents } from './api'
import type { DeviceEvent, EventListQuery, EventTypeFilter } from './types'

const PAGE_SIZE = 25

const TYPE_CHIPS: { id: EventTypeFilter; label: string }[] = [
  { id: '', label: 'All' },
  { id: 'violation_detected', label: 'Detected' },
  { id: 'violation_blocked', label: 'Blocked' },
]

function eventTypeLabel(t: string): string {
  if (t === 'violation_detected') return 'Detected'
  if (t === 'violation_blocked') return 'Blocked'
  return t
}

function localInputToRFC3339(local: string, endOfDay: boolean): string | undefined {
  const v = local.trim()
  if (!v) return undefined
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return undefined
  if (endOfDay && v.length === 10) {
    d.setHours(23, 59, 59, 999)
  } else if (!endOfDay && v.length === 10) {
    d.setHours(0, 0, 0, 0)
  }
  return d.toISOString()
}

export default function EventsPage() {
  useDocumentTitle('Events')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'event.read')

  const [events, setEvents] = useState<DeviceEvent[] | null>(null)
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [page, setPage] = useState(1)

  const [eventType, setEventType] = useState<EventTypeFilter>('')
  const [fromLocal, setFromLocal] = useState('')
  const [toLocal, setToLocal] = useState('')
  const [q, setQ] = useState('')
  const [qDraft, setQDraft] = useState('')

  useEffect(() => {
    const t = window.setTimeout(() => {
      setQ(qDraft.trim())
      setPage(1)
    }, 300)
    return () => window.clearTimeout(t)
  }, [qDraft])

  const query: EventListQuery = useMemo(() => {
    return {
      event_type: eventType || undefined,
      from: localInputToRFC3339(fromLocal, false),
      to: localInputToRFC3339(toLocal, true),
      q: q || undefined,
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
    }
  }, [eventType, fromLocal, toLocal, q, page])

  const filterQuery = useMemo((): EventListQuery => {
    return {
      event_type: query.event_type,
      from: query.from,
      to: query.to,
      q: query.q,
    }
  }, [query])

  const hasFilters = Boolean(eventType || fromLocal || toLocal || q)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listEvents(query)
      .then((data) => {
        setEvents(data.items)
        setTotal(data.total)
      })
      .catch((err: unknown) => {
        setEvents(null)
        setTotal(0)
        setError(err instanceof Error ? err.message : 'failed to load events')
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
    setEventType('')
    setFromLocal('')
    setToLocal('')
    setQDraft('')
    setQ('')
    setPage(1)
  }

  async function onExport() {
    setExporting(true)
    setError(null)
    try {
      await exportEventsCSV(filterQuery)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'export failed')
    } finally {
      setExporting(false)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="event.read" title="Events" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Telemetry</p>
          <h1 className="title">Events</h1>
          <p className="lede">
            Agent policy violations
            <span className="page-meta">
              {' '}
              · {total.toLocaleString()} matching
            </span>
          </p>
        </div>
        <div className="btn-row">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={onExport}
            disabled={exporting || loading || total === 0}
          >
            {exporting ? 'Exporting…' : 'Export CSV'}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={load}
            disabled={loading}
          >
            Refresh
          </button>
        </div>
      </header>

      <div className="audit-toolbar" role="search">
        <div className="audit-chips" aria-label="Event type">
          {TYPE_CHIPS.map((c) => (
            <button
              key={c.id || 'all'}
              type="button"
              className={
                eventType === c.id ? 'audit-chip audit-chip-on' : 'audit-chip'
              }
              onClick={() => {
                setEventType(c.id)
                setPage(1)
              }}
            >
              {c.label}
            </button>
          ))}
        </div>

        <div className="audit-filters">
          <div className="field">
            <label className="label" htmlFor="events-from">
              From
            </label>
            <input
              id="events-from"
              className="input"
              type="datetime-local"
              value={fromLocal}
              onChange={(e) => {
                setFromLocal(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <div className="field">
            <label className="label" htmlFor="events-to">
              To
            </label>
            <input
              id="events-to"
              className="input"
              type="datetime-local"
              value={toLocal}
              onChange={(e) => {
                setToLocal(e.target.value)
                setPage(1)
              }}
            />
          </div>
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="events-q">
              Search
            </label>
            <input
              id="events-q"
              className="input"
              value={qDraft}
              onChange={(e) => setQDraft(e.target.value)}
              placeholder="Host, process, user, match…"
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

      <section className="list-panel" aria-labelledby="event-list-heading">
        <div className="list-panel-header">
          <h2 id="event-list-heading" className="panel-title">
            Violations
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !events ? (
          <TableSkeleton label="Loading events…" />
        ) : null}

        {!loading && events && events.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">
              {hasFilters ? 'No matching events' : 'No events yet'}
            </p>
            <p className="muted">
              {hasFilters
                ? 'Widen the date range or clear filters.'
                : 'Publish an audit policy, then run an agent that matches a rule.'}
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
                <Link className="btn btn-ghost" to="/policies">
                  Policies
                </Link>
              )}
            </div>
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
                    <th>User</th>
                    <th>Match</th>
                    <th>Mode</th>
                  </tr>
                </thead>
                <tbody>
                  {events.map((e) => (
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
                          {e.process_user_name || '—'}
                        </div>
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
