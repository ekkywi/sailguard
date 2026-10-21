import { apiFetch } from '../../lib/api'
import type { AuditLogListData } from './types'

export function listAuditLogs() {
  return apiFetch<AuditLogListData>('/v1/audit-logs')
}
