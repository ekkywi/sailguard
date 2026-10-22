import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatDateTime, formatRelativeTime } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { exportAuditLogsCSV, listAuditLogs } from './api'
import type { AuditFamily, AuditLog, AuditLogQuery } from './types'

const PAGE_SIZE = 25

const FAMILIES: { id: AuditFamily; label: string }[] = [
  { id: '', label: 'All' },
  { id: 'policy', label: 'Policy' },
  { id: 'category', label: 'Category' },
  { id: 'override', label: 'Override' },
  { id: 'user', label: 'User' },
  { id: 'device', label: 'Device' },
  { id: 'group', label: 'Group' },
  { id: 'enrollment', label: 'Enrollment' },
  { id: 'package', label: 'Package' },
  { id: 'auth', label: 'Auth' },
]

function actionFamilyClass(action: string): string {
  const a = action.toLowerCase()
  if (a.startsWith('auth.')) return 'audit-action audit-action--auth'
  if (a.includes('.fail') || a.endsWith('.delete') || a.includes('.revoke')) {
    return 'audit-action audit-action--warn'
  }
  if (a.startsWith('policy.')) return 'audit-action audit-action--policy'
  if (a.startsWith('category.')) return 'audit-action audit-action--category'
  if (a.startsWith('user.')) return 'audit-action audit-action--user'
  if (a.startsWith('device.') || a.startsWith('override.')) {
    return 'audit-action audit-action--device'
  }
  if (a.startsWith('group.')) return 'audit-action audit-action--group'
  if (a.startsWith('agent_package.')) return 'audit-action audit-action--package'
  if (a.startsWith('enrollment_token.')) {
    return 'audit-action audit-action--enrollment'
  }
  return 'audit-action'
}

function resourceHref(
  resourceType: string,
  resourceId: string,
  meta: Record<string, unknown> | null,
): string | null {
  if (!resourceId) return null
  switch (resourceType) {
    case 'policy':
      return `/policies/${resourceId}`
    case 'device':
      return `/devices/${resourceId}`
    case 'group':
      return `/groups/${resourceId}`
    case 'category':
      return `/categories/${encodeURIComponent(resourceId)}`
    case 'device_override': {
      const deviceId = meta?.device_id
      if (typeof deviceId === 'string' && deviceId) {
        return `/devices/${deviceId}?tab=overrides`
      }
      return null
    }
    case 'policy_rule':
    case 'policy_assignment': {
      const policyId = meta?.policy_id
      if (typeof policyId === 'string' && policyId) {
        return `/policies/${policyId}`
      }
      return null
    }
    default:
      return null
  }
}

function localInputToRFC3339(local: string, endOfDay: boolean): string | undefined {
  const v = local.trim()
  if (!v) return undefined
  // datetime-local → treat as local wall clock
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return undefined
  if (endOfDay && v.length === 10) {
    // date-only: include full day
    d.setHours(23, 59, 59, 999)
  } else if (!endOfDay && v.length === 10) {
    d.setHours(0, 0, 0, 0)
  }
  return d.toISOString()
}

function shortId(id: string): string {
  if (id.length <= 12) return id
  return `${id.slice(0, 8)}…`
}

