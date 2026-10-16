import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { toast } from '../../components/toast'
import {
  formatDateTime,
  formatRelativeTime,
  statusBadgeClass,
} from '../../lib/format'
import { deleteDevice, getDevice, patchDevice } from './api'
import type { Device } from './types'

export default function DeviceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [device, setDevice] = useState<Device | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [displayNameDraft, setDisplayNameDraft] = useState('')
  const [savingName, setSavingName] = useState(false)
  const [togglingStatus, setTogglingStatus] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(() => {
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
  }, [id])

  useEffect(() => {
    load()
  }, [load])

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
          <p className="lede">
            Enrollment and agent metadata for this endpoint.
          </p>
        </div>
        {device ? (
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
            <DetailRow label="Agent">{device.agent_version || '—'}</DetailRow>
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
            <DetailRow label="ID">
              <span className="mono">{device.id}</span>
            </DetailRow>
          </div>

          <section className="panel" aria-labelledby="device-rename-heading">
            <h2 id="device-rename-heading" className="panel-title">
              Display name
            </h2>
            <form className="token-create-grid" onSubmit={onSaveDisplayName}>
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
                    displayNameDraft.trim() === (device.display_name || '')
                  }
                >
                  {savingName ? 'Saving…' : 'Save'}
                </button>
              </div>
            </form>
          </section>
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
