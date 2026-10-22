import { apiFetch } from '../../lib/api'
import { getToken } from '../../lib/auth-storage'
import type { AuditLogListData, AuditLogQuery } from './types'

function toSearchParams(query: AuditLogQuery = {}): URLSearchParams {
  const sp = new URLSearchParams()
  if (query.from) sp.set('from', query.from)
  if (query.to) sp.set('to', query.to)
  if (query.family) sp.set('family', query.family)
  if (query.actor) sp.set('actor', query.actor)
  if (query.q) sp.set('q', query.q)
  if (query.resource_type) sp.set('resource_type', query.resource_type)
  if (query.limit != null) sp.set('limit', String(query.limit))
  if (query.offset != null) sp.set('offset', String(query.offset))
  return sp
}

export function listAuditLogs(query: AuditLogQuery = {}) {
  const sp = toSearchParams(query)
  const qs = sp.toString()
  return apiFetch<AuditLogListData>(
    qs ? `/v1/audit-logs?${qs}` : '/v1/audit-logs',
  )
}

export async function exportAuditLogsCSV(query: AuditLogQuery = {}) {
  const sp = toSearchParams(query)
  // export ignores pagination; drop limit/offset for cleaner server query
  sp.delete('limit')
  sp.delete('offset')
  const qs = sp.toString()
  const path = qs
    ? `/v1/audit-logs/export?${qs}`
    : '/v1/audit-logs/export'

  const headers = new Headers()
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(path, { headers })
  if (!res.ok) {
    let message = `export failed (${res.status})`
    try {
      const body = (await res.json()) as {
        error?: { message?: string }
      }
      if (body.error?.message) message = body.error.message
    } catch {
      /* not JSON */
    }
    throw new Error(message)
  }

  const blob = await res.blob()
  const disposition = res.headers.get('Content-Disposition') || ''
  const match = /filename="([^"]+)"/.exec(disposition)
  const filename = match?.[1] ?? 'sailguard-audit.csv'

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
