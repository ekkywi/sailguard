import {
  FormEvent,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react'
import { Link, useNavigate, useOutletContext, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { PermissionDenied } from '../../components/PermissionDenied'
import { toast } from '../../components/toast'
import type { AppOutletContext } from '../../layouts/AppLayout'
import { formatRelativeTime } from '../../lib/format'
import { hasPermission } from '../../lib/permissions'
import { useDocumentTitle } from '../../lib/useDocumentTitle'
import {
  addCategorySignature,
  deleteCategory,
  deleteCategorySignature,
  getCategory,
  listCategorySignatures,
  patchCategory,
} from './api'
import type { Category, CategorySignature } from './types'

export default function CategoryDetailPage() {
  const { me } = useOutletContext<AppOutletContext>()
  const canRead = hasPermission(me, 'policy.read')
  const canWrite = hasPermission(me, 'policy.write')
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()

  const [category, setCategory] = useState<Category | null>(null)
  useDocumentTitle(category?.name ?? 'Category')
  const [signatures, setSignatures] = useState<CategorySignature[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [nameDraft, setNameDraft] = useState('')
  const [descriptionDraft, setDescriptionDraft] = useState('')
  const [savingMeta, setSavingMeta] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const editTitleId = useId()
  const editFirstRef = useRef<HTMLInputElement>(null)

  const [sigType, setSigType] = useState('process_name')
  const [sigOperator, setSigOperator] = useState('equals')
  const [sigValue, setSigValue] = useState('')
  const [addingSig, setAddingSig] = useState(false)
  const [addSigOpen, setAddSigOpen] = useState(false)
  const addSigTitleId = useId()
  const addSigFirstRef = useRef<HTMLSelectElement>(null)

  const [removeSig, setRemoveSig] = useState<CategorySignature | null>(null)
  const [removingSig, setRemovingSig] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(() => {
    if (!canRead) {
      setLoading(false)
      return
    }
    if (!code) {
      setError('missing category code')
      setLoading(false)
      return
    }

    setLoading(true)
    setError(null)

    Promise.all([getCategory(code), listCategorySignatures(code)])
      .then(([c, sigData]) => {
        setCategory(c)
        setNameDraft(c.name)
        setDescriptionDraft(c.description || '')
        setSignatures(sigData.items)
      })
      .catch((err: unknown) => {
        setCategory(null)
        setSignatures(null)
        setError(
          err instanceof Error ? err.message : 'failed to load category',
        )
      })
      .finally(() => setLoading(false))
  }, [code, canRead])

  useEffect(() => {
    load()
  }, [load])

  useEffect(() => {
    if (!editOpen) return
    editFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !savingMeta) setEditOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [editOpen, savingMeta])

  useEffect(() => {
    if (!addSigOpen) return
    addSigFirstRef.current?.focus()
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && !addingSig) setAddSigOpen(false)
    }
    document.addEventListener('keydown', onKeyDown)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = prevOverflow
    }
  }, [addSigOpen, addingSig])

  const metaDirty =
    !!category &&
    (nameDraft.trim() !== category.name ||
      descriptionDraft.trim() !== (category.description || ''))

  function openEdit() {
    if (!category) return
    setNameDraft(category.name)
    setDescriptionDraft(category.description || '')
    setError(null)
    setEditOpen(true)
  }

  function closeEdit() {
    if (savingMeta) return
    setEditOpen(false)
  }

  function openAddSig() {
    setSigType('process_name')
    setSigOperator('equals')
    setSigValue('')
    setError(null)
    setAddSigOpen(true)
  }

  function closeAddSig() {
    if (addingSig) return
    setAddSigOpen(false)
  }

  async function onSaveMeta(e: FormEvent) {
    e.preventDefault()
    if (!code) return
    const name = nameDraft.trim()
    if (!name) {
      setError('name is required')
      return
    }

    setSavingMeta(true)
    setError(null)
    try {
      const c = await patchCategory(code, {
        name,
        description: descriptionDraft.trim(),
      })
      setCategory(c)
      setNameDraft(c.name)
      setDescriptionDraft(c.description || '')
      setEditOpen(false)
      toast.success('Category updated')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to update category'
      setError(msg)
      toast.error(msg)
    } finally {
      setSavingMeta(false)
    }
  }

  async function onAddSignature(e: FormEvent) {
    e.preventDefault()
    if (!code) return
    const value = sigValue.trim()
    if (!value) {
      setError('signature value is required')
      return
    }

    setAddingSig(true)
    setError(null)
    try {
      await addCategorySignature(code, {
        rule_type: sigType,
        operator: sigOperator,
        value,
      })
      setSigValue('')
      setAddSigOpen(false)
      toast.success('Signature added')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to add signature'
      setError(msg)
      toast.error(msg)
    } finally {
      setAddingSig(false)
    }
  }

  async function onConfirmRemoveSig() {
    if (!code || !removeSig) return
    setRemovingSig(true)
    setError(null)
    try {
      await deleteCategorySignature(code, removeSig.id)
      setRemoveSig(null)
      toast.success('Signature deleted')
      load()
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to delete signature'
      setError(msg)
      toast.error(msg)
    } finally {
      setRemovingSig(false)
    }
  }

  async function onConfirmDeleteCategory() {
    if (!code) return
    setDeleting(true)
    setError(null)
    try {
      await deleteCategory(code)
      toast.success('Category deleted')
      navigate('/categories')
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : 'failed to delete category'
      setError(msg)
      toast.error(msg)
      setConfirmDelete(false)
    } finally {
      setDeleting(false)
    }
  }

  if (!canRead) {
    return <PermissionDenied permission="policy.read" title="Category" />
  }

  return (
    <div className="stack">
      <header className="page-header">
        <div className="page-header-text">
          <p className="crumbs">
            <Link to="/categories" className="crumb">
              Categories
            </Link>
            <span className="crumb-sep">/</span>
            <span className="mono">{code || '…'}</span>
          </p>
          <h1 className="title">{category?.name || 'Category'}</h1>
          <p className="lede">
            Name, description, and signatures used when policy rules reference
            this category.
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
          {category && canWrite ? (
            <button type="button" className="btn btn-ghost" onClick={openEdit}>
              Edit
            </button>
          ) : null}
          {category && canWrite ? (
            <button
              type="button"
              className="btn btn-ghost"
              onClick={openAddSig}
            >
              Add signature
            </button>
          ) : null}
          {canWrite ? (
            <button
              type="button"
              className="btn btn-danger"
              onClick={() => setConfirmDelete(true)}
              disabled={!category || deleting}
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

      {loading && !category ? <p className="loading">Loading…</p> : null}

      {category ? (
        <div className="info-list" aria-label="Category details">
          <div className="info-row">
            <span className="info-key">Code</span>
            <span className="info-val mono">{category.code}</span>
          </div>
          <div className="info-row">
            <span className="info-key">Name</span>
            <span className="info-val">{category.name}</span>
          </div>
          <div className="info-row">
            <span className="info-key">Description</span>
            <span className="info-val">{category.description || '—'}</span>
          </div>
        </div>
      ) : null}

      <section className="list-panel" aria-labelledby="sig-list-heading">
        <div className="list-panel-header">
          <h2 id="sig-list-heading" className="panel-title">
            Signatures
          </h2>
          <div className="btn-row">
            {signatures ? (
              <p className="muted list-range">
                {signatures.length} signature
                {signatures.length === 1 ? '' : 's'}
              </p>
            ) : null}
            {category && canWrite ? (
              <button
                type="button"
                className="btn btn-ghost"
                onClick={openAddSig}
              >
                Add signature
              </button>
            ) : null}
          </div>
        </div>

        {signatures && signatures.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No signatures yet</p>
            <p className="muted">
              {canWrite
                ? 'Add process names or other matchers so category policies can expand.'
                : 'No signatures on this category.'}
            </p>
            {canWrite ? (
              <div className="empty-actions">
                <button
                  type="button"
                  className="btn btn-primary"
                  onClick={openAddSig}
                >
                  Add signature
                </button>
              </div>
            ) : null}
          </div>
        ) : null}

        {signatures && signatures.length > 0 ? (
          <div className="table-wrap table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Operator</th>
                  <th>Value</th>
                  <th>Added</th>
                  {canWrite ? <th /> : null}
                </tr>
              </thead>
              <tbody>
                {signatures.map((s) => (
                  <tr key={s.id}>
                    <td className="mono">{s.rule_type}</td>
                    <td className="muted">{s.operator}</td>
                    <td>
                      <div className="cell-primary mono">{s.value}</div>
                    </td>
                    <td className="muted" title={s.created_at}>
                      {formatRelativeTime(s.created_at)}
                    </td>
                    {canWrite ? (
                      <td>
                        <button
                          type="button"
                          className="btn btn-ghost"
                          onClick={() => setRemoveSig(s)}
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

      {editOpen && category ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={savingMeta}
            onClick={closeEdit}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={editTitleId}
          >
            <h2 id={editTitleId} className="dialog-title">
              Edit category
            </h2>
            <p className="dialog-desc muted">
              Code <span className="mono">{category.code}</span> cannot change.
            </p>
            <form className="users-dialog-form" onSubmit={onSaveMeta}>
              <div className="field">
                <label className="label" htmlFor="cat-name">
                  Name
                </label>
                <input
                  ref={editFirstRef}
                  id="cat-name"
                  className="input"
                  value={nameDraft}
                  onChange={(e) => setNameDraft(e.target.value)}
                  required
                  disabled={savingMeta}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="cat-desc">
                  Description
                </label>
                <input
                  id="cat-desc"
                  className="input"
                  value={descriptionDraft}
                  onChange={(e) => setDescriptionDraft(e.target.value)}
                  disabled={savingMeta}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={closeEdit}
                  disabled={savingMeta}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={savingMeta || !metaDirty}
                >
                  {savingMeta ? 'Saving…' : 'Save'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      {addSigOpen ? (
        <div className="dialog-root" role="presentation">
          <button
            type="button"
            className="dialog-backdrop"
            aria-label="Close dialog"
            disabled={addingSig}
            onClick={closeAddSig}
          />
          <div
            className="dialog-panel dialog-panel--form"
            role="dialog"
            aria-modal="true"
            aria-labelledby={addSigTitleId}
          >
            <h2 id={addSigTitleId} className="dialog-title">
              Add signature
            </h2>
            <p className="dialog-desc muted">
              Technical match atoms (same types as manual policy rules). Agents
              never see the category label — only the expanded list.
            </p>
            <form className="users-dialog-form" onSubmit={onAddSignature}>
              <div className="field">
                <label className="label" htmlFor="sig-type">
                  Type
                </label>
                <select
                  ref={addSigFirstRef}
                  id="sig-type"
                  className="input"
                  value={sigType}
                  onChange={(e) => setSigType(e.target.value)}
                  disabled={addingSig}
                >
                  <option value="process_name">process_name</option>
                  <option value="path">path</option>
                  <option value="hash_sha256">hash_sha256</option>
                  <option value="publisher">publisher</option>
                  <option value="product_name">product_name</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="sig-op">
                  Operator
                </label>
                <select
                  id="sig-op"
                  className="input"
                  value={sigOperator}
                  onChange={(e) => setSigOperator(e.target.value)}
                  disabled={addingSig}
                >
                  <option value="equals">equals</option>
                  <option value="prefix">prefix</option>
                  <option value="contains">contains</option>
                </select>
              </div>
              <div className="field">
                <label className="label" htmlFor="sig-value">
                  Value
                </label>
                <input
                  id="sig-value"
                  className="input"
                  value={sigValue}
                  onChange={(e) => setSigValue(e.target.value)}
                  placeholder="whatsapp.exe"
                  required
                  disabled={addingSig}
                />
              </div>
              <div className="dialog-actions">
                <button
                  type="button"
                  className="btn btn-ghost"
                  onClick={closeAddSig}
                  disabled={addingSig}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="btn btn-primary"
                  disabled={addingSig}
                  aria-busy={addingSig}
                >
                  {addingSig ? 'Adding…' : 'Add signature'}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={removeSig !== null}
        title="Delete signature?"
        description={
          removeSig
            ? `Remove ${removeSig.rule_type} ${removeSig.operator} ${removeSig.value}? Policies using this category will drop the match after the next publish/sync.`
            : ''
        }
        confirmLabel="Delete"
        cancelLabel="Cancel"
        danger
        busy={removingSig}
        onConfirm={onConfirmRemoveSig}
        onCancel={() => {
          if (!removingSig) setRemoveSig(null)
        }}
      />

      <ConfirmDialog
        open={confirmDelete}
        title="Delete category?"
        description={
          category
            ? `Delete ${category.name} (${category.code}) and all its signatures? This fails if a policy rule still references the category.`
            : ''
        }
        confirmLabel="Delete category"
        cancelLabel="Cancel"
        danger
        busy={deleting}
        onConfirm={onConfirmDeleteCategory}
        onCancel={() => {
          if (!deleting) setConfirmDelete(false)
        }}
      />
    </div>
  )
}
