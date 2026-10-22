import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useOutletContext } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { TableSkeleton } from '../../components/TableSkeleton'
import { PermissionDenied } from '../../components/PermissionDenied'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import {
  formatDateTime,
  formatRelativeTime,
  statusBadgeClass,
} from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import {
  createEnrollmentToken,
  listEnrollmentTokens,
  revokeEnrollmentToken,
} from './api'
import type { CreateEnrollmentTokenInput, EnrollmentToken } from './types'

const PAGE_SIZE = 10

function tokenStatus(t: EnrollmentToken): 'active' | 'expired' | 'revoked' {
  if (t.revoked_at) return 'revoked'
  if (t.expires_at && new Date(t.expires_at).getTime() <= Date.now()) {
    return 'expired'
  }
  return 'active'
}

export default function TokensPage() {
  useDocumentTitle('Enrollment tokens')
  const { me } = useOutletContext<AppOutletContext>()
  const canWrite = hasPermission(me, 'device.write')
  const [tokens, setTokens] = useState<EnrollmentToken[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [label, setLabel] = useState('')
  const [maxUses, setMaxUses] = useState(1)
  const [expiresHours, setExpiresHours] = useState('')
  const [creating, setCreating] = useState(false)

  const [freshSecret, setFreshSecret] = useState<string | null>(null)
  const [copyDone, setCopyDone] = useState(false)

  const [revokeTarget, setRevokeTarget] = useState<EnrollmentToken | null>(null)
  const [revoking, setRevoking] = useState(false)

  const [page, setPage] = useState(1)

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

  const total = tokens?.length ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    if (!tokens) return []
    const start = (page - 1) * PAGE_SIZE
    return tokens.slice(start, start + PAGE_SIZE)
  }, [tokens, page])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  if (!canWrite) {
    return (
      <PermissionDenied permission="device.write" title="Enrollment tokens" />
    )
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    const trimmed = label.trim()
    if (!trimmed) {
      setError('label is required')
      return
    }

    setCreating(true)
    setError(null)
    setCopyDone(false)

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
      setLabel('')
      setMaxUses(1)
      setExpiresHours('')
      setPage(1)
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
        <button
          type="button"
          className="btn btn-ghost"
          onClick={load}
          disabled={loading}
        >
          Refresh
        </button>
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

      <section className="panel" aria-labelledby="create-token-heading">
        <h2 id="create-token-heading" className="panel-title">
          Create token
        </h2>
        <form className="token-create-grid" onSubmit={onCreate}>
          <div className="field">
            <label className="label" htmlFor="token-label">
              Label
            </label>
            <input
              id="token-label"
              className="input"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="lab-batch-1"
              required
            />
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
            />
          </div>
          <div className="token-create-submit">
            <button
              type="submit"
              className="btn btn-primary"
              disabled={creating}
            >
              {creating ? 'Creating…' : 'Create token'}
            </button>
          </div>
        </form>
      </section>

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
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
            </p>
          ) : null}
        </div>

        {loading && !tokens ? (
          <TableSkeleton label="Loading tokens…" />
        ) : null}

        {!loading && tokens && tokens.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No enrollment tokens yet</p>
            <p className="muted">Create one above, then open Download agent.</p>
            <div className="empty-actions">
              <Link className="btn btn-ghost" to="/agent-download">
                Download agent
              </Link>
            </div>
          </div>
        ) : null}

        {tokens && tokens.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Label</th>
                    <th>Uses</th>
                    <th>Expires</th>
                    <th>Status</th>
                    <th>Created</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((t) => {
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
