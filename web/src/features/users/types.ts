export type AdminUser = {
  id: string
  email: string
  name: string
  auth_provider: string
  is_active: boolean
  roles: string[]
  last_login_at: string | null
  created_at: string
  updated_at: string
}

export type UserListData = {
  items: AdminUser[]
}

export type Role = {
  code: string
  name: string
  description: string
}

export type RoleListData = {
  items: Role[]
}

export type CreateUserInput = {
  email: string
  name: string
  password: string
  roles: string[]
}

export type PatchUserInput = {
  name?: string
  is_active?: boolean
  roles?: string[]
}
