import { apiFetch, type Envelope } from '../../lib/api'
import { getToken } from '../../lib/auth-storage'
import { endSession } from '../../lib/session'
import type { AuthSessionListData, LoginData, Me } from './types'

export function login(email: string, password: string) {
  return apiFetch<LoginData>('/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export function fetchMe() {
  return apiFetch<Me>('/v1/auth/me')
}

export function logout() {
  return apiFetchOk('/v1/auth/logout', {
    method: 'POST',
    body: '{}',
  })
}

export function changePassword(currentPassword: string, newPassword: string) {
  return apiFetchOk('/v1/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  })
}

export function listSessions() {
  return apiFetch<AuthSessionListData>('/v1/auth/sessions')
}

export function revokeSession(id: string) {
  return apiFetchOk(`/v1/auth/sessions/${encodeURIComponent(id)}/revoke`, {
    method: 'POST',
    body: '{}',
  })
}

export function revokeOtherSessions() {
  return apiFetchOk('/v1/auth/sessions/revoke-others', {
    method: 'POST',
    body: '{}',
  })
}

/** Like apiFetch, but allows empty/omitted `data` when ok (auth mutations). */
async function apiFetchOk(path: string, init: RequestInit = {}): Promise<void> {
  const headers = new Headers(init.headers)
  headers.set('Content-Type', 'application/json')
  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(path, { ...init, headers })
  if (res.status === 401 && token && !path.endsWith('/auth/login')) {
    await endSession({ reason: 'expired' })
    throw new Error('session expired')
  }

  const body = (await res.json()) as Envelope<unknown>
  if (!res.ok || !body.ok) {
    throw new Error(body.error?.message ?? `request failed (${res.status})`)
  }
}