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
import { createNetworkPolicy, listNetworkPolicies } from './api'
import type { CreateNetworkPolicyInput, NetworkPolicy } from './types'

const PAGE_SIZE = 10

type ModeFilter = '' | 'audit' | 'enforce'

function matchesPolicy(p: NetworkPolicy, q: string): boolean {
  return (
    p.name.toLowerCase().includes(q) ||
    (p.description || '').toLowerCase().includes(q) ||
    p.mode.toLowerCase().includes(q) ||
    p.id.toLowerCase().includes(q)
  )
}

const POLICY_COMPARE: ClientListCompare<NetworkPolicy> = {
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

export default function NetworkPoliciesPage() {
  useDocumentTitle('Network policies')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'network.read')
  const canWrite = hasPermission(me, 'network.write')
  const navigate = useNavigate()
  const [policies, setPolicies] = useState<NetworkPolicy[] | null>(null)
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
    return (p: NetworkPolicy) => p.mode === modeFilter
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
    listNetworkPolicies()
      .then((data) => setPolicies(data.items))
      .catch((err: unknown) => {
        setPolicies(null)
        setError(
          err instanceof Error ? err.message : 'failed to load network policies',
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

    const body: CreateNetworkPolicyInput = {
      name: trimmed,
      mode: mode || 'audit',
    }
    const desc = description.trim()
    if (desc) body.description = desc

    try {
      const created = await createNetworkPolicy(body)
      setCreateOpen(false)
      setName('')
      setDescription('')
      setMode('audit')
      toast.success('Network policy created')
      navigate(`/network-policies/${created.id}`)
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create network policy'
      setError(msg)
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  if (!canRead) {
    return (
      <PermissionDenied permission="network.read" title="Network policies" />
    )
  }

  const hasSource = (policies?.length ?? 0) > 0
  const showNoMatches = hasSource && list.total === 0
  const hasFilters = list.hasActiveQuery || Boolean(modeFilter)

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Control</p>
          <h1 className="title">Network policies</h1>
          <p className="lede">
            Deny egress destinations (host, IP, CIDR)
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
            <label className="label" htmlFor="net-policies-q">
              Search
            </label>
            <input
              id="net-policies-q"
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

      <section className="list-panel" aria-labelledby="net-policy-list-heading">
        <div className="list-panel-header">
          <h2 id="net-policy-list-heading" className="panel-title">
            All network policies
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
          <TableSkeleton label="Loading network policies…" />
        ) : null}

        {!loading && policies && policies.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No network policies yet</p>
            <p className="muted">
              {canWrite
                ? 'Create a policy, add host or IP deny rules, assign, then publish.'
                : 'Ask an admin with network.write to create a policy.'}
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
                      onClick={() => navigate(`/network-policies/${p.id}`)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          navigate(`/network-policies/${p.id}`)
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
              Create network policy
            </h2>
            <p className="dialog-desc muted">
              Start in audit, add destination rules, assign devices, then
              publish to enforce.
            </p>
            <form className="users-dialog-form" onSubmit={onCreate}>
              <div className="field">
                <label className="label" htmlFor="net-policy-name">
                  Name
                </label>
                <input
                  ref={createFirstRef}
                  id="net-policy-name"
                  className={nameError ? 'input input-invalid' : 'input'}
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    setNameError(null)
                  }}
                  placeholder="Block Windows activation hosts"
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
                <label className="label" htmlFor="net-policy-mode">
                  Mode
                </label>
                <select
                  id="net-policy-mode"
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
                <label className="label" htmlFor="net-policy-desc">
                  Description (optional)
                </label>
                <input
                  id="net-policy-desc"
                  className="input"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Phone-home / activation destinations"
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
