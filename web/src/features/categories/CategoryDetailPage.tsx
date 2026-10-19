import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ConfirmDialog } from '../../components/ConfirmDialog'
import { toast } from '../../components/toast'
import { formatRelativeTime } from '../../lib/format'
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
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()

  const [category, setCategory] = useState<Category | null>(null)
  const [signatures, setSignatures] = useState<CategorySignature[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const [nameDraft, setNameDraft] = useState('')
  const [descriptionDraft, setDescriptionDraft] = useState('')
  const [savingMeta, setSavingMeta] = useState(false)

  const [sigType, setSigType] = useState('process_name')
  const [sigOperator, setSigOperator] = useState('equals')
  const [sigValue, setSigValue] = useState('')
  const [addingSig, setAddingSig] = useState(false)

  const [removeSig, setRemoveSig] = useState<CategorySignature | null>(null)
  const [removingSig, setRemovingSig] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  const load = useCallback(() => {
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
  }, [code])

  useEffect(() => {
    load()
  }, [load])

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
            Manage display name and technical signatures. Policy rules that
            reference this category expand to these signatures on publish/sync.
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
          <button
            type="button"
            className="btn btn-danger"
            onClick={() => setConfirmDelete(true)}
            disabled={!category || deleting}
          >
            Delete
          </button>
        </div>
      </header>

      {error ? (
        <p className="error" role="alert">
          {error}
        </p>
      ) : null}

      {loading && !category ? <p className="loading">Loading…</p> : null}

      {category ? (
        <section className="panel" aria-labelledby="category-meta-heading">
          <h2 id="category-meta-heading" className="panel-title">
            Details
          </h2>
          <form className="rule-form-grid" onSubmit={onSaveMeta}>
            <div className="field">
              <label className="label" htmlFor="cat-code-ro">
                Code
              </label>
              <input
                id="cat-code-ro"
                className="input mono"
                value={category.code}
                disabled
                readOnly
              />
            </div>
            <div className="field">
              <label className="label" htmlFor="cat-name">
                Name
              </label>
              <input
                id="cat-name"
                className="input"
                value={nameDraft}
                onChange={(e) => setNameDraft(e.target.value)}
                required
              />
            </div>
            <div className="field rule-form-value">
              <label className="label" htmlFor="cat-desc">
                Description
              </label>
              <input
                id="cat-desc"
                className="input"
                value={descriptionDraft}
                onChange={(e) => setDescriptionDraft(e.target.value)}
              />
            </div>
            <div className="rule-form-actions">
              <button
                type="submit"
                className="btn btn-primary"
                disabled={
                  savingMeta ||
                  (nameDraft.trim() === category.name &&
                    descriptionDraft.trim() === (category.description || ''))
                }
              >
                {savingMeta ? 'Saving…' : 'Save details'}
              </button>
            </div>
          </form>
        </section>
      ) : null}

      {category ? (
        <section className="panel" aria-labelledby="add-sig-heading">
          <h2 id="add-sig-heading" className="panel-title">
            Add signature
          </h2>
          <p className="muted panel-lead">
            Signatures are technical match atoms (same types as manual policy
            rules). Agents never see the category label — only the expanded
            list.
          </p>
          <form className="rule-form-grid" onSubmit={onAddSignature}>
            <div className="field">
              <label className="label" htmlFor="sig-type">
                Type
              </label>
              <select
                id="sig-type"
                className="input"
                value={sigType}
                onChange={(e) => setSigType(e.target.value)}
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
              >
                <option value="equals">equals</option>
                <option value="prefix">prefix</option>
                <option value="contains">contains</option>
              </select>
            </div>
            <div className="field rule-form-value">
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
              />
            </div>
            <div className="rule-form-actions">
              <button
                type="submit"
                className="btn btn-primary"
                disabled={addingSig}
              >
                {addingSig ? 'Adding…' : 'Add signature'}
              </button>
            </div>
          </form>
        </section>
      ) : null}

      <section className="list-panel" aria-labelledby="sig-list-heading">
        <div className="list-panel-header">
          <h2 id="sig-list-heading" className="panel-title">
            Signatures
          </h2>
          {signatures ? (
            <p className="muted list-range">
              {signatures.length} signature
              {signatures.length === 1 ? '' : 's'}
            </p>
          ) : null}
        </div>

        {signatures && signatures.length === 0 ? (
          <div className="empty-state" role="status">
            <p className="empty-title">No signatures yet</p>
            <p className="muted">
              Add process names or other matchers above so category policies can
              expand.
            </p>
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
                  <th />
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
                    <td>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={() => setRemoveSig(s)}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

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
