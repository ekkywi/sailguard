import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { PermissionDenied } from '../../components/PermissionDenied'
import { SortableTh } from '../../components/SortableTh'
import { TableSkeleton } from '../../components/TableSkeleton'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import {
  formatDateTime,
  formatRelativeTime,
  statusBadgeClass,
} from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import {
  compareDate,
  compareNumber,
  compareString,
  useClientList,
  type ClientListCompare,
} from '../../lib/useClientList'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import {
  createEnrollmentToken,
  listEnrollmentTokens,
  revokeEnrollmentToken,
} from './api'
import type { CreateEnrollmentTokenInput, EnrollmentToken } from './types'

const PAGE_SIZE = 10

type StatusFilter = '' | 'active' | 'expired' | 'revoked'

function tokenStatus(t: EnrollmentToken): 'active' | 'expired' | 'revoked' {
  if (t.revoked_at) return 'revoked'
  if (t.expires_at && new Date(t.expires_at).getTime() <= Date.now()) {
    return 'expired'
  }
  return 'active'
}

function matchesToken(t: EnrollmentToken, q: string): boolean {
  return (
    (t.label || '').toLowerCase().includes(q) ||
    t.id.toLowerCase().includes(q) ||
    tokenStatus(t).includes(q)
  )
}

const TOKEN_COMPARE: ClientListCompare<EnrollmentToken> = {
  label: (a, b) => compareString(a.label || '', b.label || ''),
  uses: (a, b) => compareNumber(a.use_count, b.use_count),
  expires_at: (a, b) => compareDate(a.expires_at, b.expires_at),
  status: (a, b) => compareString(tokenStatus(a), tokenStatus(b)),
  created_at: (a, b) => compareDate(a.created_at, b.created_at),
}

const STATUS_CHIPS: { id: StatusFilter; label: string }[] = [
  { id: '', label: 'All' },
  { id: 'active', label: 'Active' },
  { id: 'expired', label: 'Expired' },
  { id: 'revoked', label: 'Revoked' },
]

