import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { SortableTh } from '../../components/SortableTh'
import { TableSkeleton } from '../../components/TableSkeleton'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { hasPermission } from '../../lib/permissions'
import {
  compareDate,
  compareNumber,
  compareString,
  useClientList,
  type ClientListCompare,
} from '../../lib/useClientList'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import {
  listAlertChannels,
  listAlertDeliveries,
  patchAlertChannel,
  retryAlertDelivery,
  testAlertChannel,
} from './api'
import type { AlertChannel, AlertDelivery } from './types'

type DeliveryStatusFilter =
  | ''
  | 'pending'
  | 'delivered'
  | 'failed'
  | 'dead'

const STATUS_CHIPS: { id: DeliveryStatusFilter; label: string }[] = [
  { id: '', label: 'All' },
  { id: 'pending', label: 'Pending' },
  { id: 'delivered', label: 'Delivered' },
  { id: 'failed', label: 'Failed' },
  { id: 'dead', label: 'Dead' },
]

function matchesDelivery(d: AlertDelivery, q: string): boolean {
  return (
    d.status.toLowerCase().includes(q) ||
    d.event_id.toLowerCase().includes(q) ||
    (d.last_error || '').toLowerCase().includes(q)
  )
}

const DELIVERY_COMPARE: ClientListCompare<AlertDelivery> = {
  status: (a, b) => compareString(a.status, b.status),
  attempts: (a, b) => compareNumber(a.attempts, b.attempts),
  event_id: (a, b) => compareString(a.event_id, b.event_id),
  updated_at: (a, b) => compareDate(a.updated_at, b.updated_at),
}

function shortUrl(url: string): string {
  const t = url.trim()
  if (!t) return '—'
  if (t.length <= 64) return t
  return `${t.slice(0, 61)}…`
}

