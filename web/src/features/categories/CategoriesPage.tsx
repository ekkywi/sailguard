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
import { createCategory, listCategories } from './api'
import type { Category, CreateCategoryInput } from './types'

const PAGE_SIZE = 10

function matchesCategory(c: Category, q: string): boolean {
  return (
    c.name.toLowerCase().includes(q) ||
    c.code.toLowerCase().includes(q) ||
    (c.description || '').toLowerCase().includes(q)
  )
}

const CATEGORY_COMPARE: ClientListCompare<Category> = {
  name: (a, b) => compareString(a.name, b.name),
  code: (a, b) => compareString(a.code, b.code),
  description: (a, b) =>
    compareString(a.description || '', b.description || ''),
  updated_at: (a, b) =>
    compareDate(a.updated_at || a.created_at, b.updated_at || b.created_at),
}

export default function CategoriesPage() {
  useDocumentTitle('Categories')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'policy.read')
  const canWrite = hasPermission(me, 'policy.write')
  const navigate = useNavigate()
  const [categories, setCategories] = useState<Category[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [createOpen, setCreateOpen] = useState(false)
  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const createTitleId = useId()
  const createFirstRef = useRef<HTMLInputElement>(null)

  const list = useClientList({
    items: categories,
    pageSize: PAGE_SIZE,
    matches: matchesCategory,
    compare: CATEGORY_COMPARE,
    defaultSortKey: 'name',
  })

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
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
  }, [canRead])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!createOpen) return
    createFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !creating) {
        setCreateOpen(false)
        setCode('')
        setName('')
        setDescription('')
        setFormError(null)
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

  function openCreate() {
    setCode('')
    setName('')
    setDescription('')
    setFormError(null)
    setError(null)
    setCreateOpen(true)
  }

  function closeCreate() {
    if (creating) return
    setCreateOpen(false)
    setCode('')
    setName('')
    setDescription('')
    setFormError(null)
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    const trimmedCode = code.trim().toLowerCase()
    const trimmedName = name.trim()
    if (!trimmedCode || !trimmedName) {
      setFormError('Code and name are required')
      return
    }

    setCreating(true)
    setError(null)
    setFormError(null)
    const body: CreateCategoryInput = {
      code: trimmedCode,
      name: trimmedName,
    }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      const created = await createCategory(body)
      setCreateOpen(false)
      setCode('')
      setName('')
      setDescription('')
      list.setPage(1)
      toast.success('Category created')
      load()
      navigate(`/categories/${created.code}`)
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create category'
      setFormError(msg)
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="policy.read" title="Categories" />
  }

  const hasSource = (categories?.length ?? 0) > 0
  const showNoMatches = hasSource && list.total === 0

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Control</p>
          <h1 className="title">Categories</h1>
          <p className="lede">
            Category catalog for policy rules
            {categories ? (
              <span className="page-meta">
                {' '}
                · {categories.length} categor
                {categories.length === 1 ? 'y' : 'ies'}
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
              Create category
            </button>
          ) : null}
        </div>
      </header>

      <div className="audit-toolbar" role="search">
        <div className="audit-filters">
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="categories-q">
              Search
            </label>
            <input
              id="categories-q"
              className="input"
              value={list.queryDraft}
              onChange={(e) => list.setQueryDraft(e.target.value)}
              placeholder="Name, code, or description…"
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

      <section className="list-panel" aria-labelledby="category-list-heading">
        <div className="list-panel-header">
          <h2 id="category-list-heading" className="panel-title">
            All categories
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

        {loading && !categories ? (
          <TableSkeleton label="Loading categories…" />
        ) : null}

        {!loading && categories && categories.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No categories yet</p>
            <p className="muted">
              {canWrite
                ? 'Create a category, then add signatures.'
                : 'No categories in the catalog.'}
            </p>
            {canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openCreate}
                >
                  Create category
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
                      label="Code"
                      columnKey="code"
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
                      label="Updated"
                      columnKey="updated_at"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                  </tr>
                </thead>
                <tbody>
                  {list.pageItems.map((c) => (
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
                      <td
                        className="muted"
                        title={c.updated_at || c.created_at}
                      >
                        {c.updated_at || c.created_at
                          ? formatRelativeTime(
                              c.updated_at || c.created_at || '',
                            )
                          : '—'}
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
              Create category
            </h2>
            <p className="dialog-desc muted">
              Codes are stable identifiers used in policy rules.
            </p>
            {formError ? (
              <p className="error" role="alert">
                {formError}
              </p>
            ) : null}
            <form className="users-dialog-form" onSubmit={onCreate}>
              <div className="field">
                <label className="label" htmlFor="category-code">
                  Code
                </label>
                <input
                  ref={createFirstRef}
                  id="category-code"
                  className="input mono"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  placeholder="messaging"
                  pattern="[a-zA-Z0-9_]+"
                  title="Letters, digits, and underscore only"
                  required
                  disabled={creating}
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
                  disabled={creating}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="category-desc">
                  Description (optional)
                </label>
                <input
                  id="category-desc"
                  className="input"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Consumer chat / messaging apps"
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
                  {creating ? 'Creating…' : 'Create category'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}
