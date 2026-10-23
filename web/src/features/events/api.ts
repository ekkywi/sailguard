import { apiFetch } from '../../lib/api'
import { getToken } from '../../lib/auth-storage'
import { endSession } from '../../lib/session'
import type { EventListData, EventListQuery } from './types'

function toSearchParams(query: EventListQuery = {}): URLSearchParams {
  const sp = new URLSearchParams()
  if (query.from) sp.set('from', query.from)
  if (query.to) sp.set('to', query.to)
  if (query.event_type) sp.set('event_type', query.event_type)
  if (query.device_id) sp.set('device_id', query.device_id)
  if (query.q) sp.set('q', query.q)
  if (query.limit != null) sp.set('limit', String(query.limit))
  if (query.offset != null) sp.set('offset', String(query.offset))
  return sp
}

export function listEvents(query: EventListQuery = {}) {
  const sp = toSearchParams(query)
  const qs = sp.toString()
  return apiFetch<EventListData>(qs ? `/v1/events?${qs}` : '/v1/events')
}

export async function exportEventsCSV(query: EventListQuery = {}) {
  const sp = toSearchParams(query)
  sp.delete('limit')
  sp.delete('offset')
  const qs = sp.toString()
  const path = qs ? `/v1/events/export?${qs}` : '/v1/events/export'

  const headers = new Headers()
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const res = await fetch(path, { headers })
  if (res.status === 401 && token) {
    await endSession({ reason: 'expired' })
    throw new Error('session expired')
  }
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
  const filename = match?.[1] ?? 'sailguard-events.csv'

  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