export default function NotificationsPage() {
  useDocumentTitle('Notifications · Settings')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'alert.read')
  const canManage = hasPermission(me, 'alert.manage')

  const [channel, setChannel] = useState<AlertChannel | null>(null)
  const [url, setUrl] = useState('')
  const [enabled, setEnabled] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [editOpen, setEditOpen] = useState(false)
  const editTitleId = useId()
  const editFirstRef = useRef<HTMLInputElement>(null)

  const [deliveries, setDeliveries] = useState<AlertDelivery[] | null>(null)
  const [deliveriesLoading, setDeliveriesLoading] = useState(false)
  const [retryingId, setRetryingId] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<DeliveryStatusFilter>('')

  const statusFilterFn = useMemo(() => {
    if (!statusFilter) return undefined
    return (d: AlertDelivery) => d.status === statusFilter
  }, [statusFilter])

  const list = useClientList({
    items: deliveries,
    pageSize: 25,
    matches: matchesDelivery,
    compare: DELIVERY_COMPARE,
    defaultSortKey: 'updated_at',
    defaultSortDir: 'desc',
    filter: statusFilterFn,
  })

  const loadDeliveries = useCallback(() => {
    if (!canRead) return
    setDeliveriesLoading(true)
    listAlertDeliveries()
      .then((data) => setDeliveries(data.items))
      .catch(() => setDeliveries([]))
      .finally(() => setDeliveriesLoading(false))
  }, [canRead])

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listAlertChannels()
      .then((data) => {
        const ch = data.items[0] ?? null
        setChannel(ch)
        setUrl(ch?.url ?? '')
        setEnabled(ch?.enabled ?? false)
      })
      .catch((err: unknown) => {
        setChannel(null)
        setError(
          err instanceof Error ? err.message : 'failed to load alert channels',
        )
      })
      .finally(() => setLoading(false))
    loadDeliveries()
  }, [canRead, loadDeliveries])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!editOpen) return
    editFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !saving) setEditOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [editOpen, saving])

  function openEdit() {
    if (!channel) return
    setUrl(channel.url)
    setEnabled(channel.enabled)
    setError(null)
    setEditOpen(true)
  }

  function closeEdit() {
    if (saving) return
    setEditOpen(false)
    if (channel) {
      setUrl(channel.url)
      setEnabled(channel.enabled)
    }
  }

  const metaDirty =
    !!channel &&
    (url.trim() !== (channel.url || '') || enabled !== channel.enabled)

  async function onSave(e: FormEvent) {
    e.preventDefault()
    if (!channel || !canManage) return
    setSaving(true)
    setError(null)
    try {
      const updated = await patchAlertChannel(channel.id, {
        url: url.trim(),
        enabled,
      })
      setChannel(updated)
      setUrl(updated.url)
      setEnabled(updated.enabled)
      setEditOpen(false)
      toast.success('Webhook settings saved')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to save'
      setError(msg)
      toast.error(msg)
    } finally {
      setSaving(false)
    }
  }

  async function onTest() {
    if (!channel || !canManage) return
    setTesting(true)
    setError(null)
    try {
      await testAlertChannel(channel.id)
      toast.success('Test webhook sent')
      loadDeliveries()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'test failed'
      setError(msg)
      toast.error(msg)
    } finally {
      setTesting(false)
    }
  }

  async function onRetry(id: string) {
    if (!canManage) return
    setRetryingId(id)
    setError(null)
    try {
      await retryAlertDelivery(id)
      toast.success('Delivery re-queued')
      loadDeliveries()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'retry failed'
      setError(msg)
      toast.error(msg)
    } finally {
      setRetryingId(null)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="alert.read" title="Notifications" />
  }

  const source = deliveries ?? []
  const problemDeliveries = source.filter(
    (d) => d.status === 'failed' || d.status === 'dead' || d.status === 'pending',
  )
  const hasFilters = list.hasActiveQuery || Boolean(statusFilter)
  const showNoMatches = source.length > 0 && list.total === 0
  const canTest = Boolean(channel?.url?.trim())

  function clearFilters() {
    list.clearSearch()
    setStatusFilter('')
    list.setPage(1)
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Settings</p>
          <h1 className="title">Notifications</h1>
          <p className="lede">
            JSON webhook on new violations. Failed deliveries retry with
            backoff; dead ones need a manual retry.
          </p>
        </div>
        <div className="page-header-actions">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={load}
            disabled={loading || deliveriesLoading}
          >
            Refresh
          </button>
          {channel && canManage ? (
            <button type="button" className="btn btn-ghost" onClick={openEdit}>
              Edit webhook
            </button>
          ) : null}
          {channel && canManage ? (
            <button
              type="button"
              className="btn btn-ghost"
              onClick={onTest}
              disabled={testing || !canTest}
            >
              {testing ? 'Sending…' : 'Send test'}
            </button>
          ) : null}
        </div>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading && !channel ? <TableSkeleton label="Loading…" /> : null}

      {!loading && !channel ? (
        <div className="empty-state" role="status">
          <p className="empty-title">No alert channel</p>
          <p className="muted">
            Run migrations so the default webhook channel exists.
          </p>
        </div>
      ) : null}

      {channel ? (
        <div className="info-list" aria-label="Webhook channel">
          <div className="info-row">
            <span className="info-key">Status</span>
            <span className="info-val">
              <span
                className={
                  channel.enabled
                    ? 'badge badge-status-active'
                    : 'badge badge-status-disabled'
                }
              >
                {channel.enabled ? 'Enabled' : 'Disabled'}
              </span>
            </span>
          </div>
          <div className="info-row">
            <span className="info-key">Webhook URL</span>
            <span className="info-val mono" title={channel.url || undefined}>
              {shortUrl(channel.url)}
            </span>
          </div>
          <div className="info-row">
            <span className="info-key">Kind</span>
            <span className="info-val mono">{channel.kind || 'webhook'}</span>
          </div>
        </div>
      ) : null}

      <section className="list-panel" aria-labelledby="deliveries-heading">
        <div className="list-panel-header">
          <h2 id="deliveries-heading" className="panel-title">
            Recent deliveries
          </h2>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={loadDeliveries}
            disabled={deliveriesLoading}
          >
            Refresh
          </button>
        </div>

        <div className="audit-toolbar" role="search">
          <div className="audit-chips" aria-label="Delivery status">
            {STATUS_CHIPS.map((c) => (
              <button
                key={c.id || 'all-status'}
                type="button"
                className={
                  statusFilter === c.id
                    ? 'audit-chip audit-chip-on'
                    : 'audit-chip'
                }
                onClick={() => {
                  setStatusFilter(c.id)
                  list.setPage(1)
                }}
              >
                {c.label}
              </button>
            ))}
          </div>
          <div className="audit-filters">
            <div className="field audit-filter-grow">
              <label className="label" htmlFor="deliveries-q">
                Search
              </label>
              <input
                id="deliveries-q"
                className="input"
                value={list.queryDraft}
                onChange={(e) => list.setQueryDraft(e.target.value)}
                placeholder="Status, event id, or error…"
              />
            </div>
            {hasFilters ? (
              <div className="field audit-filter-actions">
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

        {list.total > 0 ? (
          <p className="muted list-range">
            Showing {list.rangeStart}–{list.rangeEnd} of {list.total}
            {list.filteredFromSource
              ? ` (filtered from ${list.sourceTotal})`
              : ''}
          </p>
        ) : null}

        {deliveriesLoading && !deliveries ? (
          <TableSkeleton label="Loading deliveries…" />
        ) : null}

        {!deliveriesLoading && deliveries && deliveries.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No deliveries yet</p>
            <p className="muted">
              New violations enqueue a delivery when the webhook channel is
              enabled.
            </p>
          </div>
        ) : null}

        {showNoMatches ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No matches</p>
            <p className="muted">Try another status or clear filters.</p>
            <div className="empty-actions">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={clearFilters}
              >
                Clear filters
              </button>
            </div>
          </div>
        ) : null}

        {list.pageItems.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <SortableTh
                      label="Status"
                      columnKey="status"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Attempts"
                      columnKey="attempts"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Event"
                      columnKey="event_id"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <th scope="col">Error</th>
                    <SortableTh
                      label="Updated"
                      columnKey="updated_at"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <th scope="col">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {list.pageItems.map((d) => (
                    <tr key={d.id}>
                      <td>
                        <code className={`audit-action${statusClass(d.status)}`}>
                          {d.status}
                        </code>
                      </td>
                      <td>
                        {d.attempts}/{d.max_attempts}
                      </td>
                      <td className="mono" title={d.event_id}>
                        {d.event_id.slice(0, 8)}…
                      </td>
                      <td className="muted" title={d.last_error ?? ''}>
                        {d.last_error
                          ? d.last_error.length > 48
                            ? `${d.last_error.slice(0, 48)}…`
                            : d.last_error
                          : '—'}
                      </td>
                      <td className="muted">{formatWhen(d.updated_at)}</td>
                      <td>
                        {canManage &&
                        (d.status === 'failed' || d.status === 'dead') ? (
                          <button
                            type="button"
                            className="btn btn-ghost"
                            disabled={retryingId === d.id}
                            onClick={() => onRetry(d.id)}
                          >
                            {retryingId === d.id ? 'Retrying…' : 'Retry'}
                          </button>
                        ) : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {list.pageCount > 1 ? (
              <div className="pagination">
                <p className="muted pagination-meta">
                  Page {list.page} of {list.pageCount}
                </p>
                <div className="pagination-actions">
                  <button
                    type="button"
                    className="btn btn-ghost"
                    disabled={list.page <= 1}
                    onClick={() => list.setPage((p) => Math.max(1, p - 1))}
                  >
                    Previous
                  </button>
                  <button
                    type="button"
                    className="btn btn-ghost"
                    disabled={list.page >= list.pageCount}
                    onClick={() =>
                      list.setPage((p) => Math.min(list.pageCount, p + 1))
                    }
                  >
                    Next
                  </button>
                </div>
              </div>
            ) : null}
          </>
        ) : null}

        {problemDeliveries.length > 0 ? (
          <p className="muted" style={{ marginTop: '0.75rem' }}>
            {problemDeliveries.filter((d) => d.status === 'dead').length} dead,{' '}
            {problemDeliveries.filter((d) => d.status === 'failed').length}{' '}
            failed,{' '}
            {problemDeliveries.filter((d) => d.status === 'pending').length}{' '}
            pending.
          </p>
        ) : null}
      </section>

      {editOpen && channel ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={saving}
            onClick={closeEdit}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={editTitleId}
          >
            <h2 id={editTitleId} className="dialog-title">
              Edit webhook
            </h2>
            <p className="dialog-desc muted">
              POST JSON when a new violation is persisted. Leave disabled to
              keep deliveries queued off.
            </p>
            <form className="users-dialog-form" onSubmit={onSave}>
              <div className="field">
                <label className="label" htmlFor="webhook-url">
                  Webhook URL
                </label>
                <input
                  ref={editFirstRef}
                  id="webhook-url"
                  className="input"
                  type="url"
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://webhook.site/…"
                  disabled={saving}
                  autoComplete="off"
                />
              </div>
              <label className="role-option settings-check">
                <input
                  type="checkbox"
                  checked={enabled}
                  onChange={(e) => setEnabled(e.target.checked)}
                  disabled={saving}
                />
                <span className="role-option-body">
                  <span className="role-option-code">Enabled</span>
                  <span className="role-option-desc muted">
                    Deliver on new violations only
                  </span>
                </span>
              </label>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={closeEdit}
                  disabled={saving}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={saving || !metaDirty}
                  aria-busy={saving}
                >
                  {saving ? 'Saving…' : 'Save'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}

function statusClass(status: string): string {
  if (status === 'failed' || status === 'dead') return ' audit-action--warn'
  return ''
}

function formatWhen(iso: string): string {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}
