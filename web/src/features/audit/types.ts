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

export type AuditLogListData = {
  items: AuditLog[]
}
