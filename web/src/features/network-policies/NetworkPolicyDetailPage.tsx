import { FormEvent, useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useNavigate, useOutletContext, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { PermissionDenied } from '../../components/PermissionDenied'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatRelativeTime } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import { listDevices } from '../devices/api'
import type { Device } from '../devices/types'
import { listGroups } from '../groups/api'
import type { DeviceGroup } from '../groups/types'
import {
  addNetworkPolicyAssignment,
  addNetworkPolicyRule,
  deleteNetworkPolicy,
  deleteNetworkPolicyRule,
  getNetworkPolicy,
  listNetworkPolicyAssignments,
  listNetworkPolicyRules,
  patchNetworkPolicy,
  publishNetworkPolicy,
  removeNetworkPolicyAssignment,
  updateNetworkPolicyRule,
} from './api'
import type {
  NetworkPolicy,
  NetworkPolicyAssignment,
  NetworkPolicyRule,
} from './types'

function parsePort(raw: string): number | null | undefined {
  const t = raw.trim()
  if (!t) return null
  const n = Number(t)
  if (!Number.isInteger(n) || n < 1 || n > 65535) return undefined
  return n
}

export default function NetworkPolicyDetailPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'network.read')
  const canWrite = hasPermission(me, 'network.write')
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [policy, setPolicy] = useState<NetworkPolicy | null>(null)
  useDocumentTitle(policy?.name ?? 'Network policy')
  const [rules, setRules] = useState<NetworkPolicyRule[] | null>(null)
  const [assignments, setAssignments] = useState<
    NetworkPolicyAssignment[] | null
  >(null)
  const [groups, setGroups] = useState<DeviceGroup[] | null>(null)
  const [devices, setDevices] = useState<Device[] | null>(null)

  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [ruleAction, setRuleAction] = useState('deny')
  const [ruleDestType, setRuleDestType] = useState('host')
  const [ruleDestValue, setRuleDestValue] = useState('')
  const [rulePort, setRulePort] = useState('')
  const [ruleLabel, setRuleLabel] = useState('')
  const [addingRule, setAddingRule] = useState(false)
  const [addRuleOpen, setAddRuleOpen] = useState(false)
  const addRuleTitleId = useId()
  const addRuleFirstRef = useRef<HTMLSelectElement>(null)

  const [editingRule, setEditingRule] = useState<NetworkPolicyRule | null>(null)
  const [editAction, setEditAction] = useState('deny')
  const [editDestType, setEditDestType] = useState('host')
  const [editDestValue, setEditDestValue] = useState('')
  const [editPort, setEditPort] = useState('')
  const [editLabel, setEditLabel] = useState('')
  const [savingRule, setSavingRule] = useState(false)
  const editTitleId = useId()
  const editFirstFieldRef = useRef<HTMLSelectElement>(null)

  const [assignScope, setAssignScope] = useState('org')
  const [assignGroupId, setAssignGroupId] = useState('')
  const [assignDeviceId, setAssignDeviceId] = useState('')
  const [addingAssign, setAddingAssign] = useState(false)
  const [addAssignOpen, setAddAssignOpen] = useState(false)
  const addAssignTitleId = useId()
  const addAssignFirstRef = useRef<HTMLSelectElement>(null)

  const [modeOpen, setModeOpen] = useState(false)
  const modeTitleId = useId()
  const modeFirstRef = useRef<HTMLSelectElement>(null)
  const [modeDraft, setModeDraft] = useState('audit')
  const [savingMode, setSavingMode] = useState(false)

  const [removeAssign, setRemoveAssign] =
    useState<NetworkPolicyAssignment | null>(null)
  const [removeRule, setRemoveRule] = useState<NetworkPolicyRule | null>(null)
  const [removing, setRemoving] = useState(false)
  const [removingRule, setRemovingRule] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [confirmDeletePolicy, setConfirmDeletePolicy] = useState(false)
  const [deletingPolicy, setDeletingPolicy] = useState(false)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    if (!id) {
      setError('missing policy id')
      setLoading(false)
      return
    }

    setLoading(true)
    setError(null)

    Promise.all([
      getNetworkPolicy(id),
      listNetworkPolicyRules(id),
      listNetworkPolicyAssignments(id),
      listGroups(),
      listDevices({ limit: 500 }),
    ])
      .then(([p, ruleData, assignData, groupData, deviceData]) => {
        setPolicy(p)
        setModeDraft(p.mode || 'audit')
        setRules(ruleData.items)
        setAssignments(assignData.items)
        setGroups(groupData.items)
        setDevices(deviceData.items)
      })
      .catch((err: unknown) => {
        setPolicy(null)
        setRules(null)
        setAssignments(null)
        setError(
          err instanceof Error ? err.message : 'failed to load network policy',
        )
      })
      .finally(() => setLoading(false))
  }, [id, canRead])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!editingRule) return
    editFirstFieldRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !savingRule) setEditingRule(null)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [editingRule, savingRule])

  useEffect(() => {
    if (!addRuleOpen) return
    addRuleFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !addingRule) setAddRuleOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [addRuleOpen, addingRule])

  useEffect(() => {
    if (!addAssignOpen) return
    addAssignFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !addingAssign) setAddAssignOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [addAssignOpen, addingAssign])

  useEffect(() => {
    if (!modeOpen) return
    modeFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !savingMode) setModeOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [modeOpen, savingMode])

  function openAddRule() {
    setRuleAction('deny')
    setRuleDestType('host')
    setRuleDestValue('')
    setRulePort('')
    setRuleLabel('')
    setError(null)
    setAddRuleOpen(true)
  }

  function openAddAssign() {
    setAssignScope('org')
    setAssignGroupId('')
    setAssignDeviceId('')
    setError(null)
    setAddAssignOpen(true)
  }

  function openMode() {
    if (!policy) return
    setModeDraft(policy.mode || 'audit')
    setError(null)
    setModeOpen(true)
  }

  function beginEditRule(r: NetworkPolicyRule) {
    setEditingRule(r)
    setEditAction(r.action || 'deny')
    setEditDestType(r.dest_type || 'host')
    setEditDestValue(r.dest_value || '')
    setEditPort(r.port != null ? String(r.port) : '')
    setEditLabel(r.label || '')
    setError(null)
  }

  async function onAddRule(e: FormEvent) {
    e.preventDefault()
    if (!id) return
    const dest = ruleDestValue.trim()
    if (!dest) {
      setError('Destination value is required')
      return
    }
    const port = parsePort(rulePort)
    if (port === undefined) {
      setError('Port must be an integer 1–65535')
      return
    }

    setAddingRule(true)
    setError(null)
    try {
      await addNetworkPolicyRule(id, {
        action: ruleAction,
        dest_type: ruleDestType,
        dest_value: dest,
        port,
        label: ruleLabel.trim(),
      })
      setAddRuleOpen(false)
      toast.success('Rule added')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to add rule'
      setError(msg)
      toast.error(msg)
    } finally {
      setAddingRule(false)
    }
  }

  async function onSaveRule(e: FormEvent) {
    e.preventDefault()
    if (!id || !editingRule) return
    const dest = editDestValue.trim()
    if (!dest) {
      setError('Destination value is required')
      return
    }
    const port = parsePort(editPort)
    if (port === undefined) {
      setError('Port must be an integer 1–65535')
      return
    }

    setSavingRule(true)
    setError(null)
    try {
      await updateNetworkPolicyRule(id, editingRule.id, {
        action: editAction,
        dest_type: editDestType,
        dest_value: dest,
        port,
        label: editLabel.trim(),
      })
      setEditingRule(null)
      toast.success('Rule updated')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to update rule'
      setError(msg)
      toast.error(msg)
    } finally {
      setSavingRule(false)
    }
  }

  async function onConfirmRemoveRule() {
    if (!id || !removeRule) return
    setRemovingRule(true)
    try {
      await deleteNetworkPolicyRule(id, removeRule.id)
      setRemoveRule(null)
      toast.success('Rule deleted')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to delete rule'
      setError(msg)
      toast.error(msg)
    } finally {
      setRemovingRule(false)
    }
  }

  async function onAddAssign(e: FormEvent) {
    e.preventDefault()
    if (!id) return
    setAddingAssign(true)
    setError(null)
    try {
      const body: {
        scope: string
        group_id?: string
        device_id?: string
      } = { scope: assignScope }
      if (assignScope === 'group') {
        if (!assignGroupId) {
          setError('Select a group')
          setAddingAssign(false)
          return
        }
        body.group_id = assignGroupId
      }
      if (assignScope === 'device') {
        if (!assignDeviceId) {
          setError('Select a device')
          setAddingAssign(false)
          return
        }
        body.device_id = assignDeviceId
      }
      await addNetworkPolicyAssignment(id, body)
      setAddAssignOpen(false)
      toast.success('Assignment added')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to add assignment'
      setError(msg)
      toast.error(msg)
    } finally {
      setAddingAssign(false)
    }
  }

  async function onConfirmRemoveAssign() {
    if (!id || !removeAssign) return
    setRemoving(true)
    try {
      await removeNetworkPolicyAssignment(id, removeAssign.id)
      setRemoveAssign(null)
      toast.success('Assignment removed')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to remove assignment'
      setError(msg)
      toast.error(msg)
    } finally {
      setRemoving(false)
    }
  }

  async function onSaveMode() {
    if (!id || !policy) return
    setSavingMode(true)
    setError(null)
    try {
      await patchNetworkPolicy(id, { mode: modeDraft })
      setModeOpen(false)
      toast.success('Mode updated')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to update mode'
      setError(msg)
      toast.error(msg)
    } finally {
      setSavingMode(false)
    }
  }

  async function onPublish() {
    if (!id) return
    setPublishing(true)
    setError(null)
    try {
      await publishNetworkPolicy(id)
      toast.success('Network policy published')
      load()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to publish'
      setError(msg)
      toast.error(msg)
    } finally {
      setPublishing(false)
    }
  }

  async function onDeletePolicy() {
    if (!id) return
    setDeletingPolicy(true)
    try {
      await deleteNetworkPolicy(id)
      toast.success('Network policy deleted')
      navigate('/network-policies')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'failed to delete policy'
      setError(msg)
      toast.error(msg)
      setConfirmDeletePolicy(false)
    } finally {
      setDeletingPolicy(false)
    }
  }

  function assignmentLabel(a: NetworkPolicyAssignment): string {
    if (a.scope === 'org') return 'Organization'
    if (a.scope === 'group') {
      const g = groups?.find((x) => x.id === a.group_id)
      return g ? `Group: ${g.name}` : `Group: ${a.group_id ?? '—'}`
    }
    const d = devices?.find((x) => x.id === a.device_id)
    return d ? `Device: ${d.hostname}` : `Device: ${a.device_id ?? '—'}`
  }

  function destPlaceholder(destType: string): string {
    switch (destType) {
      case 'host':
        return 'activation.windows.com'
      case 'cidr':
        return '203.0.113.0/24'
      default:
        return '1.1.1.1'
    }
  }

  if (!canRead) {
    return (
      <PermissionDenied permission="network.read" title="Network policy" />
    )
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">
            <Link to="/network-policies" className="crumb">
              Network policies
            </Link>
            <span className="crumb-sep">/</span>
            Detail
          </p>
          <h1 className="title">{policy?.name ?? 'Network policy'}</h1>
          <p className="lede">
            {policy?.description ||
              'Add destination rules, assign scope, then publish.'}
            {policy ? (
              <span className="page-meta">
                {' '}
                · mode {policy.mode} · v{policy.version}
                {policy.published_at
                  ? ` · published ${formatRelativeTime(policy.published_at)}`
                  : ' · draft'}
              </span>
            ) : null}
          </p>
        </div>
        <div className="page-header-actions">
          <button
            type="button"
            className="btn btn-ghost"
            onClick={load}
            disabled={loading}
          >
            Refresh
          </button>
          {policy && canWrite ? (
            <button type="button" className="btn btn-ghost" onClick={openMode}>
              Change mode
            </button>
          ) : null}
          {canWrite ? (
            <button
              type="button"
              className="btn btn-primary"
              onClick={onPublish}
              disabled={publishing || !policy}
            >
              {publishing ? 'Publishing…' : 'Publish'}
            </button>
          ) : null}
          {policy && canWrite ? (
            <button
              type="button"
              className="btn btn-danger"
              onClick={() => setConfirmDeletePolicy(true)}
            >
              Delete
            </button>
          ) : null}
        </div>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading && !policy ? (
        <p className="loading">Loading network policy…</p>
      ) : null}

      {policy ? (
        <div className="info-list" aria-label="Policy details">
          <div className="info-row">
            <span className="info-key">Mode</span>
            <span className="info-val">
              <span
                className={
                  policy.mode === 'enforce'
                    ? 'badge-mode badge-mode-enforce'
                    : 'badge-mode'
                }
              >
                {policy.mode}
              </span>
            </span>
          </div>
          <div className="info-row">
            <span className="info-key">Priority</span>
            <span className="info-val">{policy.priority}</span>
          </div>
          <div className="info-row">
            <span className="info-key">Version</span>
            <span className="info-val">{policy.version}</span>
          </div>
          <div className="info-row">
            <span className="info-key">ID</span>
            <span className="info-val mono">{policy.id}</span>
          </div>
        </div>
      ) : null}

      <section className="list-panel" aria-labelledby="net-rules-heading">
        <div className="list-panel-header">
          <h2 id="net-rules-heading" className="panel-title">
            Destination rules
          </h2>
          {policy && canWrite ? (
            <button
              type="button"
              className="btn btn-ghost"
              onClick={openAddRule}
            >
              Add rule
            </button>
          ) : null}
        </div>
        {rules && rules.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No rules yet</p>
            <p className="muted">
              {canWrite
                ? 'Add a host or IP deny rule (for example activation.windows.com), then publish.'
                : 'No rules on this policy.'}
            </p>
            {canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openAddRule}
                >
                  Add rule
                </button>
              </div>
            ) : null}
          </div>
        ) : null}
        {rules && rules.length > 0 ? (
          <div className="table-wrap table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Action</th>
                  <th>Type</th>
                  <th>Destination</th>
                  <th>Port</th>
                  <th>Label</th>
                  {canWrite ? <th /> : null}
                </tr>
              </thead>
              <tbody>
                {rules.map((r) => (
                  <tr key={r.id}>
                    <td className="mono">{r.action}</td>
                    <td className="mono">{r.dest_type}</td>
                    <td className="mono">{r.dest_value}</td>
                    <td className="muted">{r.port ?? '—'}</td>
                    <td>{r.label || '—'}</td>
                    {canWrite ? (
                      <td>
                        <div className="table-row-actions">
                          <button
                            type="button"
                            className="btn btn-ghost"
                            onClick={() => beginEditRule(r)}
                          >
                            Edit
                          </button>
                          <button
                            type="button"
                            className="btn btn-ghost"
                            onClick={() => setRemoveRule(r)}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
                    ) : null}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      <section className="list-panel" aria-labelledby="net-assign-heading">
        <div className="list-panel-header">
          <h2 id="net-assign-heading" className="panel-title">
            Assignments
          </h2>
          {policy && canWrite ? (
            <button
              type="button"
              className="btn btn-ghost"
              onClick={openAddAssign}
            >
              Add assignment
            </button>
          ) : null}
        </div>
        {assignments && assignments.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No assignments</p>
            <p className="muted">
              {canWrite
                ? 'Assign to org, a group, or a device before publishing.'
                : 'No assignments on this policy.'}
            </p>
            {canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openAddAssign}
                >
                  Add assignment
                </button>
              </div>
            ) : null}
          </div>
        ) : null}
        {assignments && assignments.length > 0 ? (
          <div className="table-wrap table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Scope</th>
                  <th>Target</th>
                  {canWrite ? <th /> : null}
                </tr>
              </thead>
              <tbody>
                {assignments.map((a) => (
                  <tr key={a.id}>
                    <td className="mono">{a.scope}</td>
                    <td>{assignmentLabel(a)}</td>
                    {canWrite ? (
                      <td>
                        <button
                          type="button"
                          className="btn btn-ghost"
                          onClick={() => setRemoveAssign(a)}
                        >
                          Remove
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

      {modeOpen && policy && canWrite ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={savingMode}
            onClick={() => {
              if (!savingMode) setModeOpen(false)
            }}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={modeTitleId}
          >
            <h2 id={modeTitleId} className="dialog-title">
              Change mode
            </h2>
            <p className="dialog-desc muted">
              Enforce installs OS filters on the agent. Publish after switching
              so agents receive a new version.
            </p>
            <form
              className="users-dialog-form"
              onSubmit={(e) => {
                e.preventDefault()
                void onSaveMode()
              }}
            >
              <div className="field">
                <label className="label" htmlFor="net-policy-mode-edit">
                  Mode
                </label>
                <select
                  ref={modeFirstRef}
                  id="net-policy-mode-edit"
                  className="input"
                  value={modeDraft}
                  onChange={(e) => setModeDraft(e.target.value)}
                  disabled={savingMode}
                >
                  <option value="audit">audit</option>
                  <option value="enforce">enforce</option>
                </select>
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => {
                    if (!savingMode) setModeOpen(false)
                  }}
                  disabled={savingMode}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={
                    savingMode || modeDraft === policy.mode || !modeDraft
                  }
                >
                  {savingMode ? 'Saving…' : 'Save mode'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {addRuleOpen && canWrite ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={addingRule}
            onClick={() => {
              if (!addingRule) setAddRuleOpen(false)
            }}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={addRuleTitleId}
          >
            <h2 id={addRuleTitleId} className="dialog-title">
              Add destination rule
            </h2>
            <p className="dialog-desc muted">
              Hostnames are resolved to IPv4 on the agent when enforcing.
              Publish for agents to sync.
            </p>
            <form className="users-dialog-form" onSubmit={onAddRule}>
              <div className="field">
                <label className="label" htmlFor="net-rule-action">
                  Action
                </label>
                <select
                  ref={addRuleFirstRef}
                  id="net-rule-action"
                  className="input"
                  value={ruleAction}
                  onChange={(e) => setRuleAction(e.target.value)}
                  disabled={addingRule}
                >
                  <option value="deny">deny</option>
                  <option value="allow">allow</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="net-rule-dest-type">
                  Destination type
                </label>
                <select
                  id="net-rule-dest-type"
                  className="input"
                  value={ruleDestType}
                  onChange={(e) => setRuleDestType(e.target.value)}
                  disabled={addingRule}
                >
                  <option value="host">host</option>
                  <option value="ip">ip</option>
                  <option value="cidr">cidr</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="net-rule-dest-value">
                  Destination
                </label>
                <input
                  id="net-rule-dest-value"
                  className="input"
                  value={ruleDestValue}
                  onChange={(e) => setRuleDestValue(e.target.value)}
                  placeholder={destPlaceholder(ruleDestType)}
                  required
                  disabled={addingRule}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="net-rule-port">
                  Port (optional)
                </label>
                <input
                  id="net-rule-port"
                  className="input"
                  value={rulePort}
                  onChange={(e) => setRulePort(e.target.value)}
                  placeholder="443"
                  inputMode="numeric"
                  disabled={addingRule}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="net-rule-label">
                  Label (optional)
                </label>
                <input
                  id="net-rule-label"
                  className="input"
                  value={ruleLabel}
                  onChange={(e) => setRuleLabel(e.target.value)}
                  placeholder="win-activation"
                  disabled={addingRule}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => {
                    if (!addingRule) setAddRuleOpen(false)
                  }}
                  disabled={addingRule}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={addingRule}
                >
                  {addingRule ? 'Adding…' : 'Add rule'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {editingRule && canWrite ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={savingRule}
            onClick={() => {
              if (!savingRule) setEditingRule(null)
            }}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={editTitleId}
          >
            <h2 id={editTitleId} className="dialog-title">
              Edit rule
            </h2>
            <form className="users-dialog-form" onSubmit={onSaveRule}>
              <div className="field">
                <label className="label" htmlFor="net-edit-action">
                  Action
                </label>
                <select
                  ref={editFirstFieldRef}
                  id="net-edit-action"
                  className="input"
                  value={editAction}
                  onChange={(e) => setEditAction(e.target.value)}
                  disabled={savingRule}
                >
                  <option value="deny">deny</option>
                  <option value="allow">allow</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="net-edit-dest-type">
                  Destination type
                </label>
                <select
                  id="net-edit-dest-type"
                  className="input"
                  value={editDestType}
                  onChange={(e) => setEditDestType(e.target.value)}
                  disabled={savingRule}
                >
                  <option value="host">host</option>
                  <option value="ip">ip</option>
                  <option value="cidr">cidr</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="net-edit-dest-value">
                  Destination
                </label>
                <input
                  id="net-edit-dest-value"
                  className="input"
                  value={editDestValue}
                  onChange={(e) => setEditDestValue(e.target.value)}
                  required
                  disabled={savingRule}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="net-edit-port">
                  Port (optional)
                </label>
                <input
                  id="net-edit-port"
                  className="input"
                  value={editPort}
                  onChange={(e) => setEditPort(e.target.value)}
                  inputMode="numeric"
                  disabled={savingRule}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="net-edit-label">
                  Label (optional)
                </label>
                <input
                  id="net-edit-label"
                  className="input"
                  value={editLabel}
                  onChange={(e) => setEditLabel(e.target.value)}
                  disabled={savingRule}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => {
                    if (!savingRule) setEditingRule(null)
                  }}
                  disabled={savingRule}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={savingRule}
                >
                  {savingRule ? 'Saving…' : 'Save rule'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {addAssignOpen && canWrite ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={addingAssign}
            onClick={() => {
              if (!addingAssign) setAddAssignOpen(false)
            }}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={addAssignTitleId}
          >
            <h2 id={addAssignTitleId} className="dialog-title">
              Add assignment
            </h2>
            <form className="users-dialog-form" onSubmit={onAddAssign}>
              <div className="field">
                <label className="label" htmlFor="net-assign-scope">
                  Scope
                </label>
                <select
                  ref={addAssignFirstRef}
                  id="net-assign-scope"
                  className="input"
                  value={assignScope}
                  onChange={(e) => setAssignScope(e.target.value)}
                  disabled={addingAssign}
                >
                  <option value="org">org</option>
                  <option value="group">group</option>
                  <option value="device">device</option>
                </select>
              </div>
              {assignScope === 'group' ? (
                <div className="field">
                  <label className="label" htmlFor="net-assign-group">
                    Group
                  </label>
                  <select
                    id="net-assign-group"
                    className="input"
                    value={assignGroupId}
                    onChange={(e) => setAssignGroupId(e.target.value)}
                    disabled={addingAssign}
                    required
                  >
                    <option value="">Select group…</option>
                    {(groups ?? []).map((g) => (
                      <option key={g.id} value={g.id}>
                        {g.name}
                      </option>
                    ))}
                  </select>
                </div>
              ) : null}
              {assignScope === 'device' ? (
                <div className="field">
                  <label className="label" htmlFor="net-assign-device">
                    Device
                  </label>
                  <select
                    id="net-assign-device"
                    className="input"
                    value={assignDeviceId}
                    onChange={(e) => setAssignDeviceId(e.target.value)}
                    disabled={addingAssign}
                    required
                  >
                    <option value="">Select device…</option>
                    {(devices ?? []).map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.hostname}
                      </option>
                    ))}
                  </select>
                </div>
              ) : null}
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={() => {
                    if (!addingAssign) setAddAssignOpen(false)
                  }}
                  disabled={addingAssign}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={addingAssign}
                >
                  {addingAssign ? 'Adding…' : 'Add assignment'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={Boolean(removeRule)}
        title="Delete rule?"
        description={
          removeRule
            ? `Remove ${removeRule.action} ${removeRule.dest_type} ${removeRule.dest_value}? Publish afterward so agents drop it.`
            : ''
        }
        confirmLabel="Delete rule"
        danger
        busy={removingRule}
        onCancel={() => {
          if (!removingRule) setRemoveRule(null)
        }}
        onConfirm={() => {
          void onConfirmRemoveRule()
        }}
      />

      <ConfirmDialog
        open={Boolean(removeAssign)}
        title="Remove assignment?"
        description={
          removeAssign
            ? `Remove ${assignmentLabel(removeAssign)} from this policy?`
            : ''
        }
        confirmLabel="Remove"
        danger
        busy={removing}
        onCancel={() => {
          if (!removing) setRemoveAssign(null)
        }}
        onConfirm={() => {
          void onConfirmRemoveAssign()
        }}
      />

      <ConfirmDialog
        open={confirmDeletePolicy}
        title="Delete network policy?"
        description={
          policy
            ? `Delete “${policy.name}”? Assigned agents will stop receiving it after their next sync.`
            : ''
        }
        confirmLabel="Delete policy"
        busy={deletingPolicy}
        danger
        onCancel={() => {
          if (!deletingPolicy) setConfirmDeletePolicy(false)
        }}
        onConfirm={() => {
          void onDeletePolicy()
        }}
      />
    </div>
  )
}
