import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { formatRelativeTime } from '../../lib/format'
import { createGroup, listGroups } from './api'
import type { CreateGroupInput, DeviceGroup } from './types'

const PAGE_SIZE = 10

export default function GroupsPage() {
  const [groups, setGroups] = useState<DeviceGroup[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [creating, setCreating] = useState(false)

  const [page, setPage] = useState(1)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listGroups()
      .then((data) => setGroups(data.items))
      .catch((err: unknown) => {
        setGroups(null)
        setError(err instanceof Error ? err.message : 'failed to load groups')
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const total = groups?.length ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    if (!groups) return []
    const start = (page - 1) * PAGE_SIZE
    return groups.slice(start, start + PAGE_SIZE)
  }, [groups, page])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) {
      setError('name is required')
      return
    }

    setCreating(true)
    setError(null)

    const body: CreateGroupInput = { name: trimmed }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      await createGroup(body)
      setName('')
      setDescription('')
      setPage(1)
      load()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'failed to create group')
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Inventory</p>
          <h1 className="title">Groups</h1>
          <p className="lede">
            Device groups for inventory and later policy assignment.
            {groups ? (
              <span className="page-meta">
                {' '}
                · {groups.length} group{groups.length === 1 ? '' : 's'}
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

      <section className="panel" aria-labelledby="create-group-heading">
        <h2 id="create-group-heading" className="panel-title">
          Create group
        </h2>
        <form className="token-create-grid" onSubmit={onCreate}>
          <div className="field">
            <label className="label" htmlFor="group-name">
              Name
            </label>
            <input
              id="group-name"
              className="input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="IT"
              required
            />
          </div>
          <div className="field" style={{ gridColumn: 'span 2' }}>
            <label className="label" htmlFor="group-desc">
              Description (optional)
            </label>
            <input
              id="group-desc"
              className="input"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Endpoints for the IT department"
            />
          </div>
          <div className="token-create-submit">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={creating}
            >
              {creating ? 'Creating…' : 'Create group'}
            </button>
          </div>
        </form>
      </section>

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
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !groups ? (
          <p className="loading">Loading groups…</p>
        ) : null}

        {!loading && groups && groups.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No groups yet</p>
            <p className="muted">Create groups such as IT or Finance above.</p>
          </div>
        ) : null}

        {groups && groups.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Description</th>
                    <th>Created</th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((g) => (
                    <tr key={g.id}>
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
                  onClick={() =>
                    setPage((p) => Math.min(pageCount, p + 1))
                  }
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