export default function AuditLogsPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'audit.read')

  const [logs, setLogs] = useState<AuditLog[] | null>(null)
  const [total, setTotal] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [exporting, setExporting] = useState(false)
  const [page, setPage] = useState(1)

  const [family, setFamily] = useState<AuditFamily>('')
  const [fromLocal, setFromLocal] = useState('')
  const [toLocal, setToLocal] = useState('')
  const [actor, setActor] = useState('')
  const [q, setQ] = useState('')
  const [actorDraft, setActorDraft] = useState('')
  const [qDraft, setQDraft] = useState('')

  const [selected, setSelected] = useState<AuditLog | null>(null)

  // Debounce text filters
  useEffect(() => {
    const t = window.setTimeout(() => {
      setActor(actorDraft.trim())
      setQ(qDraft.trim())
      setPage(1)
    }, 300)
    return () => window.clearTimeout(t)
  }, [actorDraft, qDraft])

  const query: AuditLogQuery = useMemo(() => {
    const from = localInputToRFC3339(fromLocal, false)
    const to = localInputToRFC3339(toLocal, true)
    return {
      family: family || undefined,
      from,
      to,
      actor: actor || undefined,
      q: q || undefined,
      limit: PAGE_SIZE,
      offset: (page - 1) * PAGE_SIZE,
    }
  }, [family, fromLocal, toLocal, actor, q, page])

  const filterQuery = useMemo((): AuditLogQuery => {
    return {
      family: query.family,
      from: query.from,
      to: query.to,
      actor: query.actor,
      q: query.q,
    }
  }, [query])

  const hasFilters = Boolean(
    family || fromLocal || toLocal || actor || q,
  )

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listAuditLogs(query)
      .then((data) => {
        setLogs(data.items)
        setTotal(data.total)
      })
      .catch((err: unknown) => {
        setLogs(null)
        setTotal(0)
        setError(
          err instanceof Error ? err.message : 'failed to load audit logs',
        )
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
    setFamily('')
    setFromLocal('')
    setToLocal('')
    setActorDraft('')
    setQDraft('')
    setActor('')
    setQ('')
    setPage(1)
  }

  async function onExport() {
    setExporting(true)
    setError(null)
    try {
      await exportAuditLogsCSV(filterQuery)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'export failed')
    } finally {
      setExporting(false)
    }
  }

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
            Administrative actions across the control plane.
            {typeof total === 'number' ? (
              <span className="page-meta">
                {' '}
                · {total.toLocaleString()} matching
              </span>
            ) : null}
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
        <div className="audit-chips" aria-label="Action family">
          {FAMILIES.map((f) => (
            <button
              key={f.id || 'all'}
              type="button"
              className={
                family === f.id ? 'audit-chip audit-chip-on' : 'audit-chip'
              }
              onClick={() => {
                setFamily(f.id)
                setPage(1)
              }}
            >
              {f.label}
            </button>
          ))}
        </div>

        <div className="audit-filters">
          <div className="field">
            <label className="label" htmlFor="audit-from">
              From
            </label>
            <input
              id="audit-from"
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
            <label className="label" htmlFor="audit-to">
              To
            </label>
            <input
              id="audit-to"
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
            <label className="label" htmlFor="audit-actor">
              Actor
            </label>
            <input
              id="audit-actor"
              className="input"
              value={actorDraft}
              onChange={(e) => setActorDraft(e.target.value)}
              placeholder="Email…"
              autoComplete="off"
            />
          </div>
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="audit-q">
              Search
            </label>
            <input
              id="audit-q"
              className="input"
              value={qDraft}
              onChange={(e) => setQDraft(e.target.value)}
              placeholder="Action, summary, resource…"
              autoComplete="off"
            />
          </div>
          {hasFilters ? (
            <div className="field audit-filter-clear">
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

      <section className="list-panel" aria-labelledby="audit-list-heading">
        <div className="list-panel-header">
          <h2 id="audit-list-heading" className="panel-title">
            Activity
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total.toLocaleString()}
            </p>
          ) : null}
        </div>

        {loading && !logs ? <p className="loading">Loading audit log…</p> : null}

        {!loading && logs && logs.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">
              {hasFilters ? 'No matching entries' : 'No audit entries yet'}
            </p>
            <p className="muted">
              {hasFilters
                ? 'Try another family, date range, or clear filters.'
                : 'Publish a policy or sign in to record the first entry.'}
            </p>
          </div>
        ) : null}

        {logs && logs.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table audit-table">
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
                  {logs.map((e) => {
                    const href = resourceHref(
                      e.resource_type,
                      e.resource_id,
                      e.metadata,
                    )
                    return (
                      <tr
                        key={e.id}
                        className="audit-row"
                        tabIndex={0}
                        onClick={() => setSelected(e)}
                        onKeyDown={(ev) => {
                          if (ev.key === 'Enter' || ev.key === ' ') {
                            ev.preventDefault()
                            setSelected(e)
                          }
                        }}
                      >
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
                        <td>
                          <span
                            className={actionFamilyClass(e.action)}
                            title={e.action}
                          >
                            {e.action}
                          </span>
                        </td>
                        <td>{e.summary || '—'}</td>
                        <td onClick={(ev) => ev.stopPropagation()}>
                          <div className="mono muted">
                            {e.resource_type || '—'}
                          </div>
                          {href && e.resource_id ? (
                            <Link
                              className="mono audit-resource-link"
                              to={href}
                            >
                              {shortId(e.resource_id)}
                            </Link>
                          ) : (
                            <div className="mono muted">
                              {e.resource_id
                                ? shortId(e.resource_id)
                                : '—'}
                            </div>
                          )}
                        </td>
                      </tr>
                    )
                  })}
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

      {selected ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            onClick={() => setSelected(null)}
          />
          <div
            className="dialog-panel dialog-panel--form audit-detail-panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="audit-detail-title"
          >
            <h2 id="audit-detail-title" className="dialog-title">
              Audit entry
            </h2>
            <p className="dialog-desc muted">
              {formatDateTime(selected.created_at)}
            </p>

            <dl className="audit-detail-grid">
              <div>
                <dt>Actor</dt>
                <dd>{selected.actor_email || '—'}</dd>
              </div>
              <div>
                <dt>Action</dt>
                <dd>
                  <span className={actionFamilyClass(selected.action)}>
                    {selected.action}
                  </span>
                </dd>
              </div>
              <div>
                <dt>Summary</dt>
                <dd>{selected.summary || '—'}</dd>
              </div>
              <div>
                <dt>Resource</dt>
                <dd className="mono">
                  {selected.resource_type || '—'}
                  {selected.resource_id
                    ? ` · ${selected.resource_id}`
                    : ''}
                </dd>
              </div>
              {(() => {
                const href = resourceHref(
                  selected.resource_type,
                  selected.resource_id,
                  selected.metadata,
                )
                if (!href) return null
                return (
                  <div>
                    <dt>Open</dt>
                    <dd>
                      <Link to={href} onClick={() => setSelected(null)}>
                        View resource
                      </Link>
                    </dd>
                  </div>
                )
              })()}
            </dl>

            <div className="audit-meta-block">
              <p className="label">Metadata</p>
              <pre className="audit-meta-pre">
                {JSON.stringify(selected.metadata ?? {}, null, 2)}
              </pre>
            </div>

            <div className="dialog-actions">
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => setSelected(null)}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
