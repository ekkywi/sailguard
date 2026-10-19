import { apiFetch } from '../../lib/api'
import type {
  AddCategorySignatureInput,
  Category,
  CategoryListData,
  CategorySignature,
  CategorySignatureListData,
  CreateCategoryInput,
  UpdateCategoryInput,
} from './types'

export function listCategories() {
  return apiFetch<CategoryListData>('/v1/categories')
}

export function getCategory(code: string) {
  return apiFetch<Category>(`/v1/categories/${encodeURIComponent(code)}`)
}

export function createCategory(body: CreateCategoryInput) {
  return apiFetch<Category>('/v1/categories', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchCategory(code: string, body: UpdateCategoryInput) {
  return apiFetch<Category>(`/v1/categories/${encodeURIComponent(code)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function deleteCategory(code: string) {
  return apiFetch<{ deleted: boolean; code: string }>(
    `/v1/categories/${encodeURIComponent(code)}`,
    { method: 'DELETE' },
  )
}

export function listCategorySignatures(code: string) {
  return apiFetch<CategorySignatureListData>(
    `/v1/categories/${encodeURIComponent(code)}/signatures`,
  )
}

export function addCategorySignature(
  code: string,
  body: AddCategorySignatureInput,
) {
  return apiFetch<CategorySignature>(
    `/v1/categories/${encodeURIComponent(code)}/signatures`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    },
  )
}

export function deleteCategorySignature(code: string, signatureId: string) {
  return apiFetch<{
    deleted: boolean
    category_code: string
    signature_id: string
  }>(
    `/v1/categories/${encodeURIComponent(code)}/signatures/${signatureId}`,
    { method: 'DELETE' },
  )
}
