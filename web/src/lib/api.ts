import { getToken } from './auth-storage'

export type ApiError = { code: string; message: string }
export type Envelope<T> = { ok: boolean; data?: T; error?: ApiError }

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Content-Type', 'application/json')

  const token = getToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(path, { ...init, headers })
  const body = (await res.json()) as Envelope<T>
  
  if (!res.ok || !body.ok || body.data === undefined) {
    throw new Error(body.error?.message ?? `request failed (${res.status})`)
  }
  return body.data
}