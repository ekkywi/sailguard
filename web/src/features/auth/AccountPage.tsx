import { FormEvent, useCallback, useEffect, useId, useRef, useState } from 'react'
import { useOutletContext, useSearchParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import {
  formatDateTime,
  formatRelativeTime,
} from '../../lib/format'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import {
  changePassword,
  listSessions,
  revokeOtherSessions,
  revokeSession,
} from './api'
import type { AuthSession } from './types'

type AccountTab = 'profile' | 'sessions'

function parseTab(raw: string | null): AccountTab {
  return raw === 'sessions' ? 'sessions' : 'profile'
}

export default function AccountPage() {
  useDocumentTitle('Account')
  const { me, meLoading } = useOutletContext<AppOutletContext>()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = parseTab(searchParams.get('tab'))

  const [passwordOpen, setPasswordOpen] = useState(false)
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [savingPassword, setSavingPassword] = useState(false)
  const [passwordError, setPasswordError] = useState<string | null>(null)

  const [sessions, setSessions] = useState<AuthSession[] | null>(null)
  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [sessionsError, setSessionsError] = useState<string | null>(null)
  const [revokingId, setRevokingId] = useState<string | null>(null)
  const [revokingOthers, setRevokingOthers] = useState(false)
  const [revokeTarget, setRevokeTarget] = useState<AuthSession | null>(null)
  const [confirmRevokeOthers, setConfirmRevokeOthers] = useState(false)

  const passwordTitleId = useId()
  const passwordFirstRef = useRef<HTMLInputElement>(null)

  const loadSessions = useCallback(() => {
    setSessionsLoading(true)
    setSessionsError(null)
    listSessions()
      .then((data) => setSessions(data.items ?? []))
      .catch((err: unknown) => {
        setSessions(null)
        setSessionsError(
          err instanceof Error ? err.message : 'failed to load sessions',
        )
      })
      .finally(() => setSessionsLoading(false))
  }, [])

  useEffect(() => {
    if (tab !== 'sessions') return
    loadSessions()
  }, [tab, loadSessions])

  useEffect(() => {
    if (!passwordOpen) return
    passwordFirstRef.current?.focus()

    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !savingPassword) closePassword()
    }

    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [passwordOpen, savingPassword])

  function setTab(next: AccountTab) {
    if (next === 'profile') {
      setSearchParams({}, { replace: true })
      return
    }
    setSearchParams({ tab: next }, { replace: true })
  }

  function resetPasswordForm() {
    setCurrentPassword('')
    setNewPassword('')
    setConfirmPassword('')
    setPasswordError(null)
  }

  function openPassword() {
    resetPasswordForm()
    setPasswordOpen(true)
  }

  function closePassword() {
    if (savingPassword) return
    setPasswordOpen(false)
    resetPasswordForm()
  }

  async function onChangePassword(e: FormEvent) {
    e.preventDefault()
    setPasswordError(null)
    if (newPassword.length < 10) {
      setPasswordError('New password must be at least 10 characters')
      return
    }
    if (newPassword !== confirmPassword) {
      setPasswordError('New password and confirmation do not match')
      return
    }
    if (newPassword === currentPassword) {
      setPasswordError('New password must differ from current password')
      return
    }

    setSavingPassword(true)
    try {
      await changePassword(currentPassword, newPassword)
      toast.success('Password updated')
      setPasswordOpen(false)
      resetPasswordForm()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to change password'
      setPasswordError(msg)
      toast.error(msg)
    } finally {
      setSavingPassword(false)
    }
  }

  async function onConfirmRevoke() {
    if (!revokeTarget) return
    const id = revokeTarget.id
    setRevokingId(id)
    setSessionsError(null)
    try {
      await revokeSession(id)
      toast.success('Session revoked')
      setRevokeTarget(null)
      loadSessions()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to revoke session'
      setSessionsError(msg)
      toast.error(msg)
    } finally {
      setRevokingId(null)
    }
  }

  async function onConfirmRevokeOthers() {
    setRevokingOthers(true)
    setSessionsError(null)
    try {
      await revokeOtherSessions()
      toast.success('Other sessions revoked')
      setConfirmRevokeOthers(false)
      loadSessions()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to revoke sessions'
      setSessionsError(msg)
      toast.error(msg)
    } finally {
      setRevokingOthers(false)
    }
  }

  const otherCount = sessions?.filter((s) => !s.current).length ?? 0
  const lede =
    tab === 'sessions'
      ? 'Active sign-ins for this account. Revoke a session to sign that client out.'
      : 'Profile details and password for this administrator.'

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Account</p>
          <h1 className="title">Account</h1>
          <p className="lede">{lede}</p>
        </div>
      </header>

      <nav className="detail-tabs" aria-label="Account sections">
        <button
          type="button"
          className={
            tab === 'profile' ? 'detail-tab detail-tab-active' : 'detail-tab'
          }
          aria-current={tab === 'profile' ? 'page' : undefined}
          onClick={() => setTab('profile')}
        >
          Profile
        </button>
        <button
          type="button"
          className={
            tab === 'sessions' ? 'detail-tab detail-tab-active' : 'detail-tab'
          }
          aria-current={tab === 'sessions' ? 'page' : undefined}
          onClick={() => setTab('sessions')}
        >
          Sessions
        </button>
      </nav>

      {tab === 'profile' ? (
        <section className="panel" aria-labelledby="account-profile-heading">
          <h2 id="account-profile-heading" className="panel-title">
            Profile
          </h2>
          {meLoading ? (
            <p className="loading">Loading…</p>
          ) : me ? (
            <div className="info-list" aria-label="Profile">
              <div className="info-row">
                <span className="info-key">Name</span>
                <span className="info-val">{me.name || '—'}</span>
              </div>
              <div className="info-row">
                <span className="info-key">Email</span>
                <span className="info-val mono">{me.email}</span>
              </div>
              <div className="info-row">
                <span className="info-key">Roles</span>
                <span className="info-val">
                  {me.roles?.length ? me.roles.join(', ') : '—'}
                </span>
              </div>
              <div className="info-row">
                <span className="info-key">Password</span>
                <span className="info-val">
                  <button
                    type="button"
                    className="btn btn-ghost"
                    onClick={openPassword}
                  >
                    Change password
                  </button>
                </span>
              </div>
            </div>
          ) : (
            <p className="muted">Could not load profile.</p>
          )}
        </section>
      ) : null}

      {tab === 'sessions' ? (
        <section className="panel" aria-labelledby="account-sessions-heading">
          <div className="list-panel-header">
            <h2 id="account-sessions-heading" className="panel-title">
              Sessions
            </h2>
            {otherCount > 0 ? (
              <button
                type="button"
                className="btn btn-ghost"
                disabled={
                  revokingOthers || sessionsLoading || revokeTarget !== null
                }
                onClick={() => setConfirmRevokeOthers(true)}
              >
                Revoke others
              </button>
            ) : null}
          </div>
          {sessionsError ? (
            <p className="error" role="alert">
              {sessionsError}
            </p>
          ) : null}
          {sessionsLoading && !sessions ? (
            <p className="loading">Loading sessions…</p>
          ) : null}
          {sessions && sessions.length === 0 ? (
            <p className="muted">No active sessions.</p>
          ) : null}
          {sessions && sessions.length > 0 ? (
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Signed in</th>
                    <th>IP</th>
                    <th>Client</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {sessions.map((s) => (
                    <tr key={s.id}>
                      <td title={s.created_at ?? undefined}>
                        {formatRelativeTime(s.created_at)}
                        {s.created_at ? (
                          <div className="muted">
                            {formatDateTime(s.created_at)}
                          </div>
                        ) : null}
                        {s.current ? (
                          <div>
                            <span className="badge badge-status-active">
                              Current
                            </span>
                          </div>
                        ) : null}
                      </td>
                      <td className="mono muted">{s.ip || '—'}</td>
                      <td className="muted" title={s.user_agent || undefined}>
                        {shortUserAgent(s.user_agent)}
                      </td>
                      <td>
                        {s.current ? (
                          <span className="muted">—</span>
                        ) : (
                          <button
                            type="button"
                            className="btn btn-ghost"
                            disabled={
                              revokingId !== null || confirmRevokeOthers
                            }
                            onClick={() => setRevokeTarget(s)}
                          >
                            Revoke
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </section>
      ) : null}

      <ConfirmDialog
        open={revokeTarget !== null}
        title="Revoke this session?"
        description={
          revokeTarget
            ? `This signs out that client${
                revokeTarget.ip ? ` (${revokeTarget.ip})` : ''
              }. Your current session stays signed in.`
            : ''
        }
        confirmLabel="Revoke session"
        danger
        busy={revokingId !== null}
        onConfirm={onConfirmRevoke}
        onCancel={() => {
          if (revokingId === null) setRevokeTarget(null)
        }}
      />

      <ConfirmDialog
        open={confirmRevokeOthers}
        title="Revoke other sessions?"
        description={`Ends ${otherCount} other sign-in${
          otherCount === 1 ? '' : 's'
        }. This browser stays signed in.`}
        confirmLabel="Revoke others"
        danger
        busy={revokingOthers}
        onConfirm={onConfirmRevokeOthers}
        onCancel={() => {
          if (!revokingOthers) setConfirmRevokeOthers(false)
        }}
      />

      {passwordOpen ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={savingPassword}
            onClick={closePassword}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={passwordTitleId}
          >
            <h2 id={passwordTitleId} className="dialog-title">
              Change password
            </h2>
            <p className="dialog-desc muted">
              Local accounts only. New password must be at least 10 characters.
            </p>
            {passwordError ? (
              <p className="error" role="alert">
                {passwordError}
              </p>
            ) : null}
            <form className="users-dialog-form" onSubmit={onChangePassword}>
              <div className="field">
                <label className="label" htmlFor="account-current-password">
                  Current password
                </label>
                <input
                  ref={passwordFirstRef}
                  id="account-current-password"
                  className="input"
                  type="password"
                  autoComplete="current-password"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  disabled={savingPassword}
                  required
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="account-new-password">
                  New password
                </label>
                <input
                  id="account-new-password"
                  className="input"
                  type="password"
                  autoComplete="new-password"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  disabled={savingPassword}
                  required
                  minLength={10}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="account-confirm-password">
                  Confirm new password
                </label>
                <input
                  id="account-confirm-password"
                  className="input"
                  type="password"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  disabled={savingPassword}
                  required
                  minLength={10}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={closePassword}
                  disabled={savingPassword}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={savingPassword}
                  aria-busy={savingPassword}
                >
                  {savingPassword ? 'Saving…' : 'Update password'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </div>
  )
}

function shortUserAgent(ua: string): string {
  const t = ua.trim()
  if (!t) return '—'
  if (t.length <= 64) return t
  return `${t.slice(0, 61)}…`
}
