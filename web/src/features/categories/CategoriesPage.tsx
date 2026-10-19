import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from '../../components/toast'
import { formatRelativeTime } from '../../lib/format'
import { createCategory, listCategories } from './api'
import type { Category, CreateCategoryInput } from './types'

const PAGE_SIZE = 10

export default function CategoriesPage() {
  const navigate = useNavigate()
  const [categories, setCategories] = useState<Category[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [creating, setCreating] = useState(false)
  const [page, setPage] = useState(1)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listCategories()
      .then((data) => setCategories(data.items))
      .catch((err: unknown) => {
        setCategories(null)
        setError(
          err instanceof Error ? err.message : 'failed to load categories',
        )
      })
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const total = categories?.length ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    if (!categories) return []
    const start = (page - 1) * PAGE_SIZE
    return categories.slice(start, start + PAGE_SIZE)
  }, [categories, page])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    const trimmedCode = code.trim().toLowerCase()
    const trimmedName = name.trim()
    if (!trimmedCode || !trimmedName) {
      setError('code and name are required')
      return
    }

    setCreating(true)
    setError(null)
    const body: CreateCategoryInput = {
      code: trimmedCode,
      name: trimmedName,
    }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      const created = await createCategory(body)
      setCode('')
      setName('')
      setDescription('')
      setPage(1)
      toast.success('Category created')
      load()
      navigate(`/categories/${created.code}`)
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create category'
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
          <p className="section-label">Control</p>
          <h1 className="title">Categories</h1>
          <p className="lede">
            Curated business labels expanded into technical signatures for
            policy rules.
            {categories ? (
              <span className="page-meta">
                {' '}
                · {categories.length} categor
                {categories.length === 1 ? 'y' : 'ies'}
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

      <section className="panel" aria-labelledby="create-category-heading">
        <h2 id="create-category-heading" className="panel-title">
          Create category
        </h2>
        <form className="rule-form-grid" onSubmit={onCreate}>
          <div className="field">
            <label className="label" htmlFor="category-code">
              Code
            </label>
            <input
              id="category-code"
              className="input mono"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="messaging"
              pattern="[a-zA-Z0-9_]+"
              title="Letters, digits, and underscore only"
              required
            />
          </div>
          <div className="field">
            <label className="label" htmlFor="category-name">
              Name
            </label>
            <input
              id="category-name"
              className="input"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Messaging"
              required
            />
          </div>
          <div className="field rule-form-value">
            <label className="label" htmlFor="category-desc">
              Description (optional)
            </label>
            <input
              id="category-desc"
              className="input"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Consumer chat / messaging apps"
            />
          </div>
          <div className="rule-form-actions">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={creating}
            >
              {creating ? 'Creating…' : 'Create category'}
            </button>
          </div>
        </form>
      </section>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      <section className="list-panel" aria-labelledby="category-list-heading">
        <div className="list-panel-header">
          <h2 id="category-list-heading" className="panel-title">
            All categories
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !categories ? (
          <p className="loading">Loading categories…</p>
        ) : null}

        {!loading && categories && categories.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No categories yet</p>
            <p className="muted">Create a category above, then add signatures.</p>
          </div>
        ) : null}

        {categories && categories.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Code</th>
                    <th>Description</th>
                    <th>Updated</th>
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((c) => (
                    <tr
                      key={c.code}
                      className="table-row-link"
                      tabIndex={0}
                      onClick={() => navigate(`/categories/${c.code}`)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          navigate(`/categories/${c.code}`)
                        }
                      }}
                    >
                      <td>
                        <div className="cell-primary">{c.name}</div>
                      </td>
                      <td className="mono">{c.code}</td>
                      <td className="muted">{c.description || '—'}</td>
                      <td className="muted" title={c.updated_at || c.created_at}>
                        {c.updated_at || c.created_at
                          ? formatRelativeTime(c.updated_at || c.created_at || '')
                          : '—'}
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