export default function TokensPage() {
  useDocumentTitle('Enrollment tokens')
  const { me } = useOutletContext<AppOutletContext>()
  const canWrite = hasPermission(me, 'device.write')
  const [tokens, setTokens] = useState<EnrollmentToken[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')

  const [createOpen, setCreateOpen] = useState(false)
  const [label, setLabel] = useState('')
  const [maxUses, setMaxUses] = useState(1)
  const [expiresHours, setExpiresHours] = useState('')
  const [creating, setCreating] = useState(false)
  const [labelError, setLabelError] = useState<string | null>(null)
  const createTitleId = useId()
  const createFirstRef = useRef<HTMLInputElement>(null)

  const [freshSecret, setFreshSecret] = useState<string | null>(null)
  const [copyDone, setCopyDone] = useState(false)

  const [revokeTarget, setRevokeTarget] = useState<EnrollmentToken | null>(null)
  const [revoking, setRevoking] = useState(false)

  const statusFilterFn = useMemo(() => {
    if (!statusFilter) return undefined
    return (t: EnrollmentToken) => tokenStatus(t) === statusFilter
  }, [statusFilter])

  const list = useClientList({
    items: tokens,
    pageSize: PAGE_SIZE,
    matches: matchesToken,
    compare: TOKEN_COMPARE,
    defaultSortKey: 'created_at',
    defaultSortDir: 'desc',
    filter: statusFilterFn,
  })

  const load = useCallback(() => {
    if (!canWrite) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    listEnrollmentTokens()
      .then((data) => setTokens(data.items))
      .catch((err: unknown) => {
        setTokens(null)
        setError(err instanceof Error ? err.message : 'failed to load tokens')
      })
      .finally(() => setLoading(false))
  }, [canWrite])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!createOpen) return
    createFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !creating) {
        setCreateOpen(false)
        setLabel('')
        setMaxUses(1)
        setExpiresHours('')
        setLabelError(null)
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
    setLabel('')
    setMaxUses(1)
    setExpiresHours('')
    setLabelError(null)
    setError(null)
    setCreateOpen(true)
  }

  function closeCreate() {
    if (creating) return
    setCreateOpen(false)
    setLabel('')
    setMaxUses(1)
    setExpiresHours('')
    setLabelError(null)
  }

  function clearFilters() {
    list.clearSearch()
    setStatusFilter('')
    list.setPage(1)
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    const trimmed = label.trim()
    if (!trimmed) {
      setLabelError('Label is required')
      return
    }

    setCreating(true)
    setError(null)
    setCopyDone(false)
    setLabelError(null)

    const body: CreateEnrollmentTokenInput = {
      label: trimmed,
      max_uses: maxUses > 0 ? maxUses : 1,
    }

    const hours = Number(expiresHours)
    if (expiresHours.trim() !== '' && Number.isFinite(hours) && hours > 0) {
      body.expires_in_hours = hours
    }

    try {
      const created = await createEnrollmentToken(body)
      setFreshSecret(created.token ?? null)
      setCreateOpen(false)
      setLabel('')
      setMaxUses(1)
      setExpiresHours('')
      list.setPage(1)
      toast.success('Token created', 'Copy the secret below — it is shown once.')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create token'
      setError(msg)
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  async function confirmRevoke() {
    if (!revokeTarget) return
    setRevoking(true)
    setError(null)
    try {
      await revokeEnrollmentToken(revokeTarget.id)
      setRevokeTarget(null)
      toast.success('Token revoked')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to revoke token'
      setError(msg)
      toast.error(msg)
      setRevokeTarget(null)
    } finally {
      setRevoking(false)
    }
  }

  async function onCopySecret() {
    if (!freshSecret) return
    try {
      await navigator.clipboard.writeText(freshSecret)
      setCopyDone(true)
      toast.success('Copied to clipboard')
    } catch {
      const msg = 'could not copy to clipboard'
      setError(msg)
      toast.error(msg)
    }
  }

  if (!canWrite) {
    return (
      <PermissionDenied permission="device.write" title="Enrollment tokens" />
    )
  }

  const hasSource = (tokens?.length ?? 0) > 0
  const showNoMatches = hasSource && list.total === 0
  const hasFilters = list.hasActiveQuery || Boolean(statusFilter)

  return (
    <div className="stack tokens-layout">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Inventory</p>
          <h1 className="title">Enrollment tokens</h1>
          <p className="lede">
            Enrollment tokens
            {tokens ? (
              <span className="page-meta">
                {' '}
                · {tokens.length} token{tokens.length === 1 ? '' : 's'}
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
          <button
            type="button"
            className="btn btn-primary"
            onClick={openCreate}
          >
            Create token
          </button>
        </div>
      </header>

      {freshSecret ? (
        <div className="empty-state" role="status">
          <p className="empty-title">Save this token now</p>
          <p className="muted">
            It will not be shown again. Use it with{' '}
            <span className="mono">POST /v1/agent/enroll</span>.
          </p>
          <p className="mono secret-value">{freshSecret}</p>
          <div className="secret-actions">
            <button
              type="button"
              className="btn btn-primary"
              onClick={onCopySecret}
            >
              {copyDone ? 'Copied' : 'Copy'}
            </button>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => {
                setFreshSecret(null)
                setCopyDone(false)
              }}
            >
              Dismiss
            </button>
          </div>
        </div>
      ) : null}

      <div className="audit-toolbar" role="search">
        <div className="audit-chips" aria-label="Token status">
          {STATUS_CHIPS.map((c) => (
            <button
              key={c.id || 'all-status'}
              type="button"
              className={
                statusFilter === c.id ? 'audit-chip audit-chip-on' : 'audit-chip'
              }
              onClick={() => {
                setStatusFilter(c.id)
                list.setPage(1)
              }}
            >
              {c.label}
            </button>
          ))}
        </div>
        <div className="audit-filters">
          <div className="field audit-filter-grow">
            <label className="label" htmlFor="tokens-q">
              Search
            </label>
            <input
              id="tokens-q"
              className="input"
              value={list.queryDraft}
              onChange={(e) => list.setQueryDraft(e.target.value)}
              placeholder="Label or id…"
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

      <section className="list-panel" aria-labelledby="token-list-heading">
        <div className="list-panel-header">
          <h2 id="token-list-heading" className="panel-title">
            All tokens
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

        {loading && !tokens ? (
          <TableSkeleton label="Loading tokens…" />
        ) : null}

        {!loading && tokens && tokens.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No enrollment tokens yet</p>
            <p className="muted">Create one, then open Download agent.</p>
            <div className="empty-actions">
              <button
                type="button"
                className="btn btn-primary"
                onClick={openCreate}
              >
                Create token
              </button>
              <Link className="btn btn-ghost" to="/agent-download">
                Download agent
              </Link>
            </div>
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
                      label="Label"
                      columnKey="label"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Uses"
                      columnKey="uses"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Expires"
                      columnKey="expires_at"
                      sortKey={list.sortKey}
                      sortDir={list.sortDir}
                      onSort={list.toggleSort}
                    />
                    <SortableTh
                      label="Status"
                      columnKey="status"
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
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {list.pageItems.map((t) => {
                    const status = tokenStatus(t)
                    return (
                      <tr key={t.id}>
                        <td>
                          <div className="cell-primary">{t.label || '—'}</div>
                          <div className="muted mono">{t.id}</div>
                        </td>
                        <td className="muted">
                          {t.use_count} / {t.max_uses}
                        </td>
                        <td
                          className="muted"
                          title={t.expires_at ?? undefined}
                        >
                          {t.expires_at
                            ? formatDateTime(t.expires_at)
                            : 'Never'}
                        </td>
                        <td>
                          <span className={statusBadgeClass(status)}>
                            {status}
                          </span>
                        </td>
                        <td className="muted" title={t.created_at}>
                          {formatRelativeTime(t.created_at)}
                        </td>
                        <td>
                          {status === 'active' ? (
                            <button
                              type="button"
                              className="btn btn-ghost"
                              onClick={() => setRevokeTarget(t)}
                            >
                              Revoke
                            </button>
                          ) : (
                            <span className="muted">—</span>
                          )}
                        </td>
                      </tr>
                    )
                  })}
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
              Create token
            </h2>
            <p className="dialog-desc muted">
              The plaintext secret is shown once after create.
            </p>
            <form className="users-dialog-form" onSubmit={onCreate}>
              <div className="field">
                <label className="label" htmlFor="token-label">
                  Label
                </label>
                <input
                  ref={createFirstRef}
                  id="token-label"
                  className={labelError ? 'input input-invalid' : 'input'}
                  value={label}
                  onChange={(e) => {
                    setLabel(e.target.value)
                    setLabelError(null)
                  }}
                  placeholder="lab-batch-1"
                  required
                  disabled={creating}
                  aria-invalid={labelError ? true : undefined}
                />
                {labelError ? (
                  <p className="field-error" role="alert">
                    {labelError}
                  </p>
                ) : null}
              </div>
              <div className="field">
                <label className="label" htmlFor="token-max">
                  Max uses
                </label>
                <input
                  id="token-max"
                  className="input"
                  type="number"
                  min={1}
                  value={maxUses}
                  onChange={(e) => setMaxUses(Number(e.target.value) || 1)}
                  disabled={creating}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="token-exp">
                  Expires (hours)
                </label>
                <input
                  id="token-exp"
                  className="input"
                  type="number"
                  min={1}
                  value={expiresHours}
                  onChange={(e) => setExpiresHours(e.target.value)}
                  placeholder="Optional"
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
                  {creating ? 'Creating…' : 'Create token'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={revokeTarget !== null}
        title="Revoke enrollment token?"
        description={
          revokeTarget
            ? `“${revokeTarget.label || revokeTarget.id}” will stop working immediately and cannot be used to enroll new agents.`
            : ''
        }
        confirmLabel="Revoke"
        cancelLabel="Cancel"
        danger
        busy={revoking}
        onConfirm={confirmRevoke}
        onCancel={() => {
          if (!revoking) setRevokeTarget(null)
        }}
      />
    </div>
  )
}
