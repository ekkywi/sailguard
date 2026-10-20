import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useOutletContext, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { PermissionDenied } from '../../components/PermissionDenied'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import {
  formatRelativeTime,
  statusBadgeClass,
} from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { listDevices } from '../devices/api'
import type { Device } from '../devices/types'
import {
  addGroupMember,
  deleteGroup,
  getGroup,
  listGroupMembers,
  patchGroup,
  removeGroupMember,
} from './api'
import type { DeviceGroup } from './types'

export default function GroupDetailPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'device.read')
  const canWrite = hasPermission(me, 'device.write')
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [group, setGroup] = useState<DeviceGroup | null>(null)
  const [members, setMembers] = useState<Device[] | null>(null)
  const [allDevices, setAllDevices] = useState<Device[] | null>(null)

  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [adding, setAdding] = useState(false)

  const [removeTarget, setRemoveTarget] = useState<Device | null>(null)
  const [removing, setRemoving] = useState(false)

  const [nameDraft, setNameDraft] = useState('')
  const [descriptionDraft, setDescriptionDraft] = useState('')
  const [savingMeta, setSavingMeta] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    if (!id) {
      setError('missing group id')
      setLoading(false)
      return
    }

    setLoading(true)
    setError(null)

    Promise.all([getGroup(id), listGroupMembers(id), listDevices()])
      .then(([g, memberData, deviceData]) => {
        setGroup(g)
        setNameDraft(g.name)
        setDescriptionDraft(g.description || '')
        setMembers(memberData.items)
        setAllDevices(deviceData.items)
      })
      .catch((err: unknown) => {
        setGroup(null)
        setMembers(null)
        setAllDevices(null)
        setError(err instanceof Error ? err.message : 'failed to load group')
      })
      .finally(() => setLoading(false))
  }, [id, canRead])

  useEffect(() => {
    load()
  }, [load])

  const memberIds = useMemo(
    () => new Set((members ?? []).map((d) => d.id)),
    [members],
  )

  const candidates = useMemo(() => {
    if (!allDevices) return []
    return allDevices.filter((d) => !memberIds.has(d.id))
  }, [allDevices, memberIds])

  const metaDirty =
    !!group &&
    (nameDraft.trim() !== group.name ||
      descriptionDraft.trim() !== (group.description || ''))

  if (!canRead) {
    return <PermissionDenied permission="device.read" title="Group" />
  }

  async function onAdd(e: FormEvent) {
    e.preventDefault()
    if (!id || !selectedDeviceId) {
      setError('select a device to add')
      return
    }

    setAdding(true)
    setError(null)
    try {
      await addGroupMember(id, selectedDeviceId)
      setSelectedDeviceId('')
      toast.success('Member added')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to add member'
      setError(msg)
      toast.error(msg)
    } finally {
      setAdding(false)
    }
  }

  async function onConfirmRemove() {
    if (!id || !removeTarget) return

    setRemoving(true)
    setError(null)
    try {
      await removeGroupMember(id, removeTarget.id)
      setRemoveTarget(null)
      toast.success('Member removed')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to remove member'
      setError(msg)
      toast.error(msg)
    } finally {
      setRemoving(false)
    }
  }

  async function onSaveMeta(e: FormEvent) {
    e.preventDefault()
    if (!id || !group) return
    const name = nameDraft.trim()
    if (!name) {
      setError('name is required')
      return
    }
    setSavingMeta(true)
    setError(null)
    try {
      const updated = await patchGroup(id, {
        name,
        description: descriptionDraft.trim(),
      })
      setGroup(updated)
      setNameDraft(updated.name)
      setDescriptionDraft(updated.description || '')
      toast.success('Group updated')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to update group'
      setError(msg)
      toast.error(msg)
    } finally {
      setSavingMeta(false)
    }
  }

  async function onConfirmDelete() {
    if (!id) return
    setDeleting(true)
    setError(null)
    try {
      await deleteGroup(id)
      toast.success('Group deleted')
      navigate('/groups')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to delete group'
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
            <Link to="/groups" className="crumb">
              Groups
            </Link>
            <span className="crumb-sep">/</span>
            Detail
          </p>
          <h1 className="title">{group?.name ?? 'Group'}</h1>
          <p className="lede">
            {group?.description || 'Assign enrolled devices to this group.'}
            {members ? (
              <span className="page-meta">
                {' '}
                · {members.length} member{members.length === 1 ? '' : 's'}
              </span>
            ) : null}
          </p>
        </div>
        <div className="page-header-actions">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={load}
            disabled={loading}
          >
            Refresh
          </button>
          {group && canWrite ? (
            <button
              type="button"
              className="btn btn-danger"
              onClick={() => setConfirmDelete(true)}
            >
              Delete
            </button>
          ) : null}
        </div>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading && !group ? <p className="loading">Loading group…</p> : null}

      {group ? (
        <>
          <div className="info-list" aria-label="Group details">
            <div className="info-row">
              <span className="info-key">Created</span>
              <span className="info-val" title={group.created_at}>
                {formatRelativeTime(group.created_at)}
              </span>
            </div>
            <div className="info-row">
              <span className="info-key">ID</span>
              <span className="info-val mono">{group.id}</span>
            </div>
          </div>

          {canWrite ? (
            <section className="panel" aria-labelledby="edit-group-heading">
              <h2 id="edit-group-heading" className="panel-title">
                Edit group
              </h2>
              <form className="token-create-grid" onSubmit={onSaveMeta}>
                <div className="field" style={{ gridColumn: '1 / -1' }}>
                  <label className="label" htmlFor="group-name">
                    Name
                  </label>
                  <input
                    id="group-name"
                    className="input"
                    value={nameDraft}
                    onChange={(e) => setNameDraft(e.target.value)}
                    required
                  />
                </div>
                <div className="field" style={{ gridColumn: '1 / -1' }}>
                  <label className="label" htmlFor="group-description">
                    Description
                  </label>
                  <input
                    id="group-description"
                    className="input"
                    value={descriptionDraft}
                    onChange={(e) => setDescriptionDraft(e.target.value)}
                    placeholder="optional"
                  />
                </div>
                <div className="token-create-submit">
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={savingMeta || !metaDirty}
                  >
                    {savingMeta ? 'Saving…' : 'Save'}
                  </button>
                </div>
              </form>
            </section>
          ) : null}
        </>
      ) : null}

      {group && canWrite ? (
        <section className="panel" aria-labelledby="add-member-heading">
          <h2 id="add-member-heading" className="panel-title">
            Add member
          </h2>
          {candidates.length === 0 ? (
            <p className="muted">
              {allDevices && allDevices.length === 0
                ? 'No enrolled devices yet. Enroll an agent first.'
                : 'All devices are already in this group.'}
            </p>
          ) : (
            <form className="token-create-grid" onSubmit={onAdd}>
              <div className="field" style={{ gridColumn: 'span 2' }}>
                <label className="label" htmlFor="member-device">
                  Device
                </label>
                <select
                  id="member-device"
                  className="input"
                  value={selectedDeviceId}
                  onChange={(e) => setSelectedDeviceId(e.target.value)}
                  required
                >
                  <option value="">Select a device…</option>
                  {candidates.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.hostname}
                      {d.display_name ? ` (${d.display_name})` : ''}
                      {` · ${d.os}`}
                    </option>
                  ))}
                </select>
              </div>
              <div className="token-create-submit">
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={adding || !selectedDeviceId}
                >
                  {adding ? 'Adding…' : 'Add to group'}
                </button>
              </div>
            </form>
          )}
        </section>
      ) : null}

      <section className="list-panel" aria-labelledby="member-list-heading">
        <div className="list-panel-header">
          <h2 id="member-list-heading" className="panel-title">
            Members
          </h2>
        </div>

        {loading && !members ? (
          <p className="loading">Loading members…</p>
        ) : null}

        {!loading && members && members.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No members yet</p>
            <p className="muted">Add an enrolled device above.</p>
          </div>
        ) : null}

        {members && members.length > 0 ? (
          <div className="table-wrap table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Hostname</th>
                  <th>OS</th>
                  <th>Status</th>
                  <th>Last seen</th>
                  {canWrite ? <th /> : null}
                </tr>
              </thead>
              <tbody>
                {members.map((d) => (
                  <tr key={d.id}>
                    <td>
                      <Link to={`/devices/${d.id}`} className="cell-primary">
                        {d.hostname}
                      </Link>
                      {d.machine_guid ? (
                        <div className="muted mono">{d.machine_guid}</div>
                      ) : null}
                    </td>
                    <td>
                      {d.os}
                      {d.os_version ? (
                        <span className="muted"> {d.os_version}</span>
                      ) : null}
                    </td>
                    <td>
                      <span className={statusBadgeClass(d.status)}>
                        {d.status}
                      </span>
                    </td>
                    <td
                      className="muted"
                      title={d.last_seen_at ?? undefined}
                    >
                      {formatRelativeTime(d.last_seen_at)}
                    </td>
                    {canWrite ? (
                      <td>
                        <button
                          type="button"
                          className="btn btn-ghost"
                          onClick={() => setRemoveTarget(d)}
                        >
                          Remove
                        </button>
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      <ConfirmDialog
        open={removeTarget !== null}
        title="Remove group member?"
        description={
          removeTarget
            ? `“${removeTarget.hostname}” will leave this group. The device itself stays enrolled.`
            : ''
        }
        confirmLabel="Remove"
        cancelLabel="Cancel"
        danger
        busy={removing}
        onConfirm={onConfirmRemove}
        onCancel={() => {
          if (!removing) setRemoveTarget(null)
        }}
      />

      <ConfirmDialog
        open={confirmDelete}
        title="Delete group?"
        description="Membership and policy assignments for this group will be removed. Devices stay enrolled."
        confirmLabel="Delete group"
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
