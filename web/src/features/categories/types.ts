export type Category = {
  code: string
  name: string
  description: string
  created_at?: string
  updated_at?: string
}

export type CategoryListData = {
  items: Category[]
}

export type CreateCategoryInput = {
  code: string
  name: string
  description?: string
}

export type UpdateCategoryInput = {
  name: string
  description?: string
}

export type CategorySignature = {
  id: string
  category_code: string
  rule_type: string
  operator: string
  value: string
  created_at: string
}

export type CategorySignatureListData = {
  items: CategorySignature[]
}

export type AddCategorySignatureInput = {
  rule_type: string
  operator?: string
  value: string
}
