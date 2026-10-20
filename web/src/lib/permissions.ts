import type { Me } from '../features/auth/types'

export function hasPermission(
  me: Me | null | undefined,
  code: string,
): boolean {
  return me?.permissions.includes(code) ?? false
}

export function hasAnyPermission(
  me: Me | null | undefined,
  codes: string[],
): boolean {
  if (!me) return false
  return codes.some((c) => me.permissions.includes(c))
}
