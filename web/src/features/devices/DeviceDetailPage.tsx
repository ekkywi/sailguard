import { FormEvent, useCallback, useEffect, useState } from 'react'
import {
  Link,
  useNavigate,
  useOutletContext,
  useParams,
  useSearchParams,
} from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { PermissionDenied } from '../../components/PermissionDenied'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import {
  formatDateTime,
  formatRelativeTime,
  statusBadgeClass,
} from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { deleteDevice, getDevice, patchDevice } from './api'
import DeviceOverridesPanel from './DeviceOverridesPanel'
import type { Device } from './types'

type DeviceTab = 'overview' | 'overrides'

function parseTab(raw: string | null, canPolicyRead: boolean): DeviceTab {
  if (raw === 'overrides' && canPolicyRead) return 'overrides'
  return 'overview'
}

export default function DeviceDetailPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'device.read')
  const canWrite = hasPermission(me, 'device.write')
  const canPolicyRead = hasPermission(me, 'policy.read')
  const canPolicyWrite = hasPermission(me, 'policy.write')
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = parseTab(searchParams.get('tab'), canPolicyRead)

  const [device, setDevice] = useState<Device | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [displayNameDraft, setDisplayNameDraft] = useState('')
  const [savingName, setSavingName] = useState(false)
  const [togglingStatus, setTogglingStatus] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    if (!id) {
      setError('missing device id')
      setLoading(false)
      return
    }
    setLoading(true)
    getDevice(id)
      .then((d) => {
        setDevice(d)
        setDisplayNameDraft(d.display_name || '')
        setError(null)
      })
      .catch((err: unknown) => {
        setDevice(null)
        setError(err instanceof Error ? err.message : 'failed to load device')
      })
      .finally(() => setLoading(false))
  }, [id, canRead])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (searchParams.get('tab') === 'overrides' && !canPolicyRead) {
      setSearchParams({}, { replace: true })
    }
  }, [canPolicyRead, searchParams, setSearchParams])

  if (!canRead) {
    return <PermissionDenied permission="device.read" title="Device" />
  }

  function setTab(next: DeviceTab) {
    if (next === 'overview') {
      setSearchParams({}, { replace: true })
      return
    }
    setSearchParams({ tab: next }, { replace: true })
  }

  async function onSaveDisplayName(e: FormEvent) {
    e.preventDefault()
    if (!id || !device) return
    setSavingName(true)
    setError(null)
    try {
      const updated = await patchDevice(id, {
        display_name: displayNameDraft.trim(),
      })
      setDevice(updated)
      setDisplayNameDraft(updated.display_name || '')
      toast.success('Display name saved')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to update device'
      setError(msg)
      toast.error(msg)
    } finally {
      setSavingName(false)
    }
  }

  async function onToggleStatus() {
    if (!id || !device) return
    const next = device.status === 'disabled' ? 'active' : 'disabled'
    setTogglingStatus(true)
    setError(null)
    try {
      const updated = await patchDevice(id, { status: next })
      setDevice(updated)
      toast.success(next === 'disabled' ? 'Device disabled' : 'Device enabled')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to update status'
      setError(msg)
      toast.error(msg)
    } finally {
      setTogglingStatus(false)
    }
  }

  async function onConfirmDelete() {
    if (!id) return
    setDeleting(true)
    setError(null)
    try {
      await deleteDevice(id)
      toast.success('Device deleted')
      navigate('/devices')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to delete device'
      setError(msg)
      toast.error(msg)
      setDeleting(false)
      setConfirmDelete(false)
    }
  }

  const lede =
    tab === 'overrides'
      ? 'Emergency and time-bounded policy exceptions for this endpoint.'
      : 'Enrollment and agent metadata for this endpoint.'

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">
            <Link to="/devices" className="crumb">
              Devices
            </Link>
            <span className="crumb-sep">/</span>
            Detail
          </p>
          <h1 className="title">{device?.hostname ?? 'Device'}</h1>
          <p className="lede">{lede}</p>
        </div>
        {device && canWrite ? (
          <div className="page-header-actions">
            <button
              type="button"
              className="btn btn-ghost"
              disabled={togglingStatus}
              onClick={onToggleStatus}
            >
              {togglingStatus
                ? 'Updating…'
                : device.status === 'disabled'
                  ? 'Enable'
                  : 'Disable'}
            </button>
            <button
              type="button"
              className="btn btn-danger"
              onClick={() => setConfirmDelete(true)}
            >
              Delete
            </button>
          </div>
        ) : null}
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading ? <p className="loading">Loading device…</p> : null}

      {device ? (
        <>
          <nav className="detail-tabs" aria-label="Device sections">
            <button
              type="button"
              className={
                tab === 'overview'
                  ? 'detail-tab detail-tab-active'
                  : 'detail-tab'
              }
              aria-current={tab === 'overview' ? 'page' : undefined}
              onClick={() => setTab('overview')}
            >
              Overview
            </button>
            {canPolicyRead ? (
              <button
                type="button"
                className={
                  tab === 'overrides'
                    ? 'detail-tab detail-tab-active'
                    : 'detail-tab'
                }
                aria-current={tab === 'overrides' ? 'page' : undefined}
                onClick={() => setTab('overrides')}
              >
                Overrides
              </button>
            ) : null}
          </nav>

          {tab === 'overview' ? (
            <div className="stack device-module">
              <div className="info-list" aria-label="Device details">
                <DetailRow label="Status">
                  <span className={statusBadgeClass(device.status)}>
                    {device.status}
                  </span>
                </DetailRow>
                <DetailRow label="Hostname">{device.hostname}</DetailRow>
                <DetailRow label="OS">
                  {device.os}
                  {device.os_version ? ` ${device.os_version}` : ''}
                </DetailRow>
                <DetailRow label="Agent">
                  {device.agent_version || '—'}
                </DetailRow>
                <DetailRow label="Machine GUID">
                  <span className="mono">{device.machine_guid ?? '—'}</span>
                </DetailRow>
                <DetailRow label="Last seen">
                  <span title={device.last_seen_at ?? undefined}>
                    {formatRelativeTime(device.last_seen_at)}
                    {device.last_seen_at ? (
                      <span className="muted">
                        {' '}
                        · {formatDateTime(device.last_seen_at)}
                      </span>
                    ) : null}
                  </span>
                </DetailRow>
                <DetailRow label="Enrolled">
                  <span title={device.enrolled_at ?? undefined}>
                    {formatRelativeTime(device.enrolled_at)}
                    {device.enrolled_at ? (
                      <span className="muted">
                        {' '}
                        · {formatDateTime(device.enrolled_at)}
                      </span>
                    ) : null}
                  </span>
                </DetailRow>
                <DetailRow label="Display name">
                  {device.display_name || '—'}
                </DetailRow>
                <DetailRow label="ID">
                  <span className="mono">{device.id}</span>
                </DetailRow>
              </div>

              {canWrite ? (
                <section
                  className="panel"
                  aria-labelledby="device-rename-heading"
                >
                  <h2 id="device-rename-heading" className="panel-title">
                    Display name
                  </h2>
                  <form
                    className="token-create-grid"
                    onSubmit={onSaveDisplayName}
                  >
                    <div className="field" style={{ gridColumn: '1 / -1' }}>
                      <label className="label" htmlFor="device-display-name">
                        Label shown in the console
                      </label>
                      <input
                        id="device-display-name"
                        className="input"
                        value={displayNameDraft}
                        onChange={(e) => setDisplayNameDraft(e.target.value)}
                        placeholder={device.hostname}
                      />
                    </div>
                    <div className="token-create-submit">
                      <button
                        type="submit"
                        className="btn btn-primary"
                        disabled={
                          savingName ||
                          displayNameDraft.trim() ===
                            (device.display_name || '')
                        }
                      >
                        {savingName ? 'Saving…' : 'Save'}
                      </button>
                    </div>
                  </form>
                </section>
              ) : null}
            </div>
          ) : null}

          {tab === 'overrides' && id && canPolicyRead ? (
            <DeviceOverridesPanel
              deviceId={id}
              canWrite={canPolicyWrite}
            />
          ) : null}
        </>
      ) : null}

      <ConfirmDialog
        open={confirmDelete}
        title="Delete device?"
        description="This permanently removes the device, credentials, and related events. The agent must enroll again to reappear."
        confirmLabel="Delete device"
        danger
        busy={deleting}
        onConfirm={onConfirmDelete}
        onCancel={() => {
          if (!deleting) setConfirmDelete(false)
        }}
      />
    </div>
  )
}

function DetailRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="info-row">
      <span className="info-key">{label}</span>
      <span className="info-val">{children}</span>
    </div>
  )
}
