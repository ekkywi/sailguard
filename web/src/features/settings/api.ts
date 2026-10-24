import { apiFetch } from '../../lib/api'
import type { AlertChannel, AlertChannelListData } from './types'

export function listAlertChannels() {
  return apiFetch<AlertChannelListData>('/v1/alert-channels')
}

export function patchAlertChannel(
  id: string,
  body: { name?: string; enabled?: boolean; url?: string },
) {
  return apiFetch<AlertChannel>(`/v1/alert-channels/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export function testAlertChannel(id: string) {
  return apiFetch<{ sent: boolean }>(`/v1/alert-channels/${id}/test`, {
    method: 'POST',
    body: '{}',
  })
}
