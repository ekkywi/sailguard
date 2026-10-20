import { apiFetch } from '../../lib/api'
import type {
  AdminUser,
  CreateUserInput,
  PatchUserInput,
  RoleListData,
  UserListData,
} from './types'

export function listUsers() {
  return apiFetch<UserListData>('/v1/users')
}

export function createUser(body: CreateUserInput) {
  return apiFetch<AdminUser>('/v1/users', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function patchUser(id: string, body: PatchUserInput) {
  return apiFetch<AdminUser>(`/v1/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function resetUserPassword(id: string, password: string) {
  return apiFetch<{ id: string; message: string }>(
    `/v1/users/${id}/reset-password`,
    {
      method: 'POST',
      body: JSON.stringify({ password }),
    },
  )
}

export function listRoles() {
  return apiFetch<RoleListData>('/v1/roles')
}
