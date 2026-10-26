import { FormEvent, useCallback, useEffect, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { TableSkeleton } from '../../components/TableSkeleton'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import {
  listAlertChannels,
  listAlertDeliveries,
  patchAlertChannel,
  retryAlertDelivery,
  testAlertChannel,
} from './api'
import type { AlertChannel, AlertDelivery } from './types'

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

  const [deliveries, setDeliveries] = useState<AlertDelivery[]>([])
  const [deliveriesLoading, setDeliveriesLoading] = useState(false)
  const [retryingId, setRetryingId] = useState<string | null>(null)

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
      toast.success('Notification settings saved')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'failed to save')
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
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'test failed')
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
      setError(err instanceof Error ? err.message : 'retry failed')
    } finally {
      setRetryingId(null)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="alert.read" title="Notifications" />
  }

  const problemDeliveries = deliveries.filter(
    (d) => d.status === 'failed' || d.status === 'dead' || d.status === 'pending',
  )

  return (
    <div className="stack">
      <section className="list-panel" aria-labelledby="notif-heading">
        <div className="list-panel-header">
          <h2 id="notif-heading" className="panel-title">
            Notifications
          </h2>
        </div>
        <p className="muted settings-lede">
          Send a JSON webhook when a new violation is persisted. Failed
          deliveries are retried with backoff; dead deliveries need a manual
          retry.
        </p>

        {error ? (
          <p className="error" role="alert">
            {error}
          </p>
        ) : null}

        {loading ? <TableSkeleton label="Loading…" /> : null}

        {!loading && !channel ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No alert channel</p>
            <p className="muted">
              Run migrations so the default webhook channel exists.
            </p>
          </div>
        ) : null}

        {!loading && channel ? (
          <form className="stack settings-form" onSubmit={onSave}>
            <div className="field">
              <label className="label" htmlFor="webhook-url">
                Webhook URL
              </label>
              <input
                id="webhook-url"
                className="input"
                type="url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://webhook.site/…"
                disabled={!canManage || saving}
                autoComplete="off"
              />
            </div>
            <label className="role-option settings-check">
              <input
                type="checkbox"
                checked={enabled}
                onChange={(e) => setEnabled(e.target.checked)}
                disabled={!canManage || saving}
              />
              <span className="role-option-body">
                <span className="role-option-code">Enabled</span>
                <span className="role-option-desc muted">
                  Deliver on new violations only
                </span>
              </span>
            </label>
            <div className="btn-row">
              {canManage ? (
                <>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={saving}
                  >
                    {saving ? 'Saving…' : 'Save'}
                  </button>
                  <button
                    type="button"
                    className="btn btn-ghost"
                    onClick={onTest}
                    disabled={testing || !url.trim()}
                  >
                    {testing ? 'Sending…' : 'Send test'}
                  </button>
                </>
              ) : (
                <p className="muted">You need alert.manage to change this.</p>
              )}
            </div>
          </form>
        ) : null}
      </section>

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
        <p className="muted settings-lede">
          Status of webhook sends for persisted violations (pending, failed,
          dead, delivered).
        </p>

        {deliveriesLoading && deliveries.length === 0 ? (
          <TableSkeleton label="Loading deliveries…" />
        ) : null}

        {!deliveriesLoading && deliveries.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No deliveries yet</p>
            <p className="muted">
              New violations enqueue a delivery when the webhook channel is
              enabled.
            </p>
          </div>
        ) : null}

        {deliveries.length > 0 ? (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Status</th>
                  <th scope="col">Attempts</th>
                  <th scope="col">Event</th>
                  <th scope="col">Error</th>
                  <th scope="col">Updated</th>
                  <th scope="col">Actions</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map((d) => (
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
