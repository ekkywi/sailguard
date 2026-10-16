import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { formatRelativeTime } from '../../lib/format'
import { toast } from '../../components/toast'
import { createPolicy, listPolicies } from './api'
import type { CreatePolicyInput, Policy } from './types'

const PAGE_SIZE = 10

export default function PoliciesPage() {
  const navigate = useNavigate()
  const [policies, setPolicies] = useState<Policy[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [mode, setMode] = useState('audit')
  const [creating, setCreating] = useState(false)
  const [page, setPage] = useState(1)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listPolicies()
      .then((data) => setPolicies(data.items))
      .catch((err: unknown) => {
        setPolicies(null)
        setError(err instanceof Error ? err.message : 'failed to load policies')
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const total = policies?.length ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    if (!policies) return []
    const start = (page - 1) * PAGE_SIZE
    return policies.slice(start, start + PAGE_SIZE)
  }, [policies, page])

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

    const body: CreatePolicyInput = {
      name: trimmed,
      mode: mode || 'audit',
    }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      await createPolicy(body)
      setName('')
      setDescription('')
      setMode('audit')
      setPage(1)
      toast.success('Policy created')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create policy'
      setError(msg)
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Policy</p>
          <h1 className="title">Policies</h1>
          <p className="lede">
            Application control policies (audit first, then enforce).
            {policies ? (
              <span className="page-meta">
                {' '}
                · {policies.length} polic
                {policies.length === 1 ? 'y' : 'ies'}
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

      <section className="panel" aria-labelledby="create-policy-heading">
        <h2 id="create-policy-heading" className="panel-title">
          Create policy
        </h2>
        <form className="token-create-grid" onSubmit={onCreate}>
          <div className="field">
            <label className="label" htmlFor="policy-name">
              Name
            </label>
            <input
              id="policy-name"
              className="input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Lab block notepad"
              required
            />
          </div>
          <div className="field">
            <label className="label" htmlFor="policy-mode">
              Mode
            </label>
            <select
              id="policy-mode"
              className="input"
              value={mode}
              onChange={(e) => setMode(e.target.value)}
            >
              <option value="audit">audit</option>
              <option value="enforce">enforce</option>
            </select>
          </div>
          <div className="field" style={{ gridColumn: 'span 2' }}>
            <label className="label" htmlFor="policy-desc">
              Description (optional)
            </label>
            <input
              id="policy-desc"
              className="input"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Demo policy for lab endpoints"
            />
          </div>
          <div className="token-create-submit">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={creating}
            >
              {creating ? 'Creating…' : 'Create policy'}
            </button>
          </div>
        </form>
      </section>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      <section className="list-panel" aria-labelledby="policy-list-heading">
        <div className="list-panel-header">
          <h2 id="policy-list-heading" className="panel-title">
            All policies
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !policies ? (
          <p className="loading">Loading policies…</p>
        ) : null}

        {!loading && policies && policies.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No policies yet</p>
            <p className="muted">
              Create an audit policy above, then open it to add rules.
            </p>
          </div>
        ) : null}

        {policies && policies.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Mode</th>
                    <th>Version</th>
                    <th>Published</th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((p) => (
                    <tr
                      key={p.id}
                      className="table-row-link"
                      tabIndex={0}
                      onClick={() => navigate(`/policies/${p.id}`)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          navigate(`/policies/${p.id}`)
                        }
                      }}
                    >
                      <td>
                        <div className="cell-primary">{p.name}</div>
                        <div className="muted">{p.description || '—'}</div>
                      </td>
                      <td>
                        <span className="mono">{p.mode}</span>
                      </td>
                      <td className="muted">{p.version}</td>
                      <td
                        className="muted"
                        title={p.published_at ?? undefined}
                      >
                        {p.published_at
                          ? formatRelativeTime(p.published_at)
                          : 'Draft'}
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
                  onClick={() => setPage((p) => Math.min(pageCount, p + 1))}
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
