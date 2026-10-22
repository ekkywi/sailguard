export type AuditLog = {
  id: string
  actor_user_id: string | null
  actor_email: string
  action: string
  resource_type: string
  resource_id: string
  summary: string
  metadata: Record<string, unknown> | null
  created_at: string
}

export type AuditFamily =
  | ''
  | 'policy'
  | 'category'
  | 'user'
  | 'device'
  | 'group'
  | 'auth'
  | 'package'
  | 'override'
  | 'enrollment'

export type AuditLogQuery = {
  from?: string
  to?: string
  family?: AuditFamily
  actor?: string
  q?: string
  resource_type?: string
  limit?: number
  offset?: number
}

export type AuditLogListData = {
  items: AuditLog[]
  total: number
  limit: number
  offset: number
}
