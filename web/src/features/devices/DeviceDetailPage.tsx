import { FormEvent, useCallback, useEffect, useId, useRef, useState } from 'react'
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
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { deleteDevice, getDevice, patchDevice } from './api'
import DeviceOverridesPanel from './DeviceOverridesPanel'
import type { Device } from './types'

type DeviceTab = 'overview' | 'overrides'

function parseTab(raw: string | null, canPolicyRead: boolean): DeviceTab {
  if (raw === 'overrides' && canPolicyRead) return 'overrides'
  return 'overview'
}

function draftsFromDevice(d: Device) {
  return {
    displayName: d.display_name || '',
    pic: d.pic || '',
    location: d.location || '',
    assetTag: d.asset_tag || '',
    notes: d.notes || '',
  }
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
  useDocumentTitle(device?.hostname ?? 'Device')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [displayNameDraft, setDisplayNameDraft] = useState('')
  const [picDraft, setPicDraft] = useState('')
  const [locationDraft, setLocationDraft] = useState('')
  const [assetTagDraft, setAssetTagDraft] = useState('')
  const [notesDraft, setNotesDraft] = useState('')
  const [assetOpen, setAssetOpen] = useState(false)
  const [savingAsset, setSavingAsset] = useState(false)
  const [togglingStatus, setTogglingStatus] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const assetTitleId = useId()
  const assetFirstRef = useRef<HTMLInputElement>(null)

  const applyDrafts = useCallback((d: Device) => {
    const next = draftsFromDevice(d)
    setDisplayNameDraft(next.displayName)
    setPicDraft(next.pic)
    setLocationDraft(next.location)
    setAssetTagDraft(next.assetTag)
    setNotesDraft(next.notes)
  }, [])

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
        applyDrafts(d)
        setError(null)
      })
      .catch((err: unknown) => {
        setDevice(null)
        setError(err instanceof Error ? err.message : 'failed to load device')
      })
      .finally(() => setLoading(false))
  }, [id, canRead, applyDrafts])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    const t = searchParams.get('tab')
    if (t === 'overrides' && !canPolicyRead) {
      setSearchParams({}, { replace: true })
    } else if (t === 'asset') {
      // Legacy deep-link: asset edit is a modal now.
      setSearchParams({}, { replace: true })
      if (canWrite) setAssetOpen(true)
    }
  }, [canPolicyRead, canWrite, searchParams, setSearchParams])

  useEffect(() => {
    if (!assetOpen) return
    assetFirstRef.current?.focus()

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !savingAsset) closeAsset()
    }

    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [assetOpen, savingAsset])

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

  function openAsset() {
    if (!device || !canWrite) return
    applyDrafts(device)
    setAssetOpen(true)
  }

  function closeAsset() {
    if (savingAsset) return
    setAssetOpen(false)
    if (device) applyDrafts(device)
  }

  function assetDirty(d: Device): boolean {
    return (
      displayNameDraft.trim() !== (d.display_name || '') ||
      picDraft.trim() !== (d.pic || '') ||
      locationDraft.trim() !== (d.location || '') ||
      assetTagDraft.trim() !== (d.asset_tag || '') ||
      notesDraft.trim() !== (d.notes || '')
    )
  }

  async function onSaveAsset(e: FormEvent) {
    e.preventDefault()
    if (!id || !device) return
    setSavingAsset(true)
    setError(null)
    try {
      const updated = await patchDevice(id, {
        display_name: displayNameDraft.trim(),
        pic: picDraft.trim(),
        location: locationDraft.trim(),
        asset_tag: assetTagDraft.trim(),
        notes: notesDraft.trim(),
      })
      setDevice(updated)
      applyDrafts(updated)
      setAssetOpen(false)
      toast.success('Asset details saved')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to update device'
      setError(msg)
      toast.error(msg)
    } finally {
      setSavingAsset(false)
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
      : 'Enrollment, connectivity, and asset metadata for this endpoint.'

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
            {tab === 'overview' ? (
              <button
                type="button"
                className="btn btn-ghost"
                onClick={openAsset}
              >
                Edit asset
              </button>
            ) : null}
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
              <section
                className="panel"
                aria-labelledby="device-identity-heading"
              >
                <h2 id="device-identity-heading" className="panel-title">
                  Identity
                </h2>
                <div className="info-list" aria-label="Device identity">
                  <DetailRow label="Status">
                    <span className={statusBadgeClass(device.status)}>
                      {device.status}
                    </span>
                  </DetailRow>
                  <DetailRow label="Hostname">{device.hostname}</DetailRow>
                  <DetailRow label="Display name">
                    {device.display_name || '—'}
                  </DetailRow>
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
                  <DetailRow label="ID">
                    <span className="mono">{device.id}</span>
                  </DetailRow>
                </div>
              </section>

              <section
                className="panel"
                aria-labelledby="device-asset-summary-heading"
              >
                <div className="list-panel-header">
                  <h2
                    id="device-asset-summary-heading"
                    className="panel-title"
                  >
                    Asset
                  </h2>
                  {canWrite ? (
                    <button
                      type="button"
                      className="btn btn-ghost"
                      onClick={openAsset}
                    >
                      Edit
                    </button>
                  ) : null}
                </div>
                <div className="info-list" aria-label="Asset summary">
                  <DetailRow label="PIC">{device.pic || '—'}</DetailRow>
                  <DetailRow label="Location">{device.location || '—'}</DetailRow>
                  <DetailRow label="Asset tag">
                    {device.asset_tag || '—'}
                  </DetailRow>
                  <DetailRow label="Notes">{device.notes || '—'}</DetailRow>
                </div>
              </section>
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

      {assetOpen && device && canWrite ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={savingAsset}
            onClick={closeAsset}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={assetTitleId}
          >
            <h2 id={assetTitleId} className="dialog-title">
              Edit asset details
            </h2>
            <p className="dialog-desc muted">
              Ownership and inventory fields for {device.hostname}.
            </p>
            <form className="users-dialog-form" onSubmit={onSaveAsset}>
              <div className="field">
                <label className="label" htmlFor="device-display-name">
                  Display name
                </label>
                <input
                  ref={assetFirstRef}
                  id="device-display-name"
                  className="input"
                  value={displayNameDraft}
                  onChange={(e) => setDisplayNameDraft(e.target.value)}
                  placeholder={device.hostname}
                  disabled={savingAsset}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="device-pic">
                  PIC
                </label>
                <input
                  id="device-pic"
                  className="input"
                  value={picDraft}
                  onChange={(e) => setPicDraft(e.target.value)}
                  placeholder="Owner or contact"
                  disabled={savingAsset}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="device-location">
                  Location
                </label>
                <input
                  id="device-location"
                  className="input"
                  value={locationDraft}
                  onChange={(e) => setLocationDraft(e.target.value)}
                  placeholder="Building / room"
                  disabled={savingAsset}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="device-asset-tag">
                  Asset tag
                </label>
                <input
                  id="device-asset-tag"
                  className="input mono"
                  value={assetTagDraft}
                  onChange={(e) => setAssetTagDraft(e.target.value)}
                  placeholder="Inventory number"
                  disabled={savingAsset}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="device-notes">
                  Notes
                </label>
                <textarea
                  id="device-notes"
                  className="input"
                  rows={3}
                  value={notesDraft}
                  onChange={(e) => setNotesDraft(e.target.value)}
                  placeholder="Optional notes"
                  disabled={savingAsset}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={closeAsset}
                  disabled={savingAsset}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={savingAsset || !assetDirty(device)}
                  aria-busy={savingAsset}
                >
                  {savingAsset ? 'Saving…' : 'Save'}
                </button>
              </div>
            </form>
          </div>
        </div>
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
