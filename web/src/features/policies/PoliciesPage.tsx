import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useMemo,
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
  compareNumber,
  compareString,
  useClientList,
  type ClientListCompare,
} from '../../lib/useClientList'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { createPolicy, listPolicies } from './api'
import type { CreatePolicyInput, Policy } from './types'

const PAGE_SIZE = 10

type ModeFilter = '' | 'audit' | 'enforce'

function matchesPolicy(p: Policy, q: string): boolean {
  return (
    p.name.toLowerCase().includes(q) ||
    (p.description || '').toLowerCase().includes(q) ||
    p.mode.toLowerCase().includes(q) ||
    p.id.toLowerCase().includes(q)
  )
}

const POLICY_COMPARE: ClientListCompare<Policy> = {
  name: (a, b) => compareString(a.name, b.name),
  mode: (a, b) => compareString(a.mode, b.mode),
  version: (a, b) => compareNumber(a.version, b.version),
  published_at: (a, b) => compareDate(a.published_at, b.published_at),
}

const MODE_CHIPS: { id: ModeFilter; label: string }[] = [
  { id: '', label: 'All modes' },
  { id: 'audit', label: 'Audit' },
  { id: 'enforce', label: 'Enforce' },
]

export default function PoliciesPage() {
  useDocumentTitle('Policies')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'policy.read')
  const canWrite = hasPermission(me, 'policy.write')
  const navigate = useNavigate()
  const [policies, setPolicies] = useState<Policy[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [modeFilter, setModeFilter] = useState<ModeFilter>('')

  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [mode, setMode] = useState('audit')
  const [creating, setCreating] = useState(false)
  const [nameError, setNameError] = useState<string | null>(null)
  const createTitleId = useId()
  const createFirstRef = useRef<HTMLInputElement>(null)

  const modeFilterFn = useMemo(() => {
    if (!modeFilter) return undefined
    return (p: Policy) => p.mode === modeFilter
  }, [modeFilter])

  const list = useClientList({
    items: policies,
    pageSize: PAGE_SIZE,
    matches: matchesPolicy,
    compare: POLICY_COMPARE,
    defaultSortKey: 'name',
    filter: modeFilterFn,
  })

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listPolicies()
      .then((data) => setPolicies(data.items))
      .catch((err: unknown) => {
        setPolicies(null)
        setError(err instanceof Error ? err.message : 'failed to load policies')
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
        setName('')
        setDescription('')
        setMode('audit')
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

  function openCreate() {
    setName('')
    setDescription('')
    setMode('audit')
    setNameError(null)
    setError(null)
    setCreateOpen(true)
  }

  function closeCreate() {
    if (creating) return
    setCreateOpen(false)
    setName('')
    setDescription('')
    setMode('audit')
    setNameError(null)
  }

  function clearFilters() {
    list.clearSearch()
    setModeFilter('')
    list.setPage(1)
  }

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

    const body: CreatePolicyInput = {
      name: trimmed,
      mode: mode || 'audit',
    }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      await createPolicy(body)
      setCreateOpen(false)
      setName('')
      setDescription('')
      setMode('audit')
      list.setPage(1)
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

  if (!canRead) {
    return <PermissionDenied permission="policy.read" title="Policies" />
  }

  const hasSource = (policies?.length ?? 0) > 0
  const showNoMatches = hasSource && list.total === 0
  const hasFilters = list.hasActiveQuery || Boolean(modeFilter)

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Policy</p>
          <h1 className="title">Policies</h1>
          <p className="lede">
            Application control policies
            {policies ? (
              <span className="page-meta">
                {' '}
                · {policies.length} polic
                {policies.length === 1 ? 'y' : 'ies'}
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
              Create policy
            </button>
          ) : null}
        </div>
      </header>

      <div className="audit-toolbar" role="search">
        <div className="audit-chips" aria-label="Policy mode">
          {MODE_CHIPS.map((c) => (
            <button
              key={c.id || 'all-modes'}
              type="button"
              className={
                modeFilter === c.id ? 'audit-chip audit-chip-on' : 'audit-chip'
              }
              onClick={() => {
                setModeFilter(c.id)
                list.setPage(1)
              }}
            >
              {c.label}
            </button>
          ))}
        </div>
        <div className="audit-filters">
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="policies-q">
              Search
            </label>
            <input
              id="policies-q"
              className="input"
              value={list.queryDraft}
              onChange={(e) => list.setQueryDraft(e.target.value)}
              placeholder="Name, description, or id…"
            />
          </div>
          {hasFilters ? (
            <div className="field audit-filter-actions">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={clearFilters}
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

      <section className="list-panel" aria-labelledby="policy-list-heading">
        <div className="list-panel-header">
          <h2 id="policy-list-heading" className="panel-title">
            All policies
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

        {loading && !policies ? (
          <TableSkeleton label="Loading policies…" />
        ) : null}

        {!loading && policies && policies.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No policies yet</p>
            <p className="muted">
              {canWrite
                ? 'Create an audit policy, then open it to add rules.'
                : 'Ask an admin with policy.write to create a policy.'}
            </p>
            {canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openCreate}
                >
                  Create policy
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
                onClick={clearFilters}
              >
                Clear filters
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
                      label="Mode"
                      columnKey="mode"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Version"
                      columnKey="version"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Published"
                      columnKey="published_at"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                  </tr>
                </thead>
                <tbody>
                  {list.pageItems.map((p) => (
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
                        <span
                          className={
                            p.mode === 'enforce'
                              ? 'badge-mode badge-mode-enforce'
                              : 'badge-mode'
                          }
                        >
                          {p.mode}
                        </span>
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
              Create policy
            </h2>
            <p className="dialog-desc muted">
              Start in audit mode, then add rules and publish when ready.
            </p>
            <form className="users-dialog-form" onSubmit={onCreate}>
              <div className="field">
                <label className="label" htmlFor="policy-name">
                  Name
                </label>
                <input
                  ref={createFirstRef}
                  id="policy-name"
                  className={nameError ? 'input input-invalid' : 'input'}
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    setNameError(null)
                  }}
                  placeholder="Lab block notepad"
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
                <label className="label" htmlFor="policy-mode">
                  Mode
                </label>
                <select
                  id="policy-mode"
                  className="input"
                  value={mode}
                  onChange={(e) => setMode(e.target.value)}
                  disabled={creating}
                >
                  <option value="audit">audit</option>
                  <option value="enforce">enforce</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="policy-desc">
                  Description (optional)
                </label>
                <input
                  id="policy-desc"
                  className="input"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Demo policy for lab endpoints"
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
                  {creating ? 'Creating…' : 'Create policy'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}
