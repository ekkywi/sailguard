import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import {
  formatRelativeTime,
  statusBadgeClass,
} from '../../lib/format'
import { listDevices } from '../devices/api'
import type { Device } from '../devices/types'
import {
  addGroupMember,
  getGroup,
  listGroupMembers,
  removeGroupMember,
} from './api'
import type { DeviceGroup } from './types'

export default function GroupDetailPage() {
  const { id } = useParams<{ id: string }>()

  const [group, setGroup] = useState<DeviceGroup | null>(null)
  const [members, setMembers] = useState<Device[] | null>(null)
  const [allDevices, setAllDevices] = useState<Device[] | null>(null)

  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [selectedDeviceId, setSelectedDeviceId] = useState('')
  const [adding, setAdding] = useState(false)

  const [removeTarget, setRemoveTarget] = useState<Device | null>(null)
  const [removing, setRemoving] = useState(false)

  const load = useCallback(() => {
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
  }, [id])

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
      load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'failed to add member')
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
      load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'failed to remove member')
    } finally {
      setRemoving(false)
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

      {loading && !group ? <p className="loading">Loading group…</p> : null}

      {group ? (
        <div className="info-list" aria-label="Group details">
          <div className="info-row">
            <span className="info-key">Name</span>
            <span className="info-val">{group.name}</span>
          </div>
          <div className="info-row">
            <span className="info-key">Description</span>
            <span className="info-val">{group.description || '—'}</span>
          </div>
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
      ) : null}

      {group ? (
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
                  <th />
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
                    <td>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={() => setRemoveTarget(d)}
                      >
                        Remove
                      </button>
                    </td>
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
    </div>
  )
}
