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
import { listCategories } from '../categories/api'
import type { Category } from '../categories/types'
import {
  addPolicyAssignment,
  addPolicyRule,
  deletePolicyRule,
  deletePolicy,
  getPolicy,
  listPolicyAssignments,
  listPolicyRules,
  patchPolicy,
  publishPolicy,
  removePolicyAssignment,
  updatePolicyRule,
} from './api'
import type { Policy, PolicyAssignment, PolicyRule } from './types'

export default function PolicyDetailPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'policy.read')
  const canWrite = hasPermission(me, 'policy.write')
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [policy, setPolicy] = useState<Policy | null>(null)
  useDocumentTitle(policy?.name ?? 'Policy')
  const [rules, setRules] = useState<PolicyRule[] | null>(null)
  const [assignments, setAssignments] = useState<PolicyAssignment[] | null>(
    null,
  )
  const [groups, setGroups] = useState<DeviceGroup[] | null>(null)
  const [devices, setDevices] = useState<Device[] | null>(null)

  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [categories, setCategories] = useState<Category[]>([])

  const [ruleAction, setRuleAction] = useState('block')
  const [ruleType, setRuleType] = useState('process_name')
  const [ruleOperator, setRuleOperator] = useState('equals')
  const [ruleValue, setRuleValue] = useState('')
  const [ruleCategory, setRuleCategory] = useState('')
  const [addingRule, setAddingRule] = useState(false)
  const [addRuleOpen, setAddRuleOpen] = useState(false)
  const addRuleTitleId = useId()
  const addRuleFirstRef = useRef<HTMLSelectElement>(null)

  const [editingRule, setEditingRule] = useState<PolicyRule | null>(null)
  const [editAction, setEditAction] = useState('block')
  const [editType, setEditType] = useState('process_name')
  const [editOperator, setEditOperator] = useState('equals')
  const [editValue, setEditValue] = useState('')
  const [editCategory, setEditCategory] = useState('')
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

  const [removeAssign, setRemoveAssign] = useState<PolicyAssignment | null>(
    null,
  )
  const [removeRule, setRemoveRule] = useState<PolicyRule | null>(null)
  const [removing, setRemoving] = useState(false)
  const [removingRule, setRemovingRule] = useState(false)
  const [publishing, setPublishing] = useState(false)
  const [modeDraft, setModeDraft] = useState('audit')
  const [savingMode, setSavingMode] = useState(false)
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
      getPolicy(id),
      listPolicyRules(id),
      listPolicyAssignments(id),
      listGroups(),
      listDevices(),
      listCategories(),
    ])
      .then(([p, ruleData, assignData, groupData, deviceData, categoryData]) => {
        setPolicy(p)
        setModeDraft(p.mode || 'audit')
        setRules(ruleData.items)
        setAssignments(assignData.items)
        setGroups(groupData.items)
        setDevices(deviceData.items)
        setCategories(categoryData.items)
        setRuleCategory((prev) =>
          prev || (categoryData.items[0]?.code ?? ''),
        )
      })
      .catch((err: unknown) => {
        setPolicy(null)
        setRules(null)
        setAssignments(null)
        setError(err instanceof Error ? err.message : 'failed to load policy')
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
      if (e.key === 'Escape' && !savingRule) {
        setEditingRule(null)
      }
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
    setRuleAction('block')
    setRuleType('process_name')
    setRuleOperator('equals')
    setRuleValue('')
    setRuleCategory(categories[0]?.code ?? '')
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

  function beginEditRule(r: PolicyRule) {
    setEditingRule(r)
    setEditAction(r.action || 'block')
    setEditType(r.rule_type || 'process_name')
    setEditOperator(r.operator || 'equals')
    setEditValue(r.value || '')
    setEditCategory(r.category_code || r.value || categories[0]?.code || '')
    setError(null)
  }

  function cancelEditRule() {
    if (savingRule) return
    setEditingRule(null)
  }

  async function onAddRule(e: FormEvent) {
    e.preventDefault()
    if (!id) return

    const isCategory = ruleType === 'category'
    if (isCategory) {
      if (!ruleCategory.trim()) {
        setError('category is required')
        return
      }
    } else if (!ruleValue.trim()) {
      setError('rule value is required')
      return
    }

    setAddingRule(true)
    setError(null)
    try {
      await addPolicyRule(
        id,
        isCategory
          ? {
              action: ruleAction,
              rule_type: 'category',
              category_code: ruleCategory.trim(),
              operator: 'equals',
            }
          : {
              action: ruleAction,
              rule_type: ruleType,
              operator: ruleOperator,
              value: ruleValue.trim(),
            },
      )
      if (!isCategory) setRuleValue('')
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

  async function onSaveEditRule(e: FormEvent) {
    e.preventDefault()
    if (!id || !editingRule) return

    const isCategory = editType === 'category'
    if (isCategory) {
      if (!editCategory.trim()) {
        setError('category is required')
        return
      }
    } else if (!editValue.trim()) {
      setError('rule value is required')
      return
    }

    setSavingRule(true)
    setError(null)
    try {
      await updatePolicyRule(
        id,
        editingRule.id,
        isCategory
          ? {
              action: editAction,
              rule_type: 'category',
              category_code: editCategory.trim(),
              operator: 'equals',
            }
          : {
              action: editAction,
              rule_type: editType,
              operator: editOperator,
              value: editValue.trim(),
            },
      )
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

  function ruleMatchLabel(r: PolicyRule) {
    if (r.rule_type === 'category') {
      return r.category_code || r.value || '—'
    }
    return r.value || '—'
  }

  function ruleValuePlaceholder(type: string) {
    switch (type) {
      case 'path':
        return 'C:\\Windows\\System32\\notepad.exe'
      case 'hash_sha256':
        return 'sha256 hex digest'
      case 'publisher':
        return 'Microsoft Corporation'
      case 'product_name':
        return 'Notepad'
      default:
        return 'notepad.exe'
    }
  }

  async function onAddAssignment(e: FormEvent) {
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
          setError('select a group')
          setAddingAssign(false)
          return
        }
        body.group_id = assignGroupId
      }
      if (assignScope === 'device') {
        if (!assignDeviceId) {
          setError('select a device')
          setAddingAssign(false)
          return
        }
        body.device_id = assignDeviceId
      }
      await addPolicyAssignment(id, body)
      setAssignGroupId('')
      setAssignDeviceId('')
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
    setError(null)
    try {
      await removePolicyAssignment(id, removeAssign.id)
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

  async function onConfirmRemoveRule() {
    if (!id || !removeRule) return
    setRemovingRule(true)
    setError(null)
    try {
      await deletePolicyRule(id, removeRule.id)
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

  async function onSaveMode() {
    if (!id) return
    setSavingMode(true)
    setError(null)
    try {
      const p = await patchPolicy(id, { mode: modeDraft })
      setPolicy(p)
      setModeDraft(p.mode)
      setModeOpen(false)
      toast.success(`Mode set to ${p.mode}`)
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
      const p = await publishPolicy(id)
      setPolicy(p)
      toast.success('Policy published', `Version ${p.version}`)
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to publish policy'
      setError(msg)
      toast.error(msg)
    } finally {
      setPublishing(false)
    }
  }

  async function onConfirmDeletePolicy() {
    if (!id) return
    setDeletingPolicy(true)
    setError(null)
    try {
      await deletePolicy(id)
      toast.success('Policy deleted')
      navigate('/policies')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to delete policy'
      setError(msg)
      toast.error(msg)
      setDeletingPolicy(false)
      setConfirmDeletePolicy(false)
    }
  }

  function assignmentLabel(a: PolicyAssignment): string {
    if (a.scope === 'org') return 'Organization'
    if (a.scope === 'group') {
      const g = groups?.find((x) => x.id === a.group_id)
      return g ? `Group: ${g.name}` : `Group: ${a.group_id ?? '—'}`
    }
    const d = devices?.find((x) => x.id === a.device_id)
    return d ? `Device: ${d.hostname}` : `Device: ${a.device_id ?? '—'}`
  }

  if (!canRead) {
    return <PermissionDenied permission="policy.read" title="Policy" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="section-label">
            <Link to="/policies" className="crumb">
              Policies
            </Link>
            <span className="crumb-sep">/</span>
            Detail
          </p>
          <h1 className="title">{policy?.name ?? 'Policy'}</h1>
          <p className="lede">
            {policy?.description || 'Add rules, assign scope, then publish.'}
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

      {loading && !policy ? <p className="loading">Loading policy…</p> : null}

      {policy ? (
        <>
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
        </>
      ) : null}

      <section className="list-panel" aria-labelledby="rules-heading">
        <div className="list-panel-header">
          <h2 id="rules-heading" className="panel-title">
            Rules
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
                ? 'Add a process_name block rule, then publish.'
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
                  <th>Operator</th>
                  <th>Value</th>
                  {canWrite ? <th /> : null}
                </tr>
              </thead>
              <tbody>
                {rules.map((r) => (
                  <tr
                    key={r.id}
                    className={
                      editingRule?.id === r.id ? 'is-editing' : undefined
                    }
                  >
                    <td className="mono">{r.action}</td>
                    <td className="mono">{r.rule_type}</td>
                    <td className="muted">
                      {r.rule_type === 'category' ? '—' : r.operator}
                    </td>
                    <td>
                      <div className="cell-primary mono">
                        {ruleMatchLabel(r)}
                      </div>
                    </td>
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

      <section className="list-panel" aria-labelledby="assign-heading">
        <div className="list-panel-header">
          <h2 id="assign-heading" className="panel-title">
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
              Audit reports matches only. Enforce terminates matching processes
              on the agent — publish after switching so operators see a new
              version.
            </p>
            <form
              className="users-dialog-form"
              onSubmit={(e) => {
                e.preventDefault()
                void onSaveMode()
              }}
            >
              <div className="field">
                <label className="label" htmlFor="policy-mode-edit">
                  Mode
                </label>
                <select
                  ref={modeFirstRef}
                  id="policy-mode-edit"
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
              Add rule
            </h2>
            <p className="dialog-desc muted">
              New rules apply after the next publish (agents sync the new
              version).
            </p>
            <form
              className="rule-form-grid rule-form-grid--dialog"
              onSubmit={onAddRule}
            >
              <div className="field">
                <label className="label" htmlFor="rule-action">
                  Action
                </label>
                <select
                  ref={addRuleFirstRef}
                  id="rule-action"
                  className="input"
                  value={ruleAction}
                  onChange={(e) => setRuleAction(e.target.value)}
                  disabled={addingRule}
                >
                  <option value="block">block</option>
                  <option value="allow">allow</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="rule-type">
                  Type
                </label>
                <select
                  id="rule-type"
                  className="input"
                  value={ruleType}
                  onChange={(e) => setRuleType(e.target.value)}
                  disabled={addingRule}
                >
                  <option value="process_name">process_name</option>
                  <option value="path">path</option>
                  <option value="hash_sha256">hash_sha256</option>
                  <option value="publisher">publisher</option>
                  <option value="product_name">product_name</option>
                  <option value="category">category</option>
                </select>
              </div>
              {ruleType === 'category' ? (
                <div className="field rule-form-value">
                  <label className="label" htmlFor="rule-category">
                    Category
                  </label>
                  <select
                    id="rule-category"
                    className="input"
                    value={ruleCategory}
                    onChange={(e) => setRuleCategory(e.target.value)}
                    required
                    disabled={addingRule}
                  >
                    {categories.length === 0 ? (
                      <option value="">No categories</option>
                    ) : (
                      categories.map((c) => (
                        <option key={c.code} value={c.code}>
                          {c.name} ({c.code})
                        </option>
                      ))
                    )}
                  </select>
                </div>
              ) : (
                <>
                  <div className="field">
                    <label className="label" htmlFor="rule-op">
                      Operator
                    </label>
                    <select
                      id="rule-op"
                      className="input"
                      value={ruleOperator}
                      onChange={(e) => setRuleOperator(e.target.value)}
                      disabled={addingRule}
                    >
                      <option value="equals">equals</option>
                      <option value="prefix">prefix</option>
                      <option value="contains">contains</option>
                    </select>
                  </div>
                  <div className="field rule-form-value">
                    <label className="label" htmlFor="rule-value">
                      Value
                    </label>
                    <input
                      id="rule-value"
                      className="input"
                      value={ruleValue}
                      onChange={(e) => setRuleValue(e.target.value)}
                      placeholder={ruleValuePlaceholder(ruleType)}
                      required
                      disabled={addingRule}
                    />
                  </div>
                </>
              )}
              <div className="dialog-actions rule-form-actions">
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
                  aria-busy={addingRule}
                >
                  {addingRule ? 'Adding…' : 'Add rule'}
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
            <form className="users-dialog-form" onSubmit={onAddAssignment}>
              <div className="field">
                <label className="label" htmlFor="assign-scope">
                  Scope
                </label>
                <select
                  ref={addAssignFirstRef}
                  id="assign-scope"
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
                  <label className="label" htmlFor="assign-group">
                    Group
                  </label>
                  <select
                    id="assign-group"
                    className="input"
                    value={assignGroupId}
                    onChange={(e) => setAssignGroupId(e.target.value)}
                    required
                    disabled={addingAssign}
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
                  <label className="label" htmlFor="assign-device">
                    Device
                  </label>
                  <select
                    id="assign-device"
                    className="input"
                    value={assignDeviceId}
                    onChange={(e) => setAssignDeviceId(e.target.value)}
                    required
                    disabled={addingAssign}
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
                  aria-busy={addingAssign}
                >
                  {addingAssign ? 'Adding…' : 'Add assignment'}
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
            onClick={cancelEditRule}
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
            <p className="dialog-desc muted">
              {editingRule.action} · {editingRule.rule_type}{' '}
              <span className="mono">{editingRule.value || '(empty)'}</span>
              . Publish after saving so agents pick up the change.
            </p>
            <form className="rule-form-grid rule-form-grid--dialog" onSubmit={onSaveEditRule}>
              <div className="field">
                <label className="label" htmlFor="edit-rule-action">
                  Action
                </label>
                <select
                  ref={editFirstFieldRef}
                  id="edit-rule-action"
                  className="input"
                  value={editAction}
                  onChange={(e) => setEditAction(e.target.value)}
                  disabled={savingRule}
                >
                  <option value="block">block</option>
                  <option value="allow">allow</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="edit-rule-type">
                  Type
                </label>
                <select
                  id="edit-rule-type"
                  className="input"
                  value={editType}
                  onChange={(e) => setEditType(e.target.value)}
                  disabled={savingRule}
                >
                  <option value="process_name">process_name</option>
                  <option value="path">path</option>
                  <option value="hash_sha256">hash_sha256</option>
                  <option value="publisher">publisher</option>
                  <option value="product_name">product_name</option>
                  <option value="category">category</option>
                </select>
              </div>
              {editType === 'category' ? (
                <div className="field rule-form-value">
                  <label className="label" htmlFor="edit-rule-category">
                    Category
                  </label>
                  <select
                    id="edit-rule-category"
                    className="input"
                    value={editCategory}
                    onChange={(e) => setEditCategory(e.target.value)}
                    required
                    disabled={savingRule}
                  >
                    {categories.length === 0 ? (
                      <option value="">No categories</option>
                    ) : (
                      categories.map((c) => (
                        <option key={c.code} value={c.code}>
                          {c.name} ({c.code})
                        </option>
                      ))
                    )}
                  </select>
                </div>
              ) : (
                <>
                  <div className="field">
                    <label className="label" htmlFor="edit-rule-op">
                      Operator
                    </label>
                    <select
                      id="edit-rule-op"
                      className="input"
                      value={editOperator}
                      onChange={(e) => setEditOperator(e.target.value)}
                      disabled={savingRule}
                    >
                      <option value="equals">equals</option>
                      <option value="prefix">prefix</option>
                      <option value="contains">contains</option>
                    </select>
                  </div>
                  <div className="field rule-form-value">
                    <label className="label" htmlFor="edit-rule-value">
                      Value
                    </label>
                    <input
                      id="edit-rule-value"
                      className="input"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      placeholder={ruleValuePlaceholder(editType)}
                      required
                      disabled={savingRule}
                    />
                  </div>
                </>
              )}
              <div className="dialog-actions rule-form-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={cancelEditRule}
                  disabled={savingRule}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={savingRule}
                  aria-busy={savingRule}
                >
                  {savingRule ? 'Saving…' : 'Save rule'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={removeAssign !== null}
        title="Remove assignment?"
        description={
          removeAssign
            ? `${assignmentLabel(removeAssign)} will no longer receive this policy after the next publish/sync.`
            : ''
        }
        confirmLabel="Remove"
        cancelLabel="Cancel"
        danger
        busy={removing}
        onConfirm={onConfirmRemoveAssign}
        onCancel={() => {
          if (!removing) setRemoveAssign(null)
        }}
      />

      <ConfirmDialog
        open={removeRule !== null}
        title="Delete rule?"
        description={
          removeRule
            ? `Remove ${removeRule.action} · ${removeRule.rule_type} ${removeRule.value || '(empty)'}? Agents drop it on the next policy sync; publish to bump version.`
            : ''
        }
        confirmLabel="Delete"
        cancelLabel="Cancel"
        danger
        busy={removingRule}
        onConfirm={onConfirmRemoveRule}
        onCancel={() => {
          if (!removingRule) setRemoveRule(null)
        }}
      />

      <ConfirmDialog
        open={confirmDeletePolicy}
        title="Delete policy?"
        description="Rules and assignments for this policy will be removed. Agents keep a local cache until the next policy sync."
        confirmLabel="Delete policy"
        danger
        busy={deletingPolicy}
        onConfirm={onConfirmDeletePolicy}
        onCancel={() => {
          if (!deletingPolicy) setConfirmDeletePolicy(false)
        }}
      />
    </div>
  )
}
