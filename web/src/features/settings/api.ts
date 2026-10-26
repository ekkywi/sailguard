import { apiFetch } from '../../lib/api'
import type {
  AlertChannel,
  AlertChannelListData,
  AlertDelivery,
  AlertDeliveryListData,
} from './types'

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

export function listAlertDeliveries(status?: string) {
  const q = status ? `?status=${encodeURIComponent(status)}&limit=50` : '?limit=50'
  return apiFetch<AlertDeliveryListData>(`/v1/alert-deliveries${q}`)
}

export function retryAlertDelivery(id: string) {
  return apiFetch<AlertDelivery>(`/v1/alert-deliveries/${id}/retry`, {
    method: 'POST',
    body: '{}',
  })
}
