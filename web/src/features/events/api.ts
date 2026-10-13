import { apiFetch } from '../../lib/api'
import type { EventListData } from './types'

export function listEvents() {
  return apiFetch<EventListData>('/v1/events')
}
