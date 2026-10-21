import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { toast } from '../../components/toast'
import { formatRelativeTime, statusBadgeClass } from '../../lib/format'
import {
  createDeviceOverride,
  deleteDeviceOverride,
  listDeviceOverrides,
} from './api'
import type { DeviceOverride } from './types'

function toLocalInputValue(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function defaultExpiresLocal(): string {
  return toLocalInputValue(new Date(Date.now() + 2 * 60 * 60 * 1000))
}

type DeviceOverridesPanelProps = {
  deviceId: string
  canWrite: boolean
}

export default function DeviceOverridesPanel({
  deviceId,
  canWrite,
}: DeviceOverridesPanelProps) {
  const [overrides, setOverrides] = useState<DeviceOverride[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [includeExpired, setIncludeExpired] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  const [action, setAction] = useState('allow')
  const [ruleType, setRuleType] = useState('process_name')
  const [operator, setOperator] = useState('equals')
  const [value, setValue] = useState('')
  const [reason, setReason] = useState('')
  const [expiresLocal, setExpiresLocal] = useState(defaultExpiresLocal)
  const [creating, setCreating] = useState(false)
  const [removeTarget, setRemoveTarget] = useState<DeviceOverride | null>(null)
  const [removing, setRemoving] = useState(false)

  const createTitleId = useId()
  const createFirstRef = useRef<HTMLSelectElement>(null)

  const load = useCallback(() => {
    setLoading(true)
    setError(null)
    listDeviceOverrides(deviceId, { includeExpired })
      .then((data) => setOverrides(data.items))
      .catch((err: unknown) => {
        setOverrides(null)
        setError(
          err instanceof Error ? err.message : 'failed to load overrides',
        )
      })
      .finally(() => setLoading(false))
  }, [deviceId, includeExpired])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (createOpen) createFirstRef.current?.focus()
  }, [createOpen])

  function openCreate() {
    setAction('allow')
    setRuleType('process_name')
    setOperator('equals')
    setValue('')
    setReason('')
    setExpiresLocal(defaultExpiresLocal())
    setFormError(null)
    setCreateOpen(true)
  }

  function closeCreate() {
    if (creating) return
    setCreateOpen(false)
    setFormError(null)
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    if (!canWrite) return
    const trimmedValue = value.trim()
    const trimmedReason = reason.trim()
    if (!trimmedValue || !trimmedReason) {
      setFormError('value and reason are required')
      return
    }

    let expiresAt: string | undefined
    if (expiresLocal) {
      const parsed = new Date(expiresLocal)
      if (Number.isNaN(parsed.getTime())) {
        setFormError('invalid expiry datetime')
        return
      }
      if (parsed.getTime() <= Date.now()) {
        setFormError('expiry must be in the future')
        return
      }
      expiresAt = parsed.toISOString()
    } else if (action === 'allow') {
      setFormError('allow overrides require an expiry')
      return
    }

    setCreating(true)
    setFormError(null)
    try {
      await createDeviceOverride(deviceId, {
        action,
        rule_type: ruleType,
        operator,
        value: trimmedValue,
        reason: trimmedReason,
        expires_at: expiresAt,
      })
      setCreateOpen(false)
      toast.success('Override created')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to create override'
      setFormError(msg)
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  async function onConfirmRemove() {
    if (!removeTarget) return
    setRemoving(true)
    setError(null)
    try {
      await deleteDeviceOverride(deviceId, removeTarget.id)
      setRemoveTarget(null)
      toast.success('Override removed')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to delete override'
      setError(msg)
      toast.error(msg)
    } finally {
      setRemoving(false)
    }
  }

  return (
    <div className="stack device-module">
      <div className="device-module-toolbar">
        <div className="device-module-intro">
          <h2 className="panel-title">Overrides</h2>
          <p className="muted device-module-lede">
            Time-bounded exceptions for this device only. Allow requires an
            expiry; expired rows stay for history but leave the effective
            policy automatically.
          </p>
        </div>
        <div className="device-module-actions">
          <label className="device-module-check muted">
            <input
              type="checkbox"
              checked={includeExpired}
              onChange={(e) => setIncludeExpired(e.target.checked)}
            />
            Show expired
          </label>
          {canWrite ? (
            <button
              type="button"
              className="btn btn-primary"
              onClick={openCreate}
            >
              Add override
            </button>
          ) : null}
        </div>
      </div>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      <section className="list-panel" aria-label="Override list">
        {loading && !overrides ? (
          <p className="loading">Loading overrides…</p>
        ) : null}

        {!loading && overrides && overrides.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No overrides</p>
            <p className="muted">
              {includeExpired
                ? 'No override rows for this device.'
                : canWrite
                  ? 'Use Add override for a temporary exception.'
                  : 'No active overrides on this device.'}
            </p>
          </div>
        ) : null}

        {overrides && overrides.length > 0 ? (
          <div className="table-wrap table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Action</th>
                  <th>Match</th>
                  <th>Reason</th>
                  <th>Expires</th>
                  <th>Status</th>
                  {canWrite ? <th /> : null}
                </tr>
              </thead>
              <tbody>
                {overrides.map((o) => (
                  <tr key={o.id}>
                    <td className="mono">{o.action}</td>
                    <td>
                      <div className="cell-primary mono">
                        {o.rule_type} {o.operator} {o.value}
                      </div>
                    </td>
                    <td>{o.reason || '—'}</td>
                    <td className="muted" title={o.expires_at ?? undefined}>
                      {o.expires_at
                        ? formatRelativeTime(o.expires_at)
                        : 'never'}
                    </td>
                    <td>
                      <span
                        className={statusBadgeClass(
                          o.expired ? 'offline' : 'active',
                        )}
                      >
                        {o.expired ? 'expired' : 'active'}
                      </span>
                    </td>
                    {canWrite ? (
                      <td>
                        <button
                          type="button"
                          className="btn btn-ghost"
                          onClick={() => setRemoveTarget(o)}
                        >
                          Delete
                        </button>
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      {createOpen && canWrite ? (
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
              Add override
            </h2>
            <p className="dialog-desc muted">
              Applies only to this device and wins over assigned policies.
              Allow overrides require a future expiry and a reason.
            </p>
            {formError ? (
              <p className="error" role="alert">
                {formError}
              </p>
            ) : null}
            <form
              className="rule-form-grid rule-form-grid--dialog"
              onSubmit={onCreate}
            >
              <div className="field">
                <label className="label" htmlFor="ov-action">
                  Action
                </label>
                <select
                  ref={createFirstRef}
                  id="ov-action"
                  className="input"
                  value={action}
                  onChange={(e) => setAction(e.target.value)}
                  disabled={creating}
                >
                  <option value="allow">allow</option>
                  <option value="block">block</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="ov-type">
                  Type
                </label>
                <select
                  id="ov-type"
                  className="input"
                  value={ruleType}
                  onChange={(e) => setRuleType(e.target.value)}
                  disabled={creating}
                >
                  <option value="process_name">process_name</option>
                  <option value="path">path</option>
                  <option value="hash_sha256">hash_sha256</option>
                  <option value="publisher">publisher</option>
                  <option value="product_name">product_name</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="ov-op">
                  Operator
                </label>
                <select
                  id="ov-op"
                  className="input"
                  value={operator}
                  onChange={(e) => setOperator(e.target.value)}
                  disabled={creating}
                >
                  <option value="equals">equals</option>
                  <option value="prefix">prefix</option>
                  <option value="contains">contains</option>
                </select>
              </div>
              <div className="field rule-form-value">
                <label className="label" htmlFor="ov-value">
                  Value
                </label>
                <input
                  id="ov-value"
                  className="input"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder="chrome.exe"
                  required
                  disabled={creating}
                />
              </div>
              <div className="field" style={{ gridColumn: '1 / -1' }}>
                <label className="label" htmlFor="ov-reason">
                  Reason
                </label>
                <input
                  id="ov-reason"
                  className="input"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="ticket-123 temporary unblock"
                  required
                  disabled={creating}
                />
              </div>
              <div className="field" style={{ gridColumn: '1 / -1' }}>
                <label className="label" htmlFor="ov-expires">
                  Expires
                  {action === 'allow' ? ' (required)' : ' (optional)'}
                </label>
                <input
                  id="ov-expires"
                  type="datetime-local"
                  className="input"
                  value={expiresLocal}
                  onChange={(e) => setExpiresLocal(e.target.value)}
                  required={action === 'allow'}
                  disabled={creating}
                />
              </div>
              <div className="dialog-actions rule-form-actions">
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
                  {creating ? 'Creating…' : 'Create override'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={removeTarget !== null}
        title="Delete override?"
        description={
          removeTarget
            ? `Remove ${removeTarget.action} · ${removeTarget.rule_type} ${removeTarget.value}? Agents drop it on the next policy sync.`
            : ''
        }
        confirmLabel="Delete"
        cancelLabel="Cancel"
        danger
        busy={removing}
        onConfirm={onConfirmRemove}
        onCancel={() => {
          if (!removing) setRemoveTarget(null)
        }}
      />
    </div>
  )
}
