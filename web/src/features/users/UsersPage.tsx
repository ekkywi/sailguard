import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useOutletContext } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { PermissionDenied } from '../../components/PermissionDenied'
import { TableSkeleton } from '../../components/TableSkeleton'
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
  createUser,
  listRoles,
  listUsers,
  patchUser,
  resetUserPassword,
} from './api'
import type { AdminUser, Role } from './types'

const PAGE_SIZE = 10

type StatusFilter = 'all' | 'active' | 'disabled'

function RolePicker({
  roles,
  selected,
  onToggle,
  disabled,
}: {
  roles: Role[]
  selected: string[]
  onToggle: (code: string) => void
  disabled?: boolean
}) {
  return (
    <div className="role-picker" role="group" aria-label="Roles">
      {roles.map((r) => {
        const on = selected.includes(r.code)
        return (
          <label
            key={r.code}
            className={on ? 'role-option role-option-on' : 'role-option'}
          >
            <input
              type="checkbox"
              checked={on}
              disabled={disabled}
              onChange={() => onToggle(r.code)}
            />
            <span className="role-option-body">
              <span className="role-option-code">{r.code}</span>
              <span className="role-option-desc muted">{r.description}</span>
            </span>
          </label>
        )
      })}
    </div>
  )
}

export default function UsersPage() {
  useDocumentTitle('Users')
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'user.read')
  const canWrite = hasPermission(me, 'user.write')

  const [users, setUsers] = useState<AdminUser[] | null>(null)
  const [rolesCatalog, setRolesCatalog] = useState<Role[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [query, setQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [page, setPage] = useState(1)

  const [createOpen, setCreateOpen] = useState(false)
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [createRoles, setCreateRoles] = useState<string[]>(['viewer'])
  const [creating, setCreating] = useState(false)
  const [createFieldErrors, setCreateFieldErrors] = useState<{
    email?: string
    name?: string
    password?: string
    roles?: string
  }>({})
  const createTitleId = useId()
  const createFirstRef = useRef<HTMLInputElement>(null)

  const [toggleTarget, setToggleTarget] = useState<AdminUser | null>(null)
  const [toggling, setToggling] = useState(false)

  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null)
  const [resetPassword, setResetPassword] = useState('')
  const [resetting, setResetting] = useState(false)
  const resetTitleId = useId()
  const resetFirstRef = useRef<HTMLInputElement>(null)

  const [editTarget, setEditTarget] = useState<AdminUser | null>(null)
  const [editName, setEditName] = useState('')
  const [editRoles, setEditRoles] = useState<string[]>([])
  const [savingEdit, setSavingEdit] = useState(false)
  const editTitleId = useId()
  const editFirstRef = useRef<HTMLInputElement>(null)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    Promise.all([listUsers(), listRoles()])
      .then(([userData, roleData]) => {
        setUsers(userData.items)
        setRolesCatalog(roleData.items)
      })
      .catch((err: unknown) => {
        setUsers(null)
        setError(err instanceof Error ? err.message : 'failed to load users')
      })
      .finally(() => setLoading(false))
  }, [canRead])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (createOpen) createFirstRef.current?.focus()
  }, [createOpen])

  useEffect(() => {
    if (editTarget) editFirstRef.current?.focus()
  }, [editTarget])

  useEffect(() => {
    if (resetTarget) resetFirstRef.current?.focus()
  }, [resetTarget])

  const stats = useMemo(() => {
    const list = users ?? []
    return {
      total: list.length,
      active: list.filter((u) => u.is_active).length,
      disabled: list.filter((u) => !u.is_active).length,
    }
  }, [users])

  const filtered = useMemo(() => {
    if (!users) return []
    const q = query.trim().toLowerCase()
    return users.filter((u) => {
      if (statusFilter === 'active' && !u.is_active) return false
      if (statusFilter === 'disabled' && u.is_active) return false
      if (!q) return true
      return (
        u.name.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q) ||
        u.roles.some((r) => r.toLowerCase().includes(q))
      )
    })
  }, [users, query, statusFilter])

  const total = filtered.length
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE) || 1)

  useEffect(() => {
    setPage(1)
  }, [query, statusFilter])

  useEffect(() => {
    setPage((p) => Math.min(p, pageCount))
  }, [pageCount])

  const pageItems = useMemo(() => {
    const start = (page - 1) * PAGE_SIZE
    return filtered.slice(start, start + PAGE_SIZE)
  }, [filtered, page])

  const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1
  const rangeEnd = Math.min(page * PAGE_SIZE, total)

  function toggleCreateRole(code: string) {
    setCreateRoles((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code],
    )
  }

  function toggleEditRole(code: string) {
    setEditRoles((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code],
    )
  }

  function openCreate() {
    setEmail('')
    setName('')
    setPassword('')
    setCreateRoles(['viewer'])
    setCreateFieldErrors({})
    setError(null)
    setCreateOpen(true)
  }

  function closeCreate() {
    if (creating) return
    setCreateOpen(false)
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    if (!canWrite) return
    const trimmedEmail = email.trim().toLowerCase()
    const trimmedName = name.trim()
    const nextErrors: typeof createFieldErrors = {}
    if (!trimmedEmail) nextErrors.email = 'Email is required'
    if (!trimmedName) nextErrors.name = 'Name is required'
    if (password.length < 10) {
      nextErrors.password = 'Password must be at least 10 characters'
    }
    if (createRoles.length === 0) nextErrors.roles = 'Select at least one role'
    setCreateFieldErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) {
      toast.error('Fix the highlighted fields')
      return
    }

    setCreating(true)
    setError(null)
    try {
      await createUser({
        email: trimmedEmail,
        name: trimmedName,
        password,
        roles: createRoles,
      })
      setCreateOpen(false)
      setCreateFieldErrors({})
      setPage(1)
      toast.success('User created')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to create user'
      setError(msg)
      setCreateFieldErrors({ email: msg })
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  async function confirmToggle() {
    if (!toggleTarget || !canWrite) return
    setToggling(true)
    try {
      await patchUser(toggleTarget.id, { is_active: !toggleTarget.is_active })
      toast.success(
        toggleTarget.is_active ? 'User deactivated' : 'User activated',
      )
      setToggleTarget(null)
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to update user status'
      toast.error(msg)
      setError(msg)
    } finally {
      setToggling(false)
    }
  }

  async function confirmReset(e: FormEvent) {
    e.preventDefault()
    if (!resetTarget || !canWrite) return
    const pwd = resetPassword.trim()
    if (pwd.length < 10) {
      toast.error('password must be at least 10 characters')
      return
    }
    setResetting(true)
    try {
      await resetUserPassword(resetTarget.id, pwd)
      toast.success('Password reset')
      setResetTarget(null)
      setResetPassword('')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to reset password'
      toast.error(msg)
    } finally {
      setResetting(false)
    }
  }

  function openEdit(u: AdminUser) {
    setEditTarget(u)
    setEditName(u.name)
    setEditRoles([...u.roles])
  }

  async function saveEdit(e: FormEvent) {
    e.preventDefault()
    if (!editTarget || !canWrite) return
    const trimmed = editName.trim()
    if (!trimmed) {
      toast.error('name is required')
      return
    }
    if (editRoles.length === 0) {
      toast.error('select at least one role')
      return
    }
    setSavingEdit(true)
    try {
      await patchUser(editTarget.id, { name: trimmed, roles: editRoles })
      toast.success('User updated')
      setEditTarget(null)
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to update user'
      toast.error(msg)
    } finally {
      setSavingEdit(false)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="user.read" title="Users" />
  }

  return (
    <div className="stack users-layout">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">Administration</p>
          <h1 className="title">Users</h1>
          <p className="lede">Local admin accounts and role assignments</p>
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
            <button type="button" className="btn btn-primary" onClick={openCreate}>
              Add user
            </button>
          ) : null}
        </div>
      </header>

      <div className="user-stat-grid" aria-label="User counts">
        <button
          type="button"
          className={
            statusFilter === 'all'
              ? 'user-stat user-stat-active'
              : 'user-stat'
          }
          onClick={() => setStatusFilter('all')}
        >
          <span className="stat-label">Total</span>
          <span className="stat-value">{stats.total}</span>
        </button>
        <button
          type="button"
          className={
            statusFilter === 'active'
              ? 'user-stat user-stat-active'
              : 'user-stat'
          }
          onClick={() => setStatusFilter('active')}
        >
          <span className="stat-label">Active</span>
          <span className="stat-value">{stats.active}</span>
        </button>
        <button
          type="button"
          className={
            statusFilter === 'disabled'
              ? 'user-stat user-stat-active'
              : 'user-stat'
          }
          onClick={() => setStatusFilter('disabled')}
        >
          <span className="stat-label">Disabled</span>
          <span className="stat-value">{stats.disabled}</span>
        </button>
      </div>

      <div className="users-toolbar">
        <div className="field users-search">
          <label className="label" htmlFor="users-query">
            Search
          </label>
          <input
            id="users-query"
            className="input"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Name, email, or role…"
          />
        </div>
      </div>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      <section className="list-panel" aria-labelledby="user-list-heading">
        <div className="list-panel-header">
          <h2 id="user-list-heading" className="panel-title">
            Directory
          </h2>
          {total > 0 ? (
            <p className="muted list-range">
              Showing {rangeStart}–{rangeEnd} of {total}
              {users && filtered.length !== users.length
                ? ` (filtered from ${users.length})`
                : ''}
            </p>
          ) : null}
        </div>

        {loading && !users ? (
          <TableSkeleton label="Loading users…" />
        ) : null}

        {!loading && users && filtered.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">
              {users.length === 0 ? 'No users yet' : 'No matches'}
            </p>
            <p className="muted">
              {users.length === 0
                ? 'Create an admin account to share access beyond the seed user.'
                : 'Try another search or clear the status filter.'}
            </p>
            {users.length === 0 && canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openCreate}
                >
                  Add user
                </button>
              </div>
            ) : null}
          </div>
        ) : null}

        {pageItems.length > 0 ? (
          <>
            <div className="table-wrap table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>User</th>
                    <th>Roles</th>
                    <th>Status</th>
                    <th>Created</th>
                    {canWrite ? <th>Actions</th> : null}
                  </tr>
                </thead>
                <tbody>
                  {pageItems.map((u) => {
                    const isSelf = u.id === me?.id
                    return (
                      <tr key={u.id} className={u.is_active ? undefined : 'row-muted'}>
                        <td>
                          <div className="cell-primary users-name-row">
                            {u.name}
                            {isSelf ? (
                              <span className="badge badge-you">You</span>
                            ) : null}
                          </div>
                          <div className="muted">{u.email}</div>
                        </td>
                        <td>
                          <span className="badge-row">
                            {u.roles.map((role) => (
                              <span key={role} className="badge">
                                {role}
                              </span>
                            ))}
                          </span>
                        </td>
                        <td>
                          <span
                            className={statusBadgeClass(
                              u.is_active ? 'active' : 'disabled',
                            )}
                          >
                            {u.is_active ? 'active' : 'disabled'}
                          </span>
                        </td>
                        <td
                          className="muted"
                          title={formatDateTime(u.created_at)}
                        >
                          {formatRelativeTime(u.created_at)}
                        </td>
                        {canWrite ? (
                          <td>
                            <div className="btn-row">
                              <button
                                type="button"
                                className="btn btn-ghost"
                                onClick={() => openEdit(u)}
                              >
                                Edit
                              </button>
                              <button
                                type="button"
                                className="btn btn-ghost"
                                onClick={() => {
                                  setResetPassword('')
                                  setResetTarget(u)
                                }}
                              >
                                Reset password
                              </button>
                              <button
                                type="button"
                                className={
                                  u.is_active
                                    ? 'btn btn-ghost'
                                    : 'btn btn-ghost'
                                }
                                onClick={() => setToggleTarget(u)}
                                disabled={isSelf && u.is_active}
                                title={
                                  isSelf && u.is_active
                                    ? 'Cannot deactivate yourself'
                                    : undefined
                                }
                              >
                                {u.is_active ? 'Disable' : 'Enable'}
                              </button>
                            </div>
                          </td>
                        ) : null}
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
            {pageCount > 1 ? (
              <div className="pager">
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                >
                  Previous
                </button>
                <span className="muted">
                  Page {page} / {pageCount}
                </span>
                <button
                  type="button"
                  className="btn btn-ghost"
                  disabled={page >= pageCount}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </button>
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
            className="dialog-panel dialog-panel--form dialog-panel--users"
            role="dialog"
            aria-modal="true"
            aria-labelledby={createTitleId}
          >
            <h2 id={createTitleId} className="dialog-title">
              Add user
            </h2>
            <p className="dialog-desc muted">
              Creates a local account. Share the temporary password securely;
              they can sign in immediately if active.
            </p>
            <form className="users-dialog-form" onSubmit={onCreate}>
              <div className="field">
                <label className="label" htmlFor="user-email">
                  Email
                </label>
                <input
                  ref={createFirstRef}
                  id="user-email"
                  className={
                    createFieldErrors.email ? 'input input-invalid' : 'input'
                  }
                  type="email"
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value)
                    setCreateFieldErrors((prev) => ({ ...prev, email: undefined }))
                  }}
                  placeholder="ops@sailguard.local"
                  required
                  disabled={creating}
                  aria-invalid={createFieldErrors.email ? true : undefined}
                />
                {createFieldErrors.email ? (
                  <p className="field-error" role="alert">
                    {createFieldErrors.email}
                  </p>
                ) : null}
              </div>
              <div className="field">
                <label className="label" htmlFor="user-name">
                  Name
                </label>
                <input
                  id="user-name"
                  className={
                    createFieldErrors.name ? 'input input-invalid' : 'input'
                  }
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value)
                    setCreateFieldErrors((prev) => ({ ...prev, name: undefined }))
                  }}
                  placeholder="Ops"
                  required
                  disabled={creating}
                  aria-invalid={createFieldErrors.name ? true : undefined}
                />
                {createFieldErrors.name ? (
                  <p className="field-error" role="alert">
                    {createFieldErrors.name}
                  </p>
                ) : null}
              </div>
              <div className="field">
                <label className="label" htmlFor="user-password">
                  Temporary password
                </label>
                <input
                  id="user-password"
                  className={
                    createFieldErrors.password ? 'input input-invalid' : 'input'
                  }
                  type="password"
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value)
                    setCreateFieldErrors((prev) => ({
                      ...prev,
                      password: undefined,
                    }))
                  }}
                  placeholder="min 10 characters"
                  minLength={10}
                  required
                  disabled={creating}
                  aria-invalid={createFieldErrors.password ? true : undefined}
                />
                {createFieldErrors.password ? (
                  <p className="field-error" role="alert">
                    {createFieldErrors.password}
                  </p>
                ) : null}
              </div>
              <div className="field">
                <span className="label">Roles</span>
                <RolePicker
                  roles={rolesCatalog}
                  selected={createRoles}
                  onToggle={(code) => {
                    toggleCreateRole(code)
                    setCreateFieldErrors((prev) => ({
                      ...prev,
                      roles: undefined,
                    }))
                  }}
                  disabled={creating}
                />
                {createFieldErrors.roles ? (
                  <p className="field-error" role="alert">
                    {createFieldErrors.roles}
                  </p>
                ) : null}
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
                >
                  {creating ? 'Creating…' : 'Create user'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {editTarget ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={savingEdit}
            onClick={() => {
              if (!savingEdit) setEditTarget(null)
            }}
          />
          <div
            className="dialog-panel dialog-panel--form dialog-panel--users"
            role="dialog"
            aria-modal="true"
            aria-labelledby={editTitleId}
          >
            <h2 id={editTitleId} className="dialog-title">
              Edit user
            </h2>
            <p className="dialog-desc muted">{editTarget.email}</p>
            <form className="users-dialog-form" onSubmit={saveEdit}>
              <div className="field">
                <label className="label" htmlFor="edit-user-name">
                  Name
                </label>
                <input
                  ref={editFirstRef}
                  id="edit-user-name"
                  className="input"
                  value={editName}
                  onChange={(e) => setEditName(e.target.value)}
                  required
                  disabled={savingEdit}
                />
              </div>
              <div className="field">
                <span className="label">Roles</span>
                <RolePicker
                  roles={rolesCatalog}
                  selected={editRoles}
                  onToggle={toggleEditRole}
                  disabled={savingEdit}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => setEditTarget(null)}
                  disabled={savingEdit}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={savingEdit}
                >
                  {savingEdit ? 'Saving…' : 'Save changes'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {resetTarget ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={resetting}
            onClick={() => {
              if (!resetting) {
                setResetTarget(null)
                setResetPassword('')
              }
            }}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={resetTitleId}
          >
            <h2 id={resetTitleId} className="dialog-title">
              Reset password
            </h2>
            <p className="dialog-desc muted">
              Set a new temporary password for{' '}
              <strong>{resetTarget.email}</strong>. Share it out of band.
            </p>
            <form className="users-dialog-form" onSubmit={confirmReset}>
              <div className="field">
                <label className="label" htmlFor="reset-password">
                  New password
                </label>
                <input
                  ref={resetFirstRef}
                  id="reset-password"
                  className="input"
                  type="password"
                  value={resetPassword}
                  onChange={(e) => setResetPassword(e.target.value)}
                  minLength={10}
                  required
                  disabled={resetting}
                  placeholder="min 10 characters"
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => {
                    setResetTarget(null)
                    setResetPassword('')
                  }}
                  disabled={resetting}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={resetting}
                >
                  {resetting ? 'Updating…' : 'Update password'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={toggleTarget !== null}
        title={toggleTarget?.is_active ? 'Disable user?' : 'Enable user?'}
        description={
          toggleTarget
            ? toggleTarget.is_active
              ? `${toggleTarget.email} will not be able to sign in.`
              : `${toggleTarget.email} will be able to sign in again.`
            : ''
        }
        confirmLabel={toggleTarget?.is_active ? 'Disable' : 'Enable'}
        danger={toggleTarget?.is_active === true}
        busy={toggling}
        onConfirm={() => void confirmToggle()}
        onCancel={() => setToggleTarget(null)}
      />
    </div>
  )
}
