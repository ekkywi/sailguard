import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react'
import { useNavigate, useOutletContext } from 'react-router-dom'
import { PermissionDenied } from '../../components/PermissionDenied'
import { SortableTh } from '../../components/SortableTh'
import { TableSkeleton } from '../../components/TableSkeleton'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatRelativeTime } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import {
  compareDate,
  compareString,
  useClientList,
  type ClientListCompare,
} from '../../lib/useClientList'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { createGroup, listGroups } from './api'
import type { CreateGroupInput, DeviceGroup } from './types'

const PAGE_SIZE = 10

function matchesGroup(g: DeviceGroup, q: string): boolean {
  return (
    g.name.toLowerCase().includes(q) ||
    (g.description || '').toLowerCase().includes(q) ||
    g.id.toLowerCase().includes(q)
  )
}

const GROUP_COMPARE: ClientListCompare<DeviceGroup> = {
  name: (a, b) => compareString(a.name, b.name),
  description: (a, b) =>
    compareString(a.description || '', b.description || ''),
  created_at: (a, b) => compareDate(a.created_at, b.created_at),
}

export default function GroupsPage() {
  useDocumentTitle('Groups')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'device.read')
  const canWrite = hasPermission(me, 'device.write')
  const navigate = useNavigate()
  const [groups, setGroups] = useState<DeviceGroup[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [creating, setCreating] = useState(false)
  const [nameError, setNameError] = useState<string | null>(null)
  const createTitleId = useId()
  const createFirstRef = useRef<HTMLInputElement>(null)

  const list = useClientList({
    items: groups,
    pageSize: PAGE_SIZE,
    matches: matchesGroup,
    compare: GROUP_COMPARE,
    defaultSortKey: 'name',
  })

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listGroups()
      .then((data) => setGroups(data.items))
      .catch((err: unknown) => {
        setGroups(null)
        setError(err instanceof Error ? err.message : 'failed to load groups')
      })
      .finally(() => setLoading(false))
  }, [canRead])

  useEffect(() => {
    load()
  }, [load])

  function openCreate() {
    setName('')
    setDescription('')
    setNameError(null)
    setError(null)
    setCreateOpen(true)
  }

  function closeCreate() {
    if (creating) return
    setCreateOpen(false)
    setName('')
    setDescription('')
    setNameError(null)
  }

  useEffect(() => {
    if (!createOpen) return
    createFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !creating) {
        setCreateOpen(false)
        setName('')
        setDescription('')
        setNameError(null)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [createOpen, creating])

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) {
      setNameError('Name is required')
      return
    }

    setCreating(true)
    setError(null)
    setNameError(null)

    const body: CreateGroupInput = { name: trimmed }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      await createGroup(body)
      setCreateOpen(false)
      setName('')
      setDescription('')
      list.setPage(1)
      toast.success('Group created')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create group'
      setError(msg)
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="device.read" title="Groups" />
  }

  const hasSource = (groups?.length ?? 0) > 0
  const showNoMatches = hasSource && list.total === 0

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Inventory</p>
          <h1 className="title">Groups</h1>
          <p className="lede">
            Device groups
            {groups ? (
              <span className="page-meta">
                {' '}
                · {groups.length} group{groups.length === 1 ? '' : 's'}
              </span>
            ) : null}
          </p>
        </div>
        <div className="btn-row">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={load}
            disabled={loading}
          >
            Refresh
          </button>
          {canWrite ? (
            <button
              type="button"
              className="btn btn-primary"
              onClick={openCreate}
            >
              Create group
            </button>
          ) : null}
        </div>
      </header>

      <div className="audit-toolbar" role="search">
        <div className="audit-filters">
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="groups-q">
              Search
            </label>
            <input
              id="groups-q"
              className="input"
              value={list.queryDraft}
              onChange={(e) => list.setQueryDraft(e.target.value)}
              placeholder="Name, description, or id…"
            />
          </div>
          {list.hasActiveQuery ? (
            <div className="field audit-filter-actions">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={list.clearSearch}
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

      <section className="list-panel" aria-labelledby="group-list-heading">
        <div className="list-panel-header">
          <h2 id="group-list-heading" className="panel-title">
            All groups
          </h2>
          {list.total > 0 ? (
            <p className="muted list-range">
              Showing {list.rangeStart}–{list.rangeEnd} of {list.total}
              {list.filteredFromSource
                ? ` (filtered from ${list.sourceTotal})`
                : ''}
            </p>
          ) : null}
        </div>

        {loading && !groups ? (
          <TableSkeleton label="Loading groups…" />
        ) : null}

        {!loading && groups && groups.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No groups yet</p>
            <p className="muted">
              {canWrite
                ? 'Create a group such as IT or Finance.'
                : 'Ask an admin with device.write to create groups.'}
            </p>
            {canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openCreate}
                >
                  Create group
                </button>
              </div>
            ) : null}
          </div>
        ) : null}

        {showNoMatches ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No matches</p>
            <p className="muted">Try another search or clear filters.</p>
            <div className="empty-actions">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={list.clearSearch}
              >
                Clear search
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
                      label="Name"
                      columnKey="name"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Description"
                      columnKey="description"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Created"
                      columnKey="created_at"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                  </tr>
                </thead>
                <tbody>
                  {list.pageItems.map((g) => (
                    <tr
                      key={g.id}
                      className="table-row-link"
                      tabIndex={0}
                      onClick={() => navigate(`/groups/${g.id}`)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          navigate(`/groups/${g.id}`)
                        }
                      }}
                    >
                      <td>
                        <div className="cell-primary">{g.name}</div>
                        <div className="muted mono">{g.id}</div>
                      </td>
                      <td className="muted">{g.description || '—'}</td>
                      <td className="muted" title={g.created_at}>
                        {formatRelativeTime(g.created_at)}
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
      </section>

      {createOpen ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={creating}
            onClick={closeCreate}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={createTitleId}
          >
            <h2 id={createTitleId} className="dialog-title">
              Create group
            </h2>
            <p className="dialog-desc muted">
              Groups scope policy assignments to a set of devices.
            </p>
            <form className="users-dialog-form" onSubmit={onCreate}>
              <div className="field">
                <label className="label" htmlFor="group-name">
                  Name
                </label>
                <input
                  ref={createFirstRef}
                  id="group-name"
                  className={nameError ? 'input input-invalid' : 'input'}
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    setNameError(null)
                  }}
                  placeholder="IT"
                  required
                  disabled={creating}
                  aria-invalid={nameError ? true : undefined}
                />
                {nameError ? (
                  <p className="field-error" role="alert">
                    {nameError}
                  </p>
                ) : null}
              </div>
              <div className="field">
                <label className="label" htmlFor="group-desc">
                  Description (optional)
                </label>
                <input
                  id="group-desc"
                  className="input"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Endpoints for the IT department"
                  disabled={creating}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={closeCreate}
                  disabled={creating}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={creating}
                  aria-busy={creating}
                >
                  {creating ? 'Creating…' : 'Create group'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}
